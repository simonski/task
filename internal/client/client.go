package client

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	osuser "os/user"
	"strings"

	"github.com/simonski/ticket/internal/config"
	"github.com/simonski/ticket/internal/store"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
	mode    string
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
	Title              string `json:"title"`
	Description        string `json:"description"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
}

type ProjectUpdateRequest struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
}

type TaskCreateRequest struct {
	ProjectID          int64  `json:"project_id"`
	ParentID           *int64 `json:"parent_id,omitempty"`
	CloneOf            *int64 `json:"clone_of,omitempty"`
	Type               string `json:"type"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
	Priority           int    `json:"priority"`
	EstimateEffort     int    `json:"estimate_effort"`
	EstimateComplete   string `json:"estimate_complete,omitempty"`
	Assignee           string `json:"assignee"`
}

type TaskUpdateRequest struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
	ParentID           *int64 `json:"parent_id,omitempty"`
	Assignee           string `json:"assignee"`
	Status             string `json:"status,omitempty"`
	Priority           int    `json:"priority"`
	Order              int    `json:"order"`
	EstimateEffort     int    `json:"estimate_effort"`
	EstimateComplete   string `json:"estimate_complete,omitempty"`
}

type CommentCreateRequest struct {
	Comment string `json:"comment"`
}

type DependencyRequest struct {
	ProjectID int64 `json:"project_id"`
	TaskID    int64 `json:"task_id"`
	DependsOn int64 `json:"depends_on"`
}

type TaskRequest struct {
	ProjectID int64  `json:"project_id,omitempty"`
	TaskID    *int64 `json:"task_id,omitempty"`
}

type TaskRequestResponse struct {
	Status string      `json:"status"`
	Task   *store.Task `json:"task,omitempty"`
}

func New(cfg config.Config) *Client {
	mode, err := config.ResolveMode()
	if err != nil {
		mode = config.ModeLocal
	}
	return &Client{
		baseURL: strings.TrimRight(config.ResolveServerURL(cfg), "/"),
		token:   cfg.Token,
		http:    http.DefaultClient,
		mode:    mode,
	}
}

func (c *Client) Register(username, password string) (store.User, error) {
	if c.mode == config.ModeLocal {
		return store.User{}, errors.New("ticket register requires TICKET_MODE=remote")
	}
	var user store.User
	err := c.doJSON(http.MethodPost, "/api/register", map[string]string{
		"username": username,
		"password": password,
	}, &user)
	return user, err
}

func (c *Client) Login(username, password string) (AuthResponse, error) {
	if c.mode == config.ModeLocal {
		return AuthResponse{}, errors.New("ticket login requires TICKET_MODE=remote")
	}
	var response AuthResponse
	err := c.doJSON(http.MethodPost, "/api/login", map[string]string{
		"username": username,
		"password": password,
	}, &response)
	return response, err
}

func (c *Client) Logout() error {
	if c.mode == config.ModeLocal {
		return errors.New("ticket logout requires TICKET_MODE=remote")
	}
	return c.doJSON(http.MethodPost, "/api/logout", nil, nil)
}

func (c *Client) Status() (StatusResponse, error) {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return StatusResponse{}, err
		}
		defer db.Close()

		username := localUsername()
		user, err := ensureLocalUser(db, username)
		if err != nil {
			return StatusResponse{}, err
		}
		return StatusResponse{
			Status:        "ok",
			Authenticated: true,
			User:          &user,
		}, nil
	}
	var status StatusResponse
	err := c.doJSON(http.MethodGet, "/api/status", nil, &status)
	return status, err
}

func (c *Client) Count(projectID *int64) (CountSummary, error) {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return CountSummary{}, err
		}
		defer db.Close()
		return store.CountEverything(db, projectID)
	}
	var summary CountSummary
	path := "/api/count"
	if projectID != nil {
		path = fmt.Sprintf("/api/count?project_id=%d", *projectID)
	}
	err := c.doJSON(http.MethodGet, path, nil, &summary)
	return summary, err
}

func (c *Client) CreateUser(username, password string) (store.User, error) {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return store.User{}, err
		}
		defer db.Close()
		return store.CreateUser(db, username, password, "user")
	}
	var user store.User
	err := c.doJSON(http.MethodPost, "/api/users", map[string]string{
		"username": username,
		"password": password,
	}, &user)
	return user, err
}

