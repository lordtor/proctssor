// internal/core/executor/delegate_expression.go
package executor

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/workflow-engine/v2/internal/core/statemachine"
	"github.com/workflow-engine/v2/internal/integration/nats"
	"github.com/workflow-engine/v2/internal/integration/registry"
	"go.uber.org/zap"
)

// DelegateExpressionParser разбирает ${serviceName.actionName}
type DelegateExpressionParser struct {
	pattern *regexp.Regexp
}

func NewDelegateExpressionParser() *DelegateExpressionParser {
	return &DelegateExpressionParser{
		pattern: regexp.MustCompile(`^\$\{(\w+)\.(\w+)\}$`),
	}
}

func (p *DelegateExpressionParser) Parse(expression string) (service, action string, err error) {
	matches := p.pattern.FindStringSubmatch(strings.TrimSpace(expression))
	if len(matches) != 3 {
		return "", "", fmt.Errorf("invalid delegate expression: %s, expected ${service.action}", expression)
	}
	return matches[1], matches[2], nil
}

// executeDelegateExpression публикует команду в NATS и возвращает статус ожидания
func (e *DefaultExecutor) executeDelegateExpression(
	ctx context.Context,
	expression string,
	variables map[string]interface{},
	instanceID, nodeID, nodeName string,
) (*statemachine.ExecutionResult, error) {

	parser := NewDelegateExpressionParser()
	serviceName, action, err := parser.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("parse delegate expression: %w", err)
	}

	// Проверяем, что registryCache доступен
	if e.registryCache == nil {
		return nil, fmt.Errorf("registry cache not available")
	}

	// Проверяем существование сервиса
	svcInterface, found := e.registryCache.Get(ctx, serviceName)
	if !found {
		return nil, fmt.Errorf("service not found in registry: %s", serviceName)
	}

	// Приводим тип к registry.Service
	svc, ok := svcInterface.(*registry.Service)
	if !ok {
		return nil, fmt.Errorf("invalid service type in cache")
	}

	// Проверяем наличие action
	actionFound := false
	if svc.Metadata != nil {
		_, actionFound = svc.Metadata[action]
	}
	if !actionFound {
		return nil, fmt.Errorf("action %s not found in service %s", action, serviceName)
	}

	// Создаем команду
	cmd := &nats.WorkflowCommand{
		CommandID:      "",
		CommandType:    nats.CommandTypeServiceTask,
		InstanceID:     instanceID,
		TokenID:        "",
		NodeID:         nodeID,
		ServiceName:    serviceName,
		Operation:      action,
		InputVariables: copyVariables(variables),
		CreatedAt:      time.Now(),
		MaxRetries:     3,
	}

	// Проверяем, что natsPublisher доступен
	if e.natsPublisher == nil {
		return nil, fmt.Errorf("NATS publisher not available")
	}

	// Публикуем в NATS с retry
	maxRetries := 3
	retryDelay := 100 * time.Millisecond

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := e.natsPublisher.PublishCommand(ctx, cmd); err != nil {
			if attempt < maxRetries {
				e.logger.Warn("Failed to publish command, retrying",
					zap.Error(err),
					zap.Int("attempt", attempt),
					zap.Int("maxRetries", maxRetries),
				)
				time.Sleep(retryDelay)
				retryDelay *= 2 // exponential backoff
				continue
			}
			return nil, fmt.Errorf("publish to NATS after %d attempts: %w", maxRetries, err)
		}
		break // success
	}

	e.logger.Info("Published command to NATS",
		zap.String("instance_id", instanceID),
		zap.String("service", serviceName),
		zap.String("action", action),
	)

	// Возвращаем статус ожидания
	return &statemachine.ExecutionResult{
		Variables: variables,
		Await:     true,
		AwaitType: "external_service",
	}, nil
}

func copyVariables(vars map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(vars))
	for k, v := range vars {
		result[k] = v
	}
	return result
}
