// internal/service/instance_service.go
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/workflow-engine/v2/internal/api/websocket"
	"github.com/workflow-engine/v2/internal/core/bpmn"
	"github.com/workflow-engine/v2/internal/core/executor"
	"github.com/workflow-engine/v2/internal/core/statemachine"
	"github.com/workflow-engine/v2/internal/integration/nats"
	"github.com/workflow-engine/v2/internal/integration/postgres"
	"go.uber.org/zap"
)

type InstanceService struct {
	processRepo   *postgres.PostgresProcessRepository
	instanceRepo  *postgres.PostgresInstanceRepository
	eventRepo     *postgres.PostgresEventRepository
	executor      *executor.DefaultExecutor
	natsPublisher *nats.Publisher
	wsHub         *websocket.Hub
	logger        *zap.Logger
}

func NewInstanceService(
	procRepo *postgres.PostgresProcessRepository,
	instRepo *postgres.PostgresInstanceRepository,
	evtRepo *postgres.PostgresEventRepository,
	exec *executor.DefaultExecutor,
	natsPub *nats.Publisher,
	wsHub *websocket.Hub,
	logger *zap.Logger,
) *InstanceService {
	return &InstanceService{
		processRepo:   procRepo,
		instanceRepo:  instRepo,
		eventRepo:     evtRepo,
		executor:      exec,
		natsPublisher: natsPub,
		wsHub:         wsHub,
		logger:        logger,
	}
}

// StartInstance запускает новый инстанс процесса
func (s *InstanceService) StartInstance(
	ctx context.Context,
	processID string,
	variables map[string]interface{},
	businessKey string,
	startedBy string,
) (*statemachine.ProcessInstance, error) {

	// 1. Загружаем процесс по ключу
	process, _, err := s.processRepo.GetProcessByKey(ctx, processID)
	if err != nil {
		return nil, fmt.Errorf("get process: %w", err)
	}

	// 2. Парсим и валидируем
	graph, err := bpmn.BuildGraph(process)
	if err != nil {
		return nil, fmt.Errorf("build graph: %w", err)
	}

	startNode, err := graph.GetStartNode()
	if err != nil {
		return nil, fmt.Errorf("get start node: %w", err)
	}

	// 3. Создаем инстанс
	now := time.Now()
	instance := &statemachine.ProcessInstance{
		ID:         uuid.New().String(),
		ProcessKey: processID,
		Status:     statemachine.ProcessInstanceStatusRunning,
		Variables:  variables,
		StartedAt:  &now,
	}

	if err := s.instanceRepo.CreateInstance(ctx, instance); err != nil {
		return nil, fmt.Errorf("create instance: %w", err)
	}

	// 4. Запускаем execution в горутине
	go s.executeInstance(ctx, instance.ID, graph, startNode.GetID(), variables)

	// 5. WebSocket уведомление
	s.wsHub.BroadcastToInstance(instance.ID, map[string]interface{}{
		"type":   "instance_started",
		"status": instance.Status,
		"node":   startNode.GetID(),
	})

	return instance, nil
}

// executeInstance выполняет процесс до завершения или ожидания
func (s *InstanceService) executeInstance(
	ctx context.Context,
	instanceID string,
	graph *bpmn.Graph,
	startNodeID string,
	variables map[string]interface{},
) {
	// Добавляем instanceID в контекст для использования в executor
	ctx = context.WithValue(ctx, executor.ContextKeyInstanceID, instanceID)

	currentNodeID := startNodeID
	currentVars := variables

	for {
		// Получаем текущий узел
		node, exists := graph.GetElementByID(currentNodeID)
		if !exists {
			s.logger.Error("Node not found", zap.String("node_id", currentNodeID))
			s.updateInstanceStatus(ctx, instanceID, "error")
			return
		}

		// Выполняем узел
		result, err := s.executor.ExecuteNode(ctx, graph, node, currentVars)
		if err != nil {
			s.logger.Error("Execution failed",
				zap.String("instance_id", instanceID),
				zap.String("node_id", currentNodeID),
				zap.Error(err),
			)
			s.updateInstanceStatus(ctx, instanceID, "error")
			return
		}

		// Обновляем переменные
		currentVars = result.Variables

		// Проверяем, нужно ли ждать
		if result.Await {
			// Сохраняем состояние ожидания и выходим
			s.updateInstanceStatus(ctx, instanceID, "waiting")
			s.wsHub.BroadcastToInstance(instanceID, map[string]interface{}{
				"type":      "waiting",
				"node_id":   currentNodeID,
				"wait_type": result.AwaitType,
			})
			return
		}

		// Проверяем, завершен ли процесс
		if result.NextNodeID == "" {
			// Конец процесса
			s.completeInstance(ctx, instanceID, currentVars)
			return
		}

		currentNodeID = result.NextNodeID
	}
}

