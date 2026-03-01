package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/simonski/task/internal/config"
	"github.com/simonski/task/internal/store"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

type AuthResponse struct {
	Token string     `json:"token"`
	User  store.User `json:"user"`
}

type StatusResponse struct {
	Status        string      `json:"status"`
	Authenticated bool        `json:"authenticated"`
	ServerVersion string      `json:"server_version"`
	User          *store.User `json:"user,omitempty"`
}

type CountSummary = store.CountSummary

type ProjectCreateRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type TaskCreateRequest struct {
	ProjectID          int64  `json:"project_id"`
	ParentID           *int64 `json:"parent_id,omitempty"`
	Type               string `json:"type"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
	Priority           int    `json:"priority"`
	Assignee           string `json:"assignee"`
}

type TaskUpdateRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ParentID    *int64 `json:"parent_id,omitempty"`
	Assignee    string `json:"assignee"`
	Status      string `json:"status,omitempty"`
}

type CommentCreateRequest struct {
	Comment string `json:"comment"`
}

type DependencyRequest struct {
	ProjectID int64 `json:"project_id"`
	TaskID    int64 `json:"task_id"`
	DependsOn int64 `json:"depends_on"`
}

func New(cfg config.Config) *Client {
	return &Client{
		baseURL: strings.TrimRight(config.ResolveServerURL(cfg), "/"),
		token:   cfg.Token,
		http:    http.DefaultClient,
	}
}

func (c *Client) Register(username, password string) (store.User, error) {
	var user store.User
	err := c.doJSON(http.MethodPost, "/api/register", map[string]string{
		"username": username,
		"password": password,
	}, &user)
	return user, err
}

func (c *Client) Login(username, password string) (AuthResponse, error) {
	var response AuthResponse
	err := c.doJSON(http.MethodPost, "/api/login", map[string]string{
		"username": username,
		"password": password,
	}, &response)
	return response, err
}

func (c *Client) Logout() error {
	return c.doJSON(http.MethodPost, "/api/logout", nil, nil)
}

func (c *Client) Status() (StatusResponse, error) {
	var status StatusResponse
	err := c.doJSON(http.MethodGet, "/api/status", nil, &status)
	return status, err
}

func (c *Client) Count(projectID *int64) (CountSummary, error) {
	var summary CountSummary
	path := "/api/count"
	if projectID != nil {
		path = fmt.Sprintf("/api/count?project_id=%d", *projectID)
	}
	err := c.doJSON(http.MethodGet, path, nil, &summary)
	return summary, err
}

func (c *Client) CreateUser(username, password string) (store.User, error) {
	var user store.User
	err := c.doJSON(http.MethodPost, "/api/users", map[string]string{
		"username": username,
		"password": password,
	}, &user)
	return user, err
}

func (c *Client) SetUserEnabled(username string, enabled bool) error {
	action := "disable"
	if enabled {
		action = "enable"
	}
	return c.doJSON(http.MethodPost, "/api/users/"+username+"/"+action, nil, nil)
}

func (c *Client) ListUsers() ([]store.User, error) {
	var users []store.User
	err := c.doJSON(http.MethodGet, "/api/users", nil, &users)
	return users, err
}

func (c *Client) DeleteUser(username string) error {
	return c.doJSON(http.MethodDelete, "/api/users/"+username, nil, nil)
}

func (c *Client) CreateProject(title, description string) (store.Project, error) {
	var project store.Project
	err := c.doJSON(http.MethodPost, "/api/projects", ProjectCreateRequest{
		Title:       title,
		Description: description,
	}, &project)
	return project, err
}

func (c *Client) ListProjects() ([]store.Project, error) {
	var projects []store.Project
	err := c.doJSON(http.MethodGet, "/api/projects", nil, &projects)
	return projects, err
}

func (c *Client) GetProject(slugOrID string) (store.Project, error) {
	var project store.Project
	err := c.doJSON(http.MethodGet, "/api/projects/"+slugOrID, nil, &project)
	return project, err
}

func (c *Client) CreateTask(req TaskCreateRequest) (store.Task, error) {
	var task store.Task
	err := c.doJSON(http.MethodPost, "/api/tasks", req, &task)
	return task, err
}

func (c *Client) ListTasks(projectID int64) ([]store.Task, error) {
	return c.ListTasksFiltered(projectID, "", "", "", "", 0)
}

func (c *Client) ListTasksFiltered(projectID int64, taskType, status, search, assignee string, limit int) ([]store.Task, error) {
	var tasks []store.Task
	values := url.Values{}
	if taskType != "" {
		values.Set("type", taskType)
	}
	if status != "" {
		values.Set("status", status)
	}
	if search != "" {
		values.Set("q", search)
	}
	if assignee != "" {
		values.Set("assignee", assignee)
	}
	if limit > 0 {
		values.Set("limit", fmt.Sprintf("%d", limit))
	}
	path := fmt.Sprintf("/api/projects/%d/tasks", projectID)
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	err := c.doJSON(http.MethodGet, path, nil, &tasks)
	return tasks, err
}

func (c *Client) UpdateTask(id int64, req TaskUpdateRequest) (store.Task, error) {
	var task store.Task
	err := c.doJSON(http.MethodPut, fmt.Sprintf("/api/tasks/%d", id), req, &task)
	return task, err
}

func (c *Client) GetTask(id int64) (store.Task, error) {
	var task store.Task
	err := c.doJSON(http.MethodGet, fmt.Sprintf("/api/tasks/%d", id), nil, &task)
	return task, err
}

func (c *Client) ListHistory(id int64) ([]store.HistoryEvent, error) {
	var events []store.HistoryEvent
	err := c.doJSON(http.MethodGet, fmt.Sprintf("/api/tasks/%d/history", id), nil, &events)
	return events, err
}

func (c *Client) AddComment(id int64, comment string) (store.Comment, error) {
	var created store.Comment
	err := c.doJSON(http.MethodPost, fmt.Sprintf("/api/tasks/%d/comments", id), CommentCreateRequest{Comment: comment}, &created)
	return created, err
}

func (c *Client) ListComments(id int64) ([]store.Comment, error) {
	var comments []store.Comment
	err := c.doJSON(http.MethodGet, fmt.Sprintf("/api/tasks/%d/comments", id), nil, &comments)
	return comments, err
}

func (c *Client) AddDependency(req DependencyRequest) (store.Dependency, error) {
	var dependency store.Dependency
	err := c.doJSON(http.MethodPost, "/api/dependencies", req, &dependency)
	return dependency, err
}

func (c *Client) RemoveDependency(req DependencyRequest) error {
	return c.doJSON(http.MethodDelete, fmt.Sprintf("/api/dependencies?project_id=%d&task_id=%d&depends_on=%d", req.ProjectID, req.TaskID, req.DependsOn), nil, nil)
}

func (c *Client) ListDependencies(id int64) ([]store.Dependency, error) {
	var dependencies []store.Dependency
	err := c.doJSON(http.MethodGet, fmt.Sprintf("/api/tasks/%d/dependencies", id), nil, &dependencies)
	return dependencies, err
}

func (c *Client) doJSON(method, path string, body any, out any) error {
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Error != "" {
			return errors.New(apiErr.Error)
		}
		return fmt.Errorf("request failed with status %s", resp.Status)
	}

	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
