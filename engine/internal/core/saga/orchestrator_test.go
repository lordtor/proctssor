package saga

import (
	"context"
	"fmt"
	"testing"

	"github.com/workflow-engine/v2/internal/core/bpmn"
	"github.com/workflow-engine/v2/internal/core/statemachine"
	"go.uber.org/zap"
)

// MockExecutor is a mock implementation of ExecutorInterface for testing
type MockExecutor struct {
	ExecuteNodeFunc func(ctx context.Context, graph *bpmn.Graph, node bpmn.FlowElement, variables map[string]interface{}) (*statemachine.ExecutionResult, error)
}

func (m *MockExecutor) ExecuteNode(ctx context.Context, graph *bpmn.Graph, node bpmn.FlowElement, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
	if m.ExecuteNodeFunc != nil {
		return m.ExecuteNodeFunc(ctx, graph, node, variables)
	}
	return &statemachine.ExecutionResult{Variables: variables}, nil
}

// MockCompensationHandler is a mock implementation of CompensationHandler for testing
type MockCompensationHandler struct {
	ExecuteCompensationFunc func(ctx context.Context, node bpmn.FlowElement, variables map[string]interface{}) (*statemachine.ExecutionResult, error)
}

func (m *MockCompensationHandler) ExecuteCompensation(ctx context.Context, node bpmn.FlowElement, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
	if m.ExecuteCompensationFunc != nil {
		return m.ExecuteCompensationFunc(ctx, node, variables)
	}
	return &statemachine.ExecutionResult{Variables: variables}, nil
}

// MockSagaRepository is a mock implementation of SagaRepository for testing
type MockSagaRepository struct {
	SavedSagas    []*Saga
	UpdatedSagas  []*Saga
	DeletedIDs    []string
	GetFunc       func(ctx context.Context, id string) (*Saga, error)
	GetByInstFunc func(ctx context.Context, instanceID string) (*Saga, error)
}

func (m *MockSagaRepository) Save(ctx context.Context, saga *Saga) error {
	m.SavedSagas = append(m.SavedSagas, saga)
	return nil
}

func (m *MockSagaRepository) Get(ctx context.Context, id string) (*Saga, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockSagaRepository) GetByInstanceID(ctx context.Context, instanceID string) (*Saga, error) {
	if m.GetByInstFunc != nil {
		return m.GetByInstFunc(ctx, instanceID)
	}
	return nil, nil
}

func (m *MockSagaRepository) Update(ctx context.Context, saga *Saga) error {
	m.UpdatedSagas = append(m.UpdatedSagas, saga)
	return nil
}

func (m *MockSagaRepository) Delete(ctx context.Context, id string) error {
	m.DeletedIDs = append(m.DeletedIDs, id)
	return nil
}

func TestNewSaga(t *testing.T) {
	saga := NewSaga("instance-123", "test-process", map[string]interface{}{"key": "value"})

	if saga == nil {
		t.Fatal("Expected saga to be not nil")
	}

	if saga.InstanceID != "instance-123" {
		t.Errorf("Expected InstanceID to be 'instance-123', got '%s'", saga.InstanceID)
	}

	if saga.ProcessKey != "test-process" {
		t.Errorf("Expected ProcessKey to be 'test-process', got '%s'", saga.ProcessKey)
	}

	if saga.Status != SagaStatusPending {
		t.Errorf("Expected Status to be SagaStatusPending, got '%s'", saga.Status)
	}

	if saga.Variables == nil {
		t.Error("Expected Variables to be not nil")
	}
}

func TestSaga_AddStep(t *testing.T) {
	saga := NewSaga("instance-123", "test-process", nil)
	saga.AddStep("task1", "Task 1", "undo_task1", "Undo Task 1", map[string]interface{}{"input": "value"})

	if len(saga.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(saga.Steps))
	}

	step := saga.Steps[0]
	if step.NodeID != "task1" {
		t.Errorf("Expected NodeID to be 'task1', got '%s'", step.NodeID)
	}

	if step.NodeName != "Task 1" {
		t.Errorf("Expected NodeName to be 'Task 1', got '%s'", step.NodeName)
	}

	if step.CompensateNodeID != "undo_task1" {
		t.Errorf("Expected CompensateNodeID to be 'undo_task1', got '%s'", step.CompensateNodeID)
	}

	if step.Status != SagaStepStatusPending {
		t.Errorf("Expected Status to be SagaStepStatusPending, got '%s'", step.Status)
	}
}