// HandleServiceResponse обрабатывает ответ от микросервиса
func (s *InstanceService) HandleServiceResponse(
	ctx context.Context,
	instanceID string,
	variables map[string]interface{},
) error {

	// 1. Загружаем инстанс
	instance, err := s.instanceRepo.GetInstanceByID(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("get instance: %w", err)
	}

	if instance.Status != statemachine.ProcessInstanceStatusRunning {
		return fmt.Errorf("instance not in running state: %s", instance.Status)
	}

	// 2. Загружаем процесс
	process, _, err := s.processRepo.GetProcessByID(ctx, instance.ProcessKey)
	if err != nil {
		return fmt.Errorf("get process: %w", err)
	}

	graph, err := bpmn.BuildGraph(process)
	if err != nil {
		return fmt.Errorf("build graph: %w", err)
	}

	// 3. Обновляем переменные
	mergedVars := mergeVariables(instance.Variables, variables)
	instance.Variables = mergedVars
	instance.LastActivityAt = time.Now()
	if err := s.instanceRepo.UpdateInstance(ctx, instance); err != nil {
		return fmt.Errorf("update instance: %w", err)
	}

	// WebSocket уведомление
	s.wsHub.BroadcastToInstance(instance.ID, map[string]interface{}{
		"type":      "service_response",
		"success":   true,
		"variables": variables,
	})

	// 4. Продолжаем выполнение с переданным контекстом
	go s.executeInstance(ctx, instance.ID, graph, "", mergedVars)

	return nil
}

func (s *InstanceService) updateInstanceStatus(ctx context.Context, id, status string) {
	instance, err := s.instanceRepo.GetInstanceByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get instance", zap.Error(err))
		return
	}

	instance.Status = statemachine.ProcessInstanceStatus(status)
	if err := s.instanceRepo.UpdateInstance(ctx, instance); err != nil {
		s.logger.Error("Failed to update status", zap.Error(err))
	}
}

func (s *InstanceService) completeInstance(ctx context.Context, id string, vars map[string]interface{}) {
	instance, err := s.instanceRepo.GetInstanceByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get instance", zap.Error(err))
		return
	}

	now := time.Now()
	instance.Status = statemachine.ProcessInstanceStatusCompleted
	instance.CompletedAt = &now
	instance.Variables = vars

	if err := s.instanceRepo.UpdateInstance(ctx, instance); err != nil {
		s.logger.Error("Failed to complete instance", zap.Error(err))
	}

	s.wsHub.BroadcastToInstance(id, map[string]interface{}{
		"type":   "completed",
		"status": "completed",
	})
}

func mergeVariables(existing, newVars map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range existing {
		result[k] = v
	}
	for k, v := range newVars {
		result[k] = v
	}
	return result
}

// InstanceFilter filters instances for listing
type InstanceFilter struct {
	ProcessKey string
	Status     string
	Limit      int
}

