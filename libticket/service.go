package libticket

import "github.com/simonski/ticket/internal/store"

type Service interface {
	Status() (StatusResponse, error)
	Register(username, password string) (store.User, error)
	Login(username, password string) (store.User, string, error)
	Logout() error
	Count(projectID *int64) (CountSummary, error)
	CreateUser(username, password string) (store.User, error)
	SetUserEnabled(username string, enabled bool) error
	ListUsers() ([]store.User, error)
	DeleteUser(username string) error
	CreateProject(req ProjectCreateRequest) (store.Project, error)
	ListProjects() ([]store.Project, error)
	GetProject(id string) (store.Project, error)
	UpdateProject(id int64, req ProjectUpdateRequest) (store.Project, error)
	SetProjectEnabled(id int64, enabled bool) (store.Project, error)
	CreateTask(req TaskCreateRequest) (store.Task, error)
	ListTasks(projectID int64) ([]store.Task, error)
	ListTasksFiltered(projectID int64, taskType, status, search, assignee string, limit int) ([]store.Task, error)
	UpdateTask(id int64, req TaskUpdateRequest) (store.Task, error)
	DeleteTask(id int64) error
	SetTaskParent(id, parentID int64) (store.Task, error)
	UnsetTaskParent(id int64) (store.Task, error)
	GetTask(id int64) (store.Task, error)
	CloneTask(id int64) (store.Task, error)
	ListHistory(id int64) ([]store.HistoryEvent, error)
	AddComment(id int64, comment string) (store.Comment, error)
	ListComments(id int64) ([]store.Comment, error)
	AddDependency(req DependencyRequest) (store.Dependency, error)
	RemoveDependency(req DependencyRequest) error
	ListDependencies(id int64) ([]store.Dependency, error)
	RequestTask(req TaskRequest) (TaskRequestResponse, error)
}