func TestSaga_GetCompletedSteps(t *testing.T) {
	saga := NewSaga("instance-123", "test-process", nil)

	// Add some steps
	saga.AddStep("task1", "Task 1", "undo_task1", "", nil)
	saga.AddStep("task2", "Task 2", "undo_task2", "", nil)
	saga.AddStep("task3", "Task 3", "undo_task3", "", nil)

	// Mark first two as completed
	saga.Steps[0].Status = SagaStepStatusCompleted
	saga.Steps[1].Status = SagaStepStatusCompleted
	saga.Steps[2].Status = SagaStepStatusPending

	completed := saga.GetCompletedSteps()

	if len(completed) != 2 {
		t.Errorf("Expected 2 completed steps, got %d", len(completed))
	}

	// Check that they are in reverse order
	if len(completed) > 0 && completed[0].NodeID != "task2" {
		t.Errorf("Expected first completed step to be 'task2', got '%s'", completed[0].NodeID)
	}

	if len(completed) > 1 && completed[1].NodeID != "task1" {
		t.Errorf("Expected second completed step to be 'task1', got '%s'", completed[1].NodeID)
	}
}

func TestSaga_HasCompensation(t *testing.T) {
	saga := NewSaga("instance-123", "test-process", nil)

	// No steps yet
	if saga.HasCompensation() {
		t.Error("Expected no compensation capability with no steps")
	}

	// Add step without compensation
	saga.AddStep("task1", "Task 1", "", "", nil)
	saga.Steps[0].Status = SagaStepStatusCompleted

	if saga.HasCompensation() {
		t.Error("Expected no compensation capability without compensation node")
	}

	// Add step with compensation
	saga.AddStep("task2", "Task 2", "undo_task2", "", nil)
	saga.Steps[1].Status = SagaStepStatusCompleted

	if !saga.HasCompensation() {
		t.Error("Expected compensation capability with compensation node")
	}
}