// InstanceInfo represents instance information for API responses
type InstanceInfo struct {
	ID          string                 `json:"id"`
	ProcessKey  string                 `json:"process_key"`
	Status      string                 `json:"status"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// ListInstances returns list of instances with filters
func (s *InstanceService) ListInstances(ctx context.Context, filter InstanceFilter) ([]*InstanceInfo, error) {
	postgresFilter := postgres.InstanceFilter{
		ProcessKey: filter.ProcessKey,
		Status:     filter.Status,
		Limit:      filter.Limit,
	}

	instances, err := s.instanceRepo.ListInstances(ctx, postgresFilter)
	if err != nil {
		return nil, fmt.Errorf("list instances: %w", err)
	}

	result := make([]*InstanceInfo, len(instances))
	for i, inst := range instances {
		result[i] = &InstanceInfo{
			ID:          inst.ID,
			ProcessKey:  inst.ProcessKey,
			Status:      string(inst.Status),
			Variables:   inst.Variables,
			StartedAt:   inst.StartedAt,
			CompletedAt: inst.CompletedAt,
		}
	}

	return result, nil
}

// GetInstance returns instance by ID
func (s *InstanceService) GetInstance(ctx context.Context, id string) (*InstanceInfo, error) {
	instance, err := s.instanceRepo.GetInstanceByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get instance: %w", err)
	}

	return &InstanceInfo{
		ID:          instance.ID,
		ProcessKey:  instance.ProcessKey,
		Status:      string(instance.Status),
		Variables:   instance.Variables,
		StartedAt:   instance.StartedAt,
		CompletedAt: instance.CompletedAt,
	}, nil
}

// SuspendInstance suspends a running instance
func (s *InstanceService) SuspendInstance(ctx context.Context, id string) error {
	instance, err := s.instanceRepo.GetInstanceByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get instance: %w", err)
	}

	if instance.Status != statemachine.ProcessInstanceStatusRunning {
		return fmt.Errorf("instance is not running")
	}

	instance.Status = statemachine.ProcessInstanceStatusSuspended
	if err := s.instanceRepo.UpdateInstance(ctx, instance); err != nil {
		return fmt.Errorf("update instance: %w", err)
	}

	s.wsHub.BroadcastToInstance(id, map[string]interface{}{
		"type":   "suspended",
		"status": "suspended",
	})

	return nil
}

// ResumeInstance resumes a suspended instance
func (s *InstanceService) ResumeInstance(ctx context.Context, id string) error {
	instance, err := s.instanceRepo.GetInstanceByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get instance: %w", err)
	}

	if instance.Status != statemachine.ProcessInstanceStatusSuspended {
		return fmt.Errorf("instance is not suspended")
	}

	instance.Status = statemachine.ProcessInstanceStatusRunning
	if err := s.instanceRepo.UpdateInstance(ctx, instance); err != nil {
		return fmt.Errorf("update instance: %w", err)
	}

	s.wsHub.BroadcastToInstance(id, map[string]interface{}{
		"type":   "resumed",
		"status": "running",
	})

	return nil
}

// TerminateInstance terminates an instance
func (s *InstanceService) TerminateInstance(ctx context.Context, id string) error {
	instance, err := s.instanceRepo.GetInstanceByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get instance: %w", err)
	}

	if instance.Status == statemachine.ProcessInstanceStatusCompleted ||
		instance.Status == statemachine.ProcessInstanceStatusTerminated {
		return fmt.Errorf("instance already terminated or completed")
	}

	now := time.Now()
	instance.Status = statemachine.ProcessInstanceStatusTerminated
	instance.CompletedAt = &now

	if err := s.instanceRepo.UpdateInstance(ctx, instance); err != nil {
		return fmt.Errorf("update instance: %w", err)
	}

	s.wsHub.BroadcastToInstance(id, map[string]interface{}{
		"type":   "terminated",
		"status": "terminated",
	})

	return nil
}

// GetVariables returns instance variables
func (s *InstanceService) GetVariables(ctx context.Context, id string) (map[string]interface{}, error) {
	instance, err := s.instanceRepo.GetInstanceByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get instance: %w", err)
	}
	return instance.Variables, nil
}

// UpdateVariables updates instance variables
func (s *InstanceService) UpdateVariables(ctx context.Context, id string, vars map[string]interface{}) (map[string]interface{}, error) {
	instance, err := s.instanceRepo.GetInstanceByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get instance: %w", err)
	}

	// Merge variables
	for k, v := range vars {
		instance.Variables[k] = v
	}

	if err := s.instanceRepo.UpdateInstance(ctx, instance); err != nil {
		return nil, fmt.Errorf("update instance: %w", err)
	}

	s.wsHub.BroadcastToInstance(id, map[string]interface{}{
		"type":      "variables_updated",
		"variables": instance.Variables,
	})

	return instance.Variables, nil
}

// CompleteUserTask completes a user task
func (s *InstanceService) CompleteUserTask(ctx context.Context, instanceID, taskID string, variables map[string]interface{}, userID string) error {
	_, err := s.instanceRepo.GetInstanceByID(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("get instance: %w", err)
	}

	// WebSocket уведомление
	s.wsHub.BroadcastToInstance(instanceID, map[string]interface{}{
		"type":      "task_completed",
		"task_id":   taskID,
		"user_id":   userID,
		"variables": variables,
	})

	return nil
}

// GetTaskForm returns the form for a user task
func (s *InstanceService) GetTaskForm(ctx context.Context, instanceID, taskID string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"task_id":     taskID,
		"instance_id": instanceID,
		"form_schema": map[string]interface{}{},
	}, nil
}

// GetTasks returns user tasks based on filter
func (s *InstanceService) GetTasks(ctx context.Context, filter postgres.TaskFilter) ([]postgres.UserTask, error) {
	return s.instanceRepo.GetTasks(ctx, filter)
}