func (c *Client) SetUserEnabled(username string, enabled bool) error {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return err
		}
		defer db.Close()
		return store.SetUserEnabled(db, username, enabled)
	}
	action := "disable"
	if enabled {
		action = "enable"
	}
	return c.doJSON(http.MethodPost, "/api/users/"+username+"/"+action, nil, nil)
}

func (c *Client) ListUsers() ([]store.User, error) {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return nil, err
		}
		defer db.Close()
		return store.ListUsers(db)
	}
	var users []store.User
	err := c.doJSON(http.MethodGet, "/api/users", nil, &users)
	return users, err
}

func (c *Client) DeleteUser(username string) error {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return err
		}
		defer db.Close()
		return store.DeleteUser(db, username)
	}
	return c.doJSON(http.MethodDelete, "/api/users/"+username, nil, nil)
}

func (c *Client) CreateProject(title, description, acceptanceCriteria string) (store.Project, error) {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return store.Project{}, err
		}
		defer db.Close()
		user, err := c.localUser(db)
		if err != nil {
			return store.Project{}, err
		}
		return store.CreateProject(db, title, description, acceptanceCriteria, user.ID)
	}
	var project store.Project
	err := c.doJSON(http.MethodPost, "/api/projects", ProjectCreateRequest{
		Title:              title,
		Description:        description,
		AcceptanceCriteria: acceptanceCriteria,
	}, &project)
	return project, err
}

func (c *Client) ListProjects() ([]store.Project, error) {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return nil, err
		}
		defer db.Close()
		return store.ListProjects(db)
	}
	var projects []store.Project
	err := c.doJSON(http.MethodGet, "/api/projects", nil, &projects)
	return projects, err
}

func (c *Client) GetProject(id string) (store.Project, error) {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return store.Project{}, err
		}
		defer db.Close()
		return store.GetProject(db, id)
	}
	var project store.Project
	err := c.doJSON(http.MethodGet, "/api/projects/"+id, nil, &project)
	return project, err
}

func (c *Client) UpdateProject(id int64, request ProjectUpdateRequest) (store.Project, error) {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return store.Project{}, err
		}
		defer db.Close()
		return store.UpdateProject(db, id, request.Title, request.Description, request.AcceptanceCriteria)
	}
	var project store.Project
	err := c.doJSON(http.MethodPut, fmt.Sprintf("/api/projects/%d", id), request, &project)
	return project, err
}

func (c *Client) SetProjectEnabled(id int64, enabled bool) (store.Project, error) {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return store.Project{}, err
		}
		defer db.Close()
		return store.SetProjectStatus(db, id, enabled)
	}
	action := "disable"
	if enabled {
		action = "enable"
	}
	var project store.Project
	err := c.doJSON(http.MethodPost, fmt.Sprintf("/api/projects/%d/%s", id, action), nil, &project)
	return project, err
}

func (c *Client) CreateTask(request TaskCreateRequest) (store.Task, error) {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return store.Task{}, err
		}
		defer db.Close()
		user, err := c.localUser(db)
		if err != nil {
			return store.Task{}, err
		}
		return store.CreateTask(db, store.TaskCreateParams{
			ProjectID:          request.ProjectID,
			ParentID:           request.ParentID,
			CloneOf:            request.CloneOf,
			Type:               request.Type,
			Title:              request.Title,
			Description:        request.Description,
			AcceptanceCriteria: request.AcceptanceCriteria,
			Priority:           request.Priority,
			EstimateEffort:     request.EstimateEffort,
			EstimateComplete:   request.EstimateComplete,
			Assignee:           request.Assignee,
			CreatedBy:          user.ID,
		})
	}
	var task store.Task
	err := c.doJSON(http.MethodPost, "/api/tasks", request, &task)
	return task, err
}

func (c *Client) ListTasks(projectID int64) ([]store.Task, error) {
	return c.ListTasksFiltered(projectID, "", "", "", "", 0)
}

func (c *Client) ListTasksFiltered(projectID int64, taskType, status, search, assignee string, limit int) ([]store.Task, error) {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return nil, err
		}
		defer db.Close()
		return store.ListTasks(db, store.TaskListParams{
			ProjectID: projectID,
			Type:      taskType,
			Status:    status,
			Search:    search,
			Assignee:  assignee,
			Limit:     limit,
		})
	}
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