func TestSagaOrchestrator_StartSaga(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockRepo := &MockSagaRepository{}

	orchestrator := NewSagaOrchestrator(nil, nil, logger, mockRepo)

	saga, err := orchestrator.StartSaga(context.Background(), "instance-123", "test-process", map[string]interface{}{"key": "value"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if saga == nil {
		t.Fatal("Expected saga to be not nil")
	}

	if saga.InstanceID != "instance-123" {
		t.Errorf("Expected InstanceID to be 'instance-123', got '%s'", saga.InstanceID)
	}

	if saga.Status != SagaStatusRunning {
		t.Errorf("Expected Status to be SagaStatusRunning, got '%s'", saga.Status)
	}

	if len(mockRepo.SavedSagas) != 1 {
		t.Errorf("Expected 1 saved saga, got %d", len(mockRepo.SavedSagas))
	}
}

func TestSagaOrchestrator_ExecuteStep_Success(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockRepo := &MockSagaRepository{}
	mockExecutor := &MockExecutor{
		ExecuteNodeFunc: func(ctx context.Context, graph *bpmn.Graph, node bpmn.FlowElement, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
			return &statemachine.ExecutionResult{
				Variables: map[string]interface{}{"result": "success"},
			}, nil
		},
	}

	orchestrator := NewSagaOrchestrator(mockExecutor, nil, logger, mockRepo)

	// Start saga
	saga, _ := orchestrator.StartSaga(context.Background(), "instance-123", "test-process", nil)
	orchestrator.AddStep(context.Background(), saga, "task1", "Task 1", "undo_task1", "Undo Task 1", map[string]interface{}{"input": "value"})

	// Create graph with node
	graph := &bpmn.Graph{
		Nodes: map[string]bpmn.FlowElement{
			"task1": &bpmn.ServiceTask{
				Task: bpmn.Task{
					BaseElement: bpmn.BaseElement{
						ID:   "task1",
						Name: "Task 1",
					},
				},
			},
		},
	}

	// Execute step
	step, err := orchestrator.ExecuteStep(context.Background(), saga, graph, 0)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if step.Status != SagaStepStatusCompleted {
		t.Errorf("Expected step status to be SagaStepStatusCompleted, got '%s'", step.Status)
	}

	if step.OutputVariables == nil {
		t.Error("Expected output variables to be not nil")
	}
}

func TestSagaOrchestrator_ExecuteStep_Error(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockRepo := &MockSagaRepository{}
	mockExecutor := &MockExecutor{
		ExecuteNodeFunc: func(ctx context.Context, graph *bpmn.Graph, node bpmn.FlowElement, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
			return nil, fmt.Errorf("execution failed")
		},
	}
	mockCompHandler := &MockCompensationHandler{}

	orchestrator := NewSagaOrchestrator(mockExecutor, mockCompHandler, logger, mockRepo)

	// Start saga
	saga, _ := orchestrator.StartSaga(context.Background(), "instance-123", "test-process", nil)
	orchestrator.AddStep(context.Background(), saga, "task1", "Task 1", "undo_task1", "Undo Task 1", map[string]interface{}{"input": "value"})

	// Add a second step that is completed
	orchestrator.AddStep(context.Background(), saga, "task2", "Task 2", "undo_task2", "Undo Task 2", map[string]interface{}{"input": "value"})
	saga.Steps[0].Status = SagaStepStatusCompleted

	// Create graph with node
	graph := &bpmn.Graph{
		Nodes: map[string]bpmn.FlowElement{
			"task1": &bpmn.ServiceTask{
				Task: bpmn.Task{
					BaseElement: bpmn.BaseElement{
						ID:   "task1",
						Name: "Task 1",
					},
				},
			},
		},
	}

	// Execute step that will fail
	_, err := orchestrator.ExecuteStep(context.Background(), saga, graph, 1)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Check that saga status is updated
	if saga.Status != SagaStatusCompensating && saga.Status != SagaStatusFailed {
		t.Errorf("Expected saga status to be SagaStatusCompensating or SagaStatusFailed, got '%s'", saga.Status)
	}
}

func TestSagaOrchestrator_Compensate(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockRepo := &MockSagaRepository{}
	mockCompHandler := &MockCompensationHandler{
		ExecuteCompensationFunc: func(ctx context.Context, node bpmn.FlowElement, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
			return &statemachine.ExecutionResult{
				Variables: map[string]interface{}{"compensated": true},
			}, nil
		},
	}

	orchestrator := NewSagaOrchestrator(nil, mockCompHandler, logger, mockRepo)

	// Start saga
	saga, _ := orchestrator.StartSaga(context.Background(), "instance-123", "test-process", nil)

	// Add and complete steps
	saga.AddStep("task1", "Task 1", "undo_task1", "Undo Task 1", map[string]interface{}{"input": "value1"})
	saga.AddStep("task2", "Task 2", "undo_task2", "Undo Task 2", map[string]interface{}{"input": "value2"})
	saga.Steps[0].Status = SagaStepStatusCompleted
	saga.Steps[1].Status = SagaStepStatusCompleted

	// Compensate
	err := orchestrator.Compensate(context.Background(), saga)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check saga status
	if saga.Status != SagaStatusCompensated {
		t.Errorf("Expected saga status to be SagaStatusCompensated, got '%s'", saga.Status)
	}

	// Check step statuses
	for _, step := range saga.Steps {
		if step.Status != SagaStepStatusCompensated {
			t.Errorf("Expected step status to be SagaStepStatusCompensated, got '%s'", step.Status)
		}
	}
}

func TestSagaOrchestrator_CompleteSaga(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockRepo := &MockSagaRepository{}

	orchestrator := NewSagaOrchestrator(nil, nil, logger, mockRepo)

	saga := NewSaga("instance-123", "test-process", nil)

	err := orchestrator.CompleteSaga(context.Background(), saga)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if saga.Status != SagaStatusCompleted {
		t.Errorf("Expected saga status to be SagaStatusCompleted, got '%s'", saga.Status)
	}

	if saga.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestServiceTask_GetCompensateNodeID(t *testing.T) {
	task := &bpmn.ServiceTask{
		Task: bpmn.Task{
			BaseElement: bpmn.BaseElement{
				ID:   "task1",
				Name: "Task 1",
			},
		},
		CompensateNodeID: "undo_task1",
	}

	if task.GetCompensateNodeID() != "undo_task1" {
		t.Errorf("Expected compensate node ID to be 'undo_task1', got '%s'", task.GetCompensateNodeID())
	}
}

func TestServiceTask_IsCompensatable(t *testing.T) {
	// Test with compensation node
	task1 := &bpmn.ServiceTask{
		CompensateNodeID: "undo_task1",
	}
	if !task1.IsCompensatable() {
		t.Error("Expected task with compensation node to be compensatable")
	}

	// Test with isForCompensation flag
	task2 := &bpmn.ServiceTask{
		IsForCompensation: true,
	}
	if !task2.IsCompensatable() {
		t.Error("Expected task with isForCompensation to be compensatable")
	}

	// Test without compensation
	task3 := &bpmn.ServiceTask{}
	if task3.IsCompensatable() {
		t.Error("Expected task without compensation to not be compensatable")
	}
}
