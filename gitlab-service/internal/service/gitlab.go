package service

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/xanzy/go-gitlab"
)

// GitLabClient wraps the GitLab API client
type GitLabClient struct {
	client *gitlab.Client
}

// NewGitLabClient creates a new GitLab client
func NewGitLabClient(url, token string, verifyTLS bool) (*GitLabClient, error) {
	// Get token from file if specified
	if token == "" {
		token = os.Getenv("GITLAB_TOKEN")
	}

	var client *gitlab.Client
	var err error

	// Build options
	var options []gitlab.ClientOptionFunc
	if url != "" {
		options = append(options, gitlab.WithBaseURL(url+"/api/v4"))
	}

	if token == "" {
		// Return client without token for unauthenticated requests
		client, err = gitlab.NewClient("", options...)
		if err != nil {
			return nil, fmt.Errorf("failed to create GitLab client: %w", err)
		}
	} else {
		client, err = gitlab.NewClient(token, options...)
		if err != nil {
			return nil, fmt.Errorf("failed to create GitLab client: %w", err)
		}
	}

	// Configure TLS verification
	if !verifyTLS {
		log.Println("Warning: TLS verification is disabled")
	}

	log.Printf("GitLab client initialized for %s", url)
	return &GitLabClient{client: client}, nil
}

// IsConnected checks if the GitLab client is connected
func (c *GitLabClient) IsConnected() bool {
	if c.client == nil {
		return false
	}
	// Try to get user to verify connection
	_, _, err := c.client.Users.CurrentUser()
	return err == nil
}

// GetProject gets a project by ID or path
func (c *GitLabClient) GetProject(ctx context.Context, projectID string) (*gitlab.Project, error) {
	project, _, err := c.client.Projects.GetProject(projectID, nil, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	return project, nil
}

// CreateMergeRequest creates a new merge request
func (c *GitLabClient) CreateMergeRequest(ctx context.Context, projectID, sourceBranch, targetBranch, title, description string) (*gitlab.MergeRequest, error) {
	opts := &gitlab.CreateMergeRequestOptions{
		SourceBranch: &sourceBranch,
		TargetBranch: &targetBranch,
		Title:        &title,
	}

	if description != "" {
		opts.Description = &description
	}

	mr, _, err := c.client.MergeRequests.CreateMergeRequest(projectID, opts, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to create merge request: %w", err)
	}
	return mr, nil
}

// GetMergeRequest gets a merge request by IID
func (c *GitLabClient) GetMergeRequest(ctx context.Context, projectID string, mrIID int) (*gitlab.MergeRequest, error) {
	mr, _, err := c.client.MergeRequests.GetMergeRequest(projectID, mrIID, nil, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get merge request: %w", err)
	}
	return mr, nil
}

// ApproveMergeRequest approves a merge request - uses MergeRequestApprovalsService
func (c *GitLabClient) ApproveMergeRequest(ctx context.Context, projectID string, mrIID int) error {
	_, _, err := c.client.MergeRequestApprovals.ApproveMergeRequest(projectID, mrIID, nil, gitlab.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to approve merge request: %w", err)
	}
	return nil
}

// CreateIssue creates a new issue
func (c *GitLabClient) CreateIssue(ctx context.Context, projectID, title, description, labels string) (*gitlab.Issue, error) {
	opts := &gitlab.CreateIssueOptions{
		Title: &title,
	}

	if description != "" {
		opts.Description = &description
	}

	issue, _, err := c.client.Issues.CreateIssue(projectID, opts, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}
	return issue, nil
}

// GetIssue gets an issue by IID
func (c *GitLabClient) GetIssue(ctx context.Context, projectID string, issueIID int) (*gitlab.Issue, error) {
	issue, _, err := c.client.Issues.GetIssue(projectID, issueIID, nil, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}
	return issue, nil
}

// AddIssueComment adds a comment to an issue
func (c *GitLabClient) AddIssueComment(ctx context.Context, projectID string, issueIID int, body string) (*gitlab.Note, error) {
	opts := &gitlab.CreateIssueNoteOptions{
		Body: &body,
	}

	note, _, err := c.client.Notes.CreateIssueNote(projectID, issueIID, opts, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to add issue comment: %w", err)
	}
	return note, nil
}

// AddMRComment adds a comment to a merge request
func (c *GitLabClient) AddMRComment(ctx context.Context, projectID string, mrIID int, body string) (*gitlab.Note, error) {
	opts := &gitlab.CreateMergeRequestNoteOptions{
		Body: &body,
	}

	note, _, err := c.client.Notes.CreateMergeRequestNote(projectID, mrIID, opts, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to add MR comment: %w", err)
	}
	return note, nil
}

// CreatePipeline creates a new pipeline
func (c *GitLabClient) CreatePipeline(ctx context.Context, projectID, ref string, variables map[string]string) (*gitlab.Pipeline, error) {
	opts := &gitlab.CreatePipelineOptions{
		Ref: &ref,
	}

	pipeline, _, err := c.client.Pipelines.CreatePipeline(projectID, opts, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to create pipeline: %w", err)
	}
	return pipeline, nil
}

// GetCommitStatuses gets commit statuses
func (c *GitLabClient) GetCommitStatuses(ctx context.Context, projectID, sha string) ([]*gitlab.CommitStatus, error) {
	statuses, _, err := c.client.Commits.GetCommitStatuses(projectID, sha, nil, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get commit statuses: %w", err)
	}
	return statuses, nil
}
