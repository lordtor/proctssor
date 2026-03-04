package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"gitlab-service/internal/nats"
)

// Handler handles GitLab service commands
type Handler struct {
	gitlabClient *GitLabClient
}

// NewHandler creates a new GitLab service handler
func NewHandler(gitlabClient *GitLabClient) *Handler {
	return &Handler{
		gitlabClient: gitlabClient,
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
	case "create_mr":
		return h.handleCreateMR(ctx, cmd, resp)
	case "get_project":
		return h.handleGetProject(ctx, cmd, resp)
	case "create_issue":
		return h.handleCreateIssue(ctx, cmd, resp)
	case "get_issue":
		return h.handleGetIssue(ctx, cmd, resp)
	case "add_comment":
		return h.handleAddComment(ctx, cmd, resp)
	case "create_pipeline":
		return h.handleCreatePipeline(ctx, cmd, resp)
	case "get_commit_status":
		return h.handleGetCommitStatus(ctx, cmd, resp)
	case "get_merge_request":
		return h.handleGetMergeRequest(ctx, cmd, resp)
	case "approve_merge_request":
		return h.handleApproveMR(ctx, cmd, resp)
	case "health_check":
		return h.handleHealthCheck(ctx, cmd, resp)
	default:
		return h.handleUnknownOperation(ctx, cmd, resp)
	}
}

func (h *Handler) handleCreateMR(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	projectID, ok := cmd.InputVariables["project_id"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "project_id is required"
		return resp, nil
	}

	sourceBranch, ok := cmd.InputVariables["source_branch"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "source_branch is required"
		return resp, nil
	}

	targetBranch, ok := cmd.InputVariables["target_branch"].(string)
	if !ok {
		targetBranch = "main"
	}

	title, ok := cmd.InputVariables["title"].(string)
	if !ok {
		title = fmt.Sprintf("Merge request from %s", sourceBranch)
	}

	description, _ := cmd.InputVariables["description"].(string)

	// Create merge request via GitLab API
	mr, err := h.gitlabClient.CreateMergeRequest(ctx, projectID, sourceBranch, targetBranch, title, description)
	if err != nil {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "gitlab_api_error"
		resp.ErrorMessage = fmt.Sprintf("Failed to create merge request: %v", err)
		return resp, nil
	}

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"merge_request_iid": mr.IID,
		"merge_request_id":  mr.ID,
		"merge_request_url": mr.WebURL,
		"state":             mr.State,
	}

	log.Printf("Created merge request #%d for project %s", mr.IID, projectID)
	return resp, nil
}

func (h *Handler) handleGetProject(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	projectID, ok := cmd.InputVariables["project_id"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "project_id is required"
		return resp, nil
	}

	project, err := h.gitlabClient.GetProject(ctx, projectID)
	if err != nil {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "gitlab_api_error"
		resp.ErrorMessage = fmt.Sprintf("Failed to get project: %v", err)
		return resp, nil
	}

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"project_id":                  project.ID,
		"project_name":                project.Name,
		"project_path":                project.Path,
		"project_path_with_namespace": project.PathWithNamespace,
		"project_web_url":             project.WebURL,
		"default_branch":              project.DefaultBranch,
	}

	return resp, nil
}

func (h *Handler) handleCreateIssue(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	projectID, ok := cmd.InputVariables["project_id"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "project_id is required"
		return resp, nil
	}

	title, ok := cmd.InputVariables["title"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "title is required"
		return resp, nil
	}

	description, _ := cmd.InputVariables["description"].(string)
	labels, _ := cmd.InputVariables["labels"].(string)

	issue, err := h.gitlabClient.CreateIssue(ctx, projectID, title, description, labels)
	if err != nil {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "gitlab_api_error"
		resp.ErrorMessage = fmt.Sprintf("Failed to create issue: %v", err)
		return resp, nil
	}

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"issue_id":    issue.ID,
		"issue_iid":   issue.IID,
		"issue_url":   issue.WebURL,
		"issue_state": issue.State,
		"issue_title": issue.Title,
	}

	return resp, nil
}

func (h *Handler) handleGetIssue(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	projectID, ok := cmd.InputVariables["project_id"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "project_id is required"
		return resp, nil
	}

	issueIID, ok := cmd.InputVariables["issue_iid"].(float64)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "issue_iid is required"
		return resp, nil
	}

	issue, err := h.gitlabClient.GetIssue(ctx, projectID, int(issueIID))
	if err != nil {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "gitlab_api_error"
		resp.ErrorMessage = fmt.Sprintf("Failed to get issue: %v", err)
		return resp, nil
	}

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"issue_id":    issue.ID,
		"issue_iid":   issue.IID,
		"issue_url":   issue.WebURL,
		"issue_state": issue.State,
		"issue_title": issue.Title,
		"issue_desc":  issue.Description,
	}

	return resp, nil
}