func (c *Client) UpdateTask(id int64, request TaskUpdateRequest) (store.Task, error) {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return store.Task{}, err
		}
		defer db.Close()
		user, err := c.localUser(db)
		if err != nil {
			return store.Task{}, err
		}
		return store.UpdateTask(db, id, store.TaskUpdateParams{
			Title:              request.Title,
			Description:        request.Description,
			AcceptanceCriteria: request.AcceptanceCriteria,
			ParentID:           request.ParentID,
			Assignee:           request.Assignee,
			Status:             request.Status,
			Priority:           request.Priority,
			Order:              request.Order,
			EstimateEffort:     request.EstimateEffort,
			EstimateComplete:   request.EstimateComplete,
			UpdatedBy:          user.ID,
			ActorUsername:      user.Username,
			// Local mode bypasses server-side ownership restrictions.
			ActorRole: "admin",
		})
	}
	var task store.Task
	err := c.doJSON(http.MethodPut, fmt.Sprintf("/api/tasks/%d", id), request, &task)
	return task, err
}

func (c *Client) DeleteTask(id int64) error {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return err
		}
		defer db.Close()
		return store.DeleteTask(db, id)
	}
	return c.doJSON(http.MethodDelete, fmt.Sprintf("/api/tasks/%d", id), nil, nil)
}

func (c *Client) SetTaskParent(id, parentID int64) (store.Task, error) {
	current, err := c.GetTask(id)
	if err != nil {
		return store.Task{}, err
	}
	return c.UpdateTask(id, TaskUpdateRequest{
		Title:              current.Title,
		Description:        current.Description,
		AcceptanceCriteria: current.AcceptanceCriteria,
		ParentID:           &parentID,
		Assignee:           current.Assignee,
		Status:             current.Status,
		Priority:           current.Priority,
		Order:              current.Order,
		EstimateEffort:     current.EstimateEffort,
		EstimateComplete:   current.EstimateComplete,
	})
}

func (c *Client) UnsetTaskParent(id int64) (store.Task, error) {
	current, err := c.GetTask(id)
	if err != nil {
		return store.Task{}, err
	}
	return c.UpdateTask(id, TaskUpdateRequest{
		Title:              current.Title,
		Description:        current.Description,
		AcceptanceCriteria: current.AcceptanceCriteria,
		ParentID:           nil,
		Assignee:           current.Assignee,
		Status:             current.Status,
		Priority:           current.Priority,
		Order:              current.Order,
		EstimateEffort:     current.EstimateEffort,
		EstimateComplete:   current.EstimateComplete,
	})
}

func (c *Client) GetTask(id int64) (store.Task, error) {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return store.Task{}, err
		}
		defer db.Close()
		return store.GetTask(db, id)
	}
	var task store.Task
	err := c.doJSON(http.MethodGet, fmt.Sprintf("/api/tasks/%d", id), nil, &task)
	return task, err
}

func (c *Client) CloneTask(id int64) (store.Task, error) {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return store.Task{}, err
		}
		defer db.Close()
		user, err := c.localUser(db)
		if err != nil {
			return store.Task{}, err
		}
		return store.CloneTask(db, id, user.ID)
	}
	var task store.Task
	err := c.doJSON(http.MethodPost, fmt.Sprintf("/api/tasks/%d/clone", id), nil, &task)
	return task, err
}

func (c *Client) ListHistory(id int64) ([]store.HistoryEvent, error) {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return nil, err
		}
		defer db.Close()
		return store.ListHistoryEvents(db, id)
	}
	var events []store.HistoryEvent
	err := c.doJSON(http.MethodGet, fmt.Sprintf("/api/tasks/%d/history", id), nil, &events)
	return events, err
}

func (c *Client) AddComment(id int64, comment string) (store.Comment, error) {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return store.Comment{}, err
		}
		defer db.Close()
		user, err := c.localUser(db)
		if err != nil {
			return store.Comment{}, err
		}
		return store.AddComment(db, id, user.ID, comment)
	}
	var created store.Comment
	err := c.doJSON(http.MethodPost, fmt.Sprintf("/api/tasks/%d/comments", id), CommentCreateRequest{Comment: comment}, &created)
	return created, err
}

