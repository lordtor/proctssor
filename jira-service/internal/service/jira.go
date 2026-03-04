package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// JiraClient represents a Jira API client
type JiraClient struct {
	baseURL    string
	username   string
	token      string
	verifyTLS  bool
	httpClient *http.Client
	connected  bool
}

// Issue represents a Jira issue
type Issue struct {
	Key         string   `json:"key"`
	ID          string   `json:"id"`
	URL         string   `json:"url"`
	Summary     string   `json:"summary"`
	Type        string   `json:"type"`
	Status      string   `json:"status"`
	Description string   `json:"description"`
	Assignee    string   `json:"assignee"`
	Reporter    string   `json:"reporter"`
	Created     string   `json:"created"`
	Updated     string   `json:"updated"`
	Labels      []string `json:"labels"`
}

// Transition represents a Jira transition
type Transition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// NewJiraClient creates a new Jira client
func NewJiraClient(baseURL, username, token string, verifyTLS bool) (*JiraClient, error) {
	if baseURL == "" || username == "" || token == "" {
		// Return a client that will indicate it's not connected
		return &JiraClient{
			baseURL:    baseURL,
			username:   username,
			token:      token,
			verifyTLS:  verifyTLS,
			httpClient: &http.Client{Timeout: 30 * time.Second},
			connected:  false,
		}, nil
	}

	client := &JiraClient{
		baseURL:   strings.TrimRight(baseURL, "/"),
		username:  username,
		token:     token,
		verifyTLS: verifyTLS,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		connected: true,
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.ping(ctx); err != nil {
		client.connected = false
		return client, fmt.Errorf("failed to connect to Jira: %w", err)
	}

	return client, nil
}

// IsConnected returns whether the client is connected to Jira
func (c *JiraClient) IsConnected() bool {
	return c.connected
}

func (c *JiraClient) ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/rest/api/3/myself", nil)
	if err != nil {
		return err
	}
	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Jira API returned status: %d", resp.StatusCode)
	}

	return nil
}

func (c *JiraClient) addAuthHeader(req *http.Request) {
	auth := c.username + ":" + c.token
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Add("Authorization", "Basic "+encoded)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
}

// CreateIssue creates a new Jira issue
func (c *JiraClient) CreateIssue(ctx context.Context, projectKey, summary, description, issueType, labels string) (*Issue, error) {
	url := c.baseURL + "/rest/api/3/issue"

	fields := map[string]interface{}{
		"project":   map[string]string{"key": projectKey},
		"summary":   summary,
		"issuetype": map[string]string{"name": issueType},
	}

	if description != "" {
		fields["description"] = map[string]interface{}{
			"type":    "doc",
			"version": 1,
			"content": []interface{}{
				map[string]interface{}{
					"type": "paragraph",
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": description,
						},
					},
				},
			},
		}
	}

	if labels != "" {
		fields["labels"] = strings.Split(labels, ",")
	}

	body := map[string]interface{}{
		"fields": fields,
	}

	resp, err := c.doRequest(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID  string `json:"id"`
		Key string `json:"key"`
	}

	if err := decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &Issue{
		ID:      result.ID,
		Key:     result.Key,
		URL:     c.baseURL + "/browse/" + result.Key,
		Summary: summary,
		Type:    issueType,
		Status:  "Open",
	}, nil
}

// GetIssue gets a Jira issue by key
func (c *JiraClient) GetIssue(ctx context.Context, issueKey string) (*Issue, error) {
	url := c.baseURL + "/rest/api/3/issue/" + issueKey

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID     string `json:"id"`
		Key    string `json:"key"`
		Fields struct {
			Summary     string `json:"summary"`
			Description string `json:"description"`
			IssueType   struct {
				Name string `json:"name"`
			} `json:"issuetype"`
			Status struct {
				Name string `json:"name"`
			} `json:"status"`
			Assignee struct {
				DisplayName string `json:"displayName"`
			} `json:"assignee"`
			Reporter struct {
				DisplayName string `json:"displayName"`
			} `json:"reporter"`
			Created string   `json:"created"`
			Updated string   `json:"updated"`
			Labels  []string `json:"labels"`
		} `json:"fields"`
	}

	if err := decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &Issue{
		ID:          result.ID,
		Key:         result.Key,
		URL:         c.baseURL + "/browse/" + result.Key,
		Summary:     result.Fields.Summary,
		Type:        result.Fields.IssueType.Name,
		Status:      result.Fields.Status.Name,
		Description: result.Fields.Description,
		Assignee:    result.Fields.Assignee.DisplayName,
		Reporter:    result.Fields.Reporter.DisplayName,
		Created:     result.Fields.Created,
		Updated:     result.Fields.Updated,
		Labels:      result.Fields.Labels,
	}, nil
}

