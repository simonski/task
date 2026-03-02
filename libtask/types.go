package libtask

import "github.com/simonski/task/internal/store"

type StatusResponse struct {
	Status        string      `json:"status"`
	Authenticated bool        `json:"authenticated"`
	ServerVersion string      `json:"server_version,omitempty"`
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
