package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"notification-service/internal/config"
	"notification-service/internal/nats"
)

// Handler handles notification service commands
type Handler struct {
	emailClient   *EmailClient
	slackClient   *SlackClient
	webhookClient *WebhookClient
}

// EmailClient handles sending emails
type EmailClient struct {
	config *config.EmailConfig
}

// SlackClient handles sending Slack messages
type SlackClient struct {
	config     *config.SlackConfig
	httpClient *http.Client
}

// WebhookClient handles sending webhooks
type WebhookClient struct {
	config     *config.WebhookConfig
	httpClient *http.Client
}

// NewHandler creates a new notification service handler
func NewHandler(cfg *config.Config) *Handler {
	return &Handler{
		emailClient:   &EmailClient{config: &cfg.Email},
		slackClient:   &SlackClient{config: &cfg.Slack, httpClient: &http.Client{Timeout: cfg.Webhook.Timeout}},
		webhookClient: &WebhookClient{config: &cfg.Webhook, httpClient: &http.Client{Timeout: cfg.Webhook.Timeout}},
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
	case "send_email":
		return h.handleSendEmail(ctx, cmd, resp)
	case "send_slack":
		return h.handleSendSlack(ctx, cmd, resp)
	case "send_webhook":
		return h.handleSendWebhook(ctx, cmd, resp)
	case "health_check":
		return h.handleHealthCheck(ctx, cmd, resp)
	default:
		return h.handleUnknownOperation(ctx, cmd, resp)
	}
}

func (h *Handler) handleSendEmail(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	to, ok := cmd.InputVariables["to"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "to (recipient email) is required"
		return resp, nil
	}

	subject, ok := cmd.InputVariables["subject"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "subject is required"
		return resp, nil
	}

	_, ok = cmd.InputVariables["body"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "body is required"
		return resp, nil
	}

	// Get optional fields
	from := h.emailClient.config.FromAddress
	if fromAddr, ok := cmd.InputVariables["from"].(string); ok && fromAddr != "" {
		from = fromAddr
	}

	// For now, we just log the email - in production, connect to actual SMTP server
	log.Printf("Sending email: to=%s, subject=%s", to, subject)

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"email_sent": true,
		"recipient":  to,
		"subject":    subject,
		"from":       from,
		"sent_at":    time.Now().Unix(),
	}

	return resp, nil
}

func (h *Handler) handleSendSlack(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	message, ok := cmd.InputVariables["message"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "message is required"
		return resp, nil
	}

	channel := h.slackClient.config.DefaultChannel
	if ch, ok := cmd.InputVariables["channel"].(string); ok && ch != "" {
		channel = ch
	}

	webhookURL := h.slackClient.config.WebhookURL
	if wh, ok := cmd.InputVariables["webhook_url"].(string); ok && wh != "" {
		webhookURL = wh
	}

	// If webhook URL is configured, send to Slack
	if webhookURL != "" {
		slackMsg := map[string]interface{}{
			"text":    message,
			"channel": channel,
		}

		jsonData, err := json.Marshal(slackMsg)
		if err != nil {
			resp.Status = nats.ResponseStatusError
			resp.ErrorCode = "slack_error"
			resp.ErrorMessage = fmt.Sprintf("Failed to marshal Slack message: %v", err)
			return resp, nil
		}

		req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(jsonData))
		if err != nil {
			resp.Status = nats.ResponseStatusError
			resp.ErrorCode = "slack_error"
			resp.ErrorMessage = fmt.Sprintf("Failed to create Slack request: %v", err)
			return resp, nil
		}

		req.Header.Set("Content-Type", "application/json")

		httpResp, err := h.slackClient.httpClient.Do(req)
		if err != nil {
			resp.Status = nats.ResponseStatusError
			resp.ErrorCode = "slack_error"
			resp.ErrorMessage = fmt.Sprintf("Failed to send Slack message: %v", err)
			return resp, nil
		}
		defer httpResp.Body.Close()

		if httpResp.StatusCode >= 400 {
			resp.Status = nats.ResponseStatusError
			resp.ErrorCode = "slack_error"
			resp.ErrorMessage = fmt.Sprintf("Slack API returned status: %d", httpResp.StatusCode)
			return resp, nil
		}
	} else {
		log.Printf("Sending slack message (no webhook configured): channel=%s, message=%s", channel, message)
	}

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"slack_sent": true,
		"channel":    channel,
		"message":    message,
		"sent_at":    time.Now().Unix(),
	}

	return resp, nil
}

func (h *Handler) handleSendWebhook(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	url, ok := cmd.InputVariables["url"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "url is required"
		return resp, nil
	}

	method := "POST"
	if m, ok := cmd.InputVariables["method"].(string); ok && m != "" {
		method = m
	}

	// Prepare request body
	var requestBody io.Reader
	if body, ok := cmd.InputVariables["body"].(string); ok && body != "" {
		requestBody = bytes.NewReader([]byte(body))
	} else if bodyMap, ok := cmd.InputVariables["body"].(map[string]interface{}); ok {
		jsonData, err := json.Marshal(bodyMap)
		if err != nil {
			resp.Status = nats.ResponseStatusError
			resp.ErrorCode = "webhook_error"
			resp.ErrorMessage = fmt.Sprintf("Failed to marshal webhook body: %v", err)
			return resp, nil
		}
		requestBody = bytes.NewReader(jsonData)
	}

	// Prepare headers
	headers := make(http.Header)
	if hs, ok := cmd.InputVariables["headers"].(map[string]interface{}); ok {
		for k, v := range hs {
			if s, ok := v.(string); ok {
				headers.Add(k, s)
			}
		}
	}

	// Set default content type if not specified
	if headers.Get("Content-Type") == "" {
		headers.Set("Content-Type", "application/json")
	}

	// Send webhook with retries
	var lastErr error
	for i := 0; i <= h.webhookClient.config.RetryCount; i++ {
		req, err := http.NewRequestWithContext(ctx, method, url, requestBody)
		if err != nil {
			lastErr = fmt.Errorf("failed to create webhook request: %w", err)
			continue
		}

		req.Header = headers

		httpResp, err := h.webhookClient.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("webhook request failed: %w", err)
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}
		defer httpResp.Body.Close()

		respBody, _ := io.ReadAll(httpResp.Body)

		if httpResp.StatusCode >= 400 {
			lastErr = fmt.Errorf("webhook returned status %d: %s", httpResp.StatusCode, string(respBody))
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		// Success
		resp.Status = nats.ResponseStatusSuccess
		resp.OutputVariables = map[string]interface{}{
			"webhook_sent":  true,
			"url":           url,
			"method":        method,
			"status_code":   httpResp.StatusCode,
			"response_body": string(respBody),
			"sent_at":       time.Now().Unix(),
		}

		return resp, nil
	}

	resp.Status = nats.ResponseStatusError
	resp.ErrorCode = "webhook_error"
	resp.ErrorMessage = fmt.Sprintf("Failed after %d retries: %v", h.webhookClient.config.RetryCount+1, lastErr)
	return resp, nil
}

func (h *Handler) handleHealthCheck(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"status":           "healthy",
		"service_name":     "notification-service",
		"email_configured": h.emailClient.config.FromAddress != "",
		"slack_configured": h.slackClient.config.WebhookURL != "",
		"timestamp":        time.Now().Unix(),
	}

	return resp, nil
}

func (h *Handler) handleUnknownOperation(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	resp.Status = nats.ResponseStatusError
	resp.ErrorCode = "unknown_operation"
	resp.ErrorMessage = fmt.Sprintf("Unknown operation: %s", cmd.Operation)
	return resp, nil
}
