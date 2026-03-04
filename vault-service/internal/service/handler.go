package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"vault-service/internal/nats"
	"vault-service/internal/vault"
)

// Handler handles Vault service commands
type Handler struct {
	storage      vault.Storage
	secretPrefix string
}

// NewHandler creates a new Vault service handler
func NewHandler(storage vault.Storage, secretPrefix string) *Handler {
	return &Handler{
		storage:      storage,
		secretPrefix: secretPrefix,
	}
}

// HandleCommand processes a workflow command and returns a response
func (h *Handler) HandleCommand(ctx context.Context, cmd *nats.WorkflowCommand) (*nats.ServiceResponse, error) {
	resp := &nats.ServiceResponse{
		CommandID:   cmd.CommandID,
		InstanceID:  cmd.InstanceID,
		TokenID:     cmd.TokenID,
		NodeID:      cmd.NodeID,
		ProcessedAt: time.Now(),
	}

	// Process based on operation
	switch cmd.Operation {
	case "get_secret":
		return h.handleGetSecret(ctx, cmd, resp)
	case "set_secret":
		return h.handleSetSecret(ctx, cmd, resp)
	case "delete_secret":
		return h.handleDeleteSecret(ctx, cmd, resp)
	case "list_secrets":
		return h.handleListSecrets(ctx, cmd, resp)
	case "health_check":
		return h.handleHealthCheck(ctx, cmd, resp)
	default:
		return h.handleUnknownOperation(ctx, cmd, resp)
	}
}

func (h *Handler) handleGetSecret(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	key, ok := cmd.InputVariables["key"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "key is required"
		return resp, nil
	}

	// Apply secret prefix
	fullKey := h.getFullKey(key)

	secret, err := h.storage.Get(fullKey)
	if err != nil {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "secret_not_found"
		resp.ErrorMessage = err.Error()
		return resp, nil
	}

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"key":        secret.Key,
		"value":      secret.Value,
		"metadata":   secret.Metadata,
		"created_at": secret.CreatedAt.Unix(),
		"updated_at": secret.UpdatedAt.Unix(),
	}

	log.Printf("Retrieved secret: %s", key)
	return resp, nil
}

func (h *Handler) handleSetSecret(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	key, ok := cmd.InputVariables["key"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "key is required"
		return resp, nil
	}

	value, ok := cmd.InputVariables["value"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "value is required"
		return resp, nil
	}

	// Extract metadata if provided
	metadata := make(map[string]string)
	if meta, ok := cmd.InputVariables["metadata"].(map[string]interface{}); ok {
		for k, v := range meta {
			if s, ok := v.(string); ok {
				metadata[k] = s
			}
		}
	}

	// Apply secret prefix
	fullKey := h.getFullKey(key)

	if err := h.storage.Set(fullKey, value, metadata); err != nil {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "storage_error"
		resp.ErrorMessage = err.Error()
		return resp, nil
	}

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"key":        key,
		"secret_set": true,
		"timestamp":  time.Now().Unix(),
	}

	log.Printf("Set secret: %s", key)
	return resp, nil
}

func (h *Handler) handleDeleteSecret(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	key, ok := cmd.InputVariables["key"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "key is required"
		return resp, nil
	}

	// Apply secret prefix
	fullKey := h.getFullKey(key)

	if err := h.storage.Delete(fullKey); err != nil {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "deletion_error"
		resp.ErrorMessage = err.Error()
		return resp, nil
	}

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"key":            key,
		"secret_deleted": true,
	}

	log.Printf("Deleted secret: %s", key)
	return resp, nil
}

func (h *Handler) handleListSecrets(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	prefix := ""
	if p, ok := cmd.InputVariables["prefix"].(string); ok {
		prefix = h.getFullKey(p)
	} else {
		prefix = h.secretPrefix
	}

	secrets, err := h.storage.List(prefix)
	if err != nil {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "list_error"
		resp.ErrorMessage = err.Error()
		return resp, nil
	}

	// Build list response
	secretList := make([]map[string]interface{}, len(secrets))
	for i, secret := range secrets {
		secretList[i] = map[string]interface{}{
			"key":        secret.Key,
			"metadata":   secret.Metadata,
			"created_at": secret.CreatedAt.Unix(),
			"updated_at": secret.UpdatedAt.Unix(),
		}
	}

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"secrets": secretList,
		"count":   len(secrets),
	}

	log.Printf("Listed secrets with prefix: %s (count: %d)", prefix, len(secrets))
	return resp, nil
}

func (h *Handler) handleHealthCheck(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"status":       "healthy",
		"service_name": "vault-service",
		"timestamp":    time.Now().Unix(),
	}

	return resp, nil
}

func (h *Handler) handleUnknownOperation(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	resp.Status = nats.ResponseStatusError
	resp.ErrorCode = "unknown_operation"
	resp.ErrorMessage = fmt.Sprintf("Unknown operation: %s", cmd.Operation)
	return resp, nil
}

// getFullKey applies the secret prefix to a key
func (h *Handler) getFullKey(key string) string {
	if h.secretPrefix == "" {
		return key
	}
	return h.secretPrefix + "/" + key
}