func (h *Handler) handleAddComment(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	projectID, ok := cmd.InputVariables["project_id"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "project_id is required"
		return resp, nil
	}

	issueIID, hasIssue := cmd.InputVariables["issue_iid"].(float64)
	mrIID, hasMR := cmd.InputVariables["mr_iid"].(float64)
	body, ok := cmd.InputVariables["body"].(string)

	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "body is required"
		return resp, nil
	}

	var err error
	if hasIssue {
		_, err = h.gitlabClient.AddIssueComment(ctx, projectID, int(issueIID), body)
	} else if hasMR {
		_, err = h.gitlabClient.AddMRComment(ctx, projectID, int(mrIID), body)
	} else {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "either issue_iid or mr_iid is required"
		return resp, nil
	}

	if err != nil {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "gitlab_api_error"
		resp.ErrorMessage = fmt.Sprintf("Failed to add comment: %v", err)
		return resp, nil
	}

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"comment_added": true,
	}

	return resp, nil
}

func (h *Handler) handleCreatePipeline(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	projectID, ok := cmd.InputVariables["project_id"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "project_id is required"
		return resp, nil
	}

	ref, ok := cmd.InputVariables["ref"].(string)
	if !ok {
		ref = "main"
	}

	variables := make(map[string]string)
	if vars, ok := cmd.InputVariables["variables"].(map[string]interface{}); ok {
		for k, v := range vars {
			if s, ok := v.(string); ok {
				variables[k] = s
			}
		}
	}

	pipeline, err := h.gitlabClient.CreatePipeline(ctx, projectID, ref, variables)
	if err != nil {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "gitlab_api_error"
		resp.ErrorMessage = fmt.Sprintf("Failed to create pipeline: %v", err)
		return resp, nil
	}

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"pipeline_id":      pipeline.ID,
		"pipeline_status":  pipeline.Status,
		"pipeline_web_url": pipeline.WebURL,
	}

	return resp, nil
}

func (h *Handler) handleGetCommitStatus(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	projectID, ok := cmd.InputVariables["project_id"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "project_id is required"
		return resp, nil
	}

	sha, ok := cmd.InputVariables["sha"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "sha is required"
		return resp, nil
	}

	statuses, err := h.gitlabClient.GetCommitStatuses(ctx, projectID, sha)
	if err != nil {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "gitlab_api_error"
		resp.ErrorMessage = fmt.Sprintf("Failed to get commit status: %v", err)
		return resp, nil
	}

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"commit_sha":      sha,
		"commit_statuses": statuses,
	}

	return resp, nil
}

func (h *Handler) handleGetMergeRequest(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	projectID, ok := cmd.InputVariables["project_id"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "project_id is required"
		return resp, nil
	}

	mrIID, ok := cmd.InputVariables["mr_iid"].(float64)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "mr_iid is required"
		return resp, nil
	}

	mr, err := h.gitlabClient.GetMergeRequest(ctx, projectID, int(mrIID))
	if err != nil {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "gitlab_api_error"
		resp.ErrorMessage = fmt.Sprintf("Failed to get merge request: %v", err)
		return resp, nil
	}

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"mr_id":            mr.ID,
		"mr_iid":           mr.IID,
		"mr_title":         mr.Title,
		"mr_state":         mr.State,
		"mr_web_url":       mr.WebURL,
		"mr_source_branch": mr.SourceBranch,
		"mr_target_branch": mr.TargetBranch,
		"mr_merged_by":     mr.MergedBy.Username,
		"mr_merged_at":     mr.MergedAt,
		"mr_has_conflicts": mr.HasConflicts,
	}

	return resp, nil
}

func (h *Handler) handleApproveMR(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	projectID, ok := cmd.InputVariables["project_id"].(string)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "project_id is required"
		return resp, nil
	}

	mrIID, ok := cmd.InputVariables["mr_iid"].(float64)
	if !ok {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "invalid_input"
		resp.ErrorMessage = "mr_iid is required"
		return resp, nil
	}

	err := h.gitlabClient.ApproveMergeRequest(ctx, projectID, int(mrIID))
	if err != nil {
		resp.Status = nats.ResponseStatusError
		resp.ErrorCode = "gitlab_api_error"
		resp.ErrorMessage = fmt.Sprintf("Failed to approve merge request: %v", err)
		return resp, nil
	}

	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"mr_approved": true,
	}

	return resp, nil
}

func (h *Handler) handleHealthCheck(ctx context.Context, cmd *nats.WorkflowCommand, resp *nats.ServiceResponse) (*nats.ServiceResponse, error) {
	resp.Status = nats.ResponseStatusSuccess
	resp.OutputVariables = map[string]interface{}{
		"status":           "healthy",
		"service_name":     "gitlab-service",
		"gitlab_connected": h.gitlabClient.IsConnected(),
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
