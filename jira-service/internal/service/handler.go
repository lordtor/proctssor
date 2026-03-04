package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"jira-service/internal/nats"
)

// Handler handles Jira service commands
type Handler struct {
	jiraClient *JiraClient
}

// NewHandler creates a new Jira service handler
func NewHandler(jiraClient *JiraClient) *Handler {
	return &Handler{
		jiraClient: jiraClient,
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
	case "create_issue":
		return h.handleCreateIssue(ctx, cmd, resp)
	case "get_issue":
		return h.handleGetIssue(ctx, cmd, resp)
	case "update_issue":
		return h.handleUpdateIssue(ctx, cmd, resp)
	case "transition_issue":
		return h.handleTransitionIssue(ctx, cmd, resp)
	case "search_issues":
		return h.handleSearchIssues(ctx, cmd, resp)
	case "add_comment":
		return h.handleAddComment(ctx, cmd, resp)
	case "assign_issue":
		return h.handleAssignIssue(ctx, cmd, resp)
	case "health_check":
		return h.handleHealthCheck(ctx, cmd, resp)
	default:
		return h.handleUnknownOperation(ctx, cmd, resp)
	}
}

func (h *Handler) handleCreateIssue(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	projectKey, ok := cmd.InputVariables["project_key"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "project_key is required"
		return resp, nil
	}

	summary, ok := cmd.InputVariables["summary"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "summary is required"
		return resp, nil
	}

	description, _ := cmd.InputVariables["description"].(string)
	issueType, _ := cmd.InputVariables["issue_type"].(string)
	if issueType == "" {
		issueType = "Task"
	}
	labels, _ := cmd.InputVariables["labels"].(string)

	// Create issue via Jira API
	issue, err := h.jiraClient.CreateIssue(ctx, projectKey, summary, description, issueType, labels)
	if err != nil {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "jira_api_error"
		resp.ErrorMessage = fmt.Sprintf("Failed to create issue: %v", err)
		return resp, nil
	}

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"issue_key":     issue.Key,
		"issue_id":      issue.ID,
		"issue_url":     issue.URL,
		"issue_summary": issue.Summary,
		"issue_type":    issue.Type,
		"issue_status":  issue.Status,
	}

	log.Printf("Created issue %s in project %s", issue.Key, projectKey)
	return resp, nil
}

func (h *Handler) handleGetIssue(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	issueKey, ok := cmd.InputVariables["issue_key"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "issue_key is required"
		return resp, nil
	}

	issue, err := h.jiraClient.GetIssue(ctx, issueKey)
	if err != nil {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "jira_api_error"
		resp.ErrorMessage = fmt.Sprintf("Failed to get issue: %v", err)
		return resp, nil
	}

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"issue_key":     issue.Key,
		"issue_id":      issue.ID,
		"issue_url":     issue.URL,
		"issue_summary": issue.Summary,
		"issue_type":    issue.Type,
		"issue_status":  issue.Status,
		"issue_desc":    issue.Description,
		"assignee":      issue.Assignee,
		"reporter":      issue.Reporter,
		"created":       issue.Created,
		"updated":       issue.Updated,
	}

	return resp, nil
}

func (h *Handler) handleUpdateIssue(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	issueKey, ok := cmd.InputVariables["issue_key"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "issue_key is required"
		return resp, nil
	}

	summary, hasSummary := cmd.InputVariables["summary"].(string)
	description, hasDescription := cmd.InputVariables["description"].(string)
	labels, hasLabels := cmd.InputVariables["labels"].(string)

	if !hasSummary && !hasDescription && !hasLabels {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "At least one of summary, description, or labels is required"
		return resp, nil
	}

	err := h.jiraClient.UpdateIssue(ctx, issueKey, summary, description, labels)
	if err != nil {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "jira_api_error"
		resp.ErrorMessage = fmt.Sprintf("Failed to update issue: %v", err)
		return resp, nil
	}

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"issue_key": issueKey,
		"updated":   true,
	}

	log.Printf("Updated issue %s", issueKey)
	return resp, nil
}

func (h *Handler) handleTransitionIssue(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	issueKey, ok := cmd.InputVariables["issue_key"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "issue_key is required"
		return resp, nil
	}

	transitionID, hasTransitionID := cmd.InputVariables["transition_id"].(string)
	transitionName, hasTransitionName := cmd.InputVariables["transition_name"].(string)

	if !hasTransitionID && !hasTransitionName {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "transition_id or transition_name is required"
		return resp, nil
	}

	var err error
	if hasTransitionID {
		err = h.jiraClient.TransitionIssueByID(ctx, issueKey, transitionID)
	} else {
		err = h.jiraClient.TransitionIssueByName(ctx, issueKey, transitionName)
	}

	if err != nil {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "jira_api_error"
		resp.ErrorMessage = fmt.Sprintf("Failed to transition issue: %v", err)
		return resp, nil
	}

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"issue_key":    issueKey,
		"transitioned": true,
	}

	log.Printf("Transitioned issue %s to %s", issueKey, transitionName)
	return resp, nil
}

func (h *Handler) handleSearchIssues(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	jql, ok := cmd.InputVariables["jql"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "jql is required"
		return resp, nil
	}

	maxResults := 50
	if max, ok := cmd.InputVariables["max_results"].(float64); ok {
		maxResults = int(max)
	}

	issues, err := h.jiraClient.SearchIssues(ctx, jql, maxResults)
	if err != nil {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "jira_api_error"
		resp.ErrorMessage = fmt.Sprintf("Failed to search issues: %v", err)
		return resp, nil
	}

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"issues":      issues,
		"total_count": len(issues),
	}

	return resp, nil
}

func (h *Handler) handleAddComment(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	issueKey, ok := cmd.InputVariables["issue_key"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "issue_key is required"
		return resp, nil
	}

	body, ok := cmd.InputVariables["body"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "body is required"
		return resp, nil
	}

	err := h.jiraClient.AddComment(ctx, issueKey, body)
	if err != nil {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "jira_api_error"
		resp.ErrorMessage = fmt.Sprintf("Failed to add comment: %v", err)
		return resp, nil
	}

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"issue_key":     issueKey,
		"comment_added": true,
	}

	return resp, nil
}

func (h *Handler) handleAssignIssue(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	issueKey, ok := cmd.InputVariables["issue_key"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "issue_key is required"
		return resp, nil
	}

	assignee, ok := cmd.InputVariables["assignee"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "assignee is required"
		return resp, nil
	}

	err := h.jiraClient.AssignIssue(ctx, issueKey, assignee)
	if err != nil {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "jira_api_error"
		resp.ErrorMessage = fmt.Sprintf("Failed to assign issue: %v", err)
		return resp, nil
	}

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"issue_key": issueKey,
		"assigned":  true,
		"assignee":  assignee,
	}

	return resp, nil
}

func (h *Handler) handleHealthCheck(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"status":         "healthy",
		"service_name":   "jira-service",
		"jira_connected": h.jiraClient.IsConnected(),
		"timestamp":      time.Now().Unix(),
	}

	return resp, nil
}

func (h *Handler) handleUnknownOperation(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	resp.Status = nats.ResponseStatusError
	resp.ErrorCode = "unknown_operation"
	resp.ErrorMessage = fmt.Sprintf("Unknown operation: %s", cmd.Operation)
	return resp, nil
}