func (c *Client) ListComments(id int64) ([]store.Comment, error) {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return nil, err
		}
		defer db.Close()
		return store.ListComments(db, id)
	}
	var comments []store.Comment
	err := c.doJSON(http.MethodGet, fmt.Sprintf("/api/tasks/%d/comments", id), nil, &comments)
	return comments, err
}

func (c *Client) AddDependency(request DependencyRequest) (store.Dependency, error) {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return store.Dependency{}, err
		}
		defer db.Close()
		user, err := c.localUser(db)
		if err != nil {
			return store.Dependency{}, err
		}
		return store.AddDependency(db, request.ProjectID, request.TaskID, request.DependsOn, user.ID)
	}
	var dependency store.Dependency
	err := c.doJSON(http.MethodPost, "/api/dependencies", request, &dependency)
	return dependency, err
}

func (c *Client) RemoveDependency(request DependencyRequest) error {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return err
		}
		defer db.Close()
		return store.DeleteDependency(db, request.ProjectID, request.TaskID, request.DependsOn)
	}
	return c.doJSON(http.MethodDelete, fmt.Sprintf("/api/dependencies?project_id=%d&task_id=%d&depends_on=%d", request.ProjectID, request.TaskID, request.DependsOn), nil, nil)
}

func (c *Client) ListDependencies(id int64) ([]store.Dependency, error) {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return nil, err
		}
		defer db.Close()
		return store.ListDependencies(db, id)
	}
	var dependencies []store.Dependency
	err := c.doJSON(http.MethodGet, fmt.Sprintf("/api/tasks/%d/dependencies", id), nil, &dependencies)
	return dependencies, err
}

func (c *Client) RequestTask(request TaskRequest) (TaskRequestResponse, error) {
	if c.mode == config.ModeLocal {
		db, err := c.openLocalDB()
		if err != nil {
			return TaskRequestResponse{}, err
		}
		defer db.Close()
		user, err := c.localUser(db)
		if err != nil {
			return TaskRequestResponse{}, err
		}
		task, status, err := store.RequestTask(db, store.TaskRequestParams{
			ProjectID: request.ProjectID,
			TaskID:    request.TaskID,
			Username:  user.Username,
			UserID:    user.ID,
		})
		if err != nil {
			return TaskRequestResponse{}, err
		}
		response := TaskRequestResponse{Status: status}
		if status == "ASSIGNED" {
			response.Task = &task
		}
		return response, nil
	}
	var reader *bytes.Reader
	payload, err := json.Marshal(request)
	if err != nil {
		return TaskRequestResponse{}, err
	}
	reader = bytes.NewReader(payload)

	httpReq, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/tasks/request", reader)
	if err != nil {
		return TaskRequestResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return TaskRequestResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Error != "" {
			return TaskRequestResponse{}, errors.New(apiErr.Error)
		}
		return TaskRequestResponse{}, fmt.Errorf("request failed with status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TaskRequestResponse{}, err
	}
	var response TaskRequestResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return TaskRequestResponse{}, err
	}
	return response, nil
}

func (c *Client) openLocalDB() (*sql.DB, error) {
	path, err := config.ResolveDatabasePath()
	if err != nil {
		return nil, err
	}
	return store.Open(path)
}

func (c *Client) localUser(db *sql.DB) (store.User, error) {
	return ensureLocalUser(db, localUsername())
}

func ensureLocalUser(db *sql.DB, username string) (store.User, error) {
	if user, err := store.GetUserByUsername(db, username); err == nil {
		if user.Enabled {
			return user, nil
		}
		if err := store.SetUserEnabled(db, username, true); err != nil {
			return store.User{}, err
		}
		return store.GetUserByUsername(db, username)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return store.User{}, err
	}
	user, err := store.CreateUser(db, username, "local-mode", "admin")
	if err != nil {
		return store.User{}, err
	}
	return user, nil
}

func localUsername() string {
	user, err := osuser.Current()
	if err == nil && strings.TrimSpace(user.Username) != "" {
		parts := strings.Split(user.Username, `\`)
		return parts[len(parts)-1]
	}
	if env := strings.TrimSpace(getenvFirst("USER", "USERNAME")); env != "" {
		return env
	}
	return "user"
}

func getenvFirst(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
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

	httpRequest, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		httpRequest.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(httpRequest)
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