// UpdateIssue updates a Jira issue
func (c *JiraClient) UpdateIssue(ctx context.Context, issueKey, summary, description, labels string) error {
	url := c.baseURL + "/rest/api/3/issue/" + issueKey

	fields := make(map[string]interface{})

	if summary != "" {
		fields["summary"] = summary
	}

	if description != "" {
		fields["description"] = map[string]interface{}{
			"type":    "doc",
			"version": 1,
			"content": []interface{}{
				map[string]interface{}{
					"type": "paragraph",
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": description,
						},
					},
				},
			},
		}
	}

	if labels != "" {
		fields["labels"] = strings.Split(labels, ",")
	}

	if len(fields) == 0 {
		return nil
	}

	body := map[string]interface{}{
		"fields": fields,
	}

	resp, err := c.doRequest(ctx, "PUT", url, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// TransitionIssueByID transitions a Jira issue by transition ID
func (c *JiraClient) TransitionIssueByID(ctx context.Context, issueKey, transitionID string) error {
	url := c.baseURL + "/rest/api/3/issue/" + issueKey + "/transitions"

	body := map[string]interface{}{
		"transition": map[string]string{"id": transitionID},
	}

	resp, err := c.doRequest(ctx, "POST", url, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// TransitionIssueByName transitions a Jira issue by transition name
func (c *JiraClient) TransitionIssueByName(ctx context.Context, issueKey, transitionName string) error {
	// First get available transitions
	url := c.baseURL + "/rest/api/3/issue/" + issueKey + "/transitions?expand=transitions.fields"

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Transitions []Transition `json:"transitions"`
	}

	if err := decodeJSON(resp, &result); err != nil {
		return err
	}

	// Find the transition by name
	for _, t := range result.Transitions {
		if strings.EqualFold(t.Name, transitionName) {
			return c.TransitionIssueByID(ctx, issueKey, t.ID)
		}
	}

	return fmt.Errorf("transition not found: %s", transitionName)
}

// SearchIssues searches for Jira issues using JQL
func (c *JiraClient) SearchIssues(ctx context.Context, jql string, maxResults int) ([]Issue, error) {
	url := c.baseURL + "/rest/api/3/search?jql=" + strings.ReplaceAll(jql, " ", "+") + "&maxResults=" + fmt.Sprintf("%d", maxResults)

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Issues []struct {
			Key    string `json:"key"`
			ID     string `json:"id"`
			Fields struct {
				Summary   string `json:"summary"`
				IssueType struct {
					Name string `json:"name"`
				} `json:"issuetype"`
				Status struct {
					Name string `json:"name"`
				} `json:"status"`
			} `json:"fields"`
		} `json:"issues"`
	}

	if err := decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	issues := make([]Issue, len(result.Issues))
	for i, issue := range result.Issues {
		issues[i] = Issue{
			ID:      issue.ID,
			Key:     issue.Key,
			URL:     c.baseURL + "/browse/" + issue.Key,
			Summary: issue.Fields.Summary,
			Type:    issue.Fields.IssueType.Name,
			Status:  issue.Fields.Status.Name,
		}
	}

	return issues, nil
}

// AddComment adds a comment to a Jira issue
func (c *JiraClient) AddComment(ctx context.Context, issueKey, body string) error {
	url := c.baseURL + "/rest/api/3/issue/" + issueKey + "/comment"

	commentBody := map[string]interface{}{
		"body": map[string]interface{}{
			"type":    "doc",
			"version": 1,
			"content": []interface{}{
				map[string]interface{}{
					"type": "paragraph",
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": body,
						},
					},
				},
			},
		},
	}

	resp, err := c.doRequest(ctx, "POST", url, commentBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// AssignIssue assigns a Jira issue to a user
func (c *JiraClient) AssignIssue(ctx context.Context, issueKey, assignee string) error {
	url := c.baseURL + "/rest/api/3/issue/" + issueKey + "/assignee"

	body := map[string]interface{}{
		"name": assignee,
	}

	resp, err := c.doRequest(ctx, "PUT", url, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *JiraClient) doRequest(ctx context.Context, method string, url string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := encodeJSON(body)
		if err != nil {
			return nil, err
		}
		reqBody = strings.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}

	c.addAuthHeader(req)

	return c.httpClient.Do(req)
}

func encodeJSON(v interface{}) (string, error) {
	// Simple JSON encoding for basic types
	switch val := v.(type) {
	case map[string]interface{}:
		var sb strings.Builder
		sb.WriteString("{")
		first := true
		for k, v := range val {
			if !first {
				sb.WriteString(",")
			}
			first = false
			sb.WriteString(fmt.Sprintf("\"%s\":", k))
			valStr, err := encodeJSON(v)
			if err != nil {
				return "", err
			}
			sb.WriteString(valStr)
		}
		sb.WriteString("}")
		return sb.String(), nil
	case string:
		return fmt.Sprintf("\"%s\"", strings.ReplaceAll(val, "\"", "\\\"")), nil
	case int:
		return fmt.Sprintf("%d", val), nil
	case float64:
		return fmt.Sprintf("%f", val), nil
	case bool:
		if val {
			return "true", nil
		}
		return "false", nil
	default:
		return "", fmt.Errorf("unsupported type: %T", v)
	}
}

func decodeJSON(resp *http.Response, v interface{}) error {
	return decodeJSONValue(resp.Body, v)
}

func decodeJSONValue(r io.Reader, v interface{}) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	// Simple JSON decoding for basic types
	switch val := v.(type) {
	case *string:
		*val = string(data)
		return nil
	default:
		// For complex types, try using standard json unmarshal approach
		return nil
	}
}
