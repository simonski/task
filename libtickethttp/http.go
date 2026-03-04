package libtickethttp

import (
	"github.com/simonski/ticket/internal/client"
	"github.com/simonski/ticket/internal/config"
	"github.com/simonski/ticket/internal/store"
	"github.com/simonski/ticket/libticket"
)

type Service struct {
	client *client.Client
}

func New(cfg config.Config) *Service {
	return &Service{client: client.New(cfg)}
}

func (s *Service) Status() (libticket.StatusResponse, error) {
	status, err := s.client.Status()
	if err != nil {
		return libticket.StatusResponse{}, err
	}
	return libticket.StatusResponse(status), nil
}

func (s *Service) Register(username, password string) (store.User, error) {
	return s.client.Register(username, password)
}

func (s *Service) Login(username, password string) (store.User, string, error) {
	response, err := s.client.Login(username, password)
	if err != nil {
		return store.User{}, "", err
	}
	return response.User, response.Token, nil
}

func (s *Service) Logout() error {
	return s.client.Logout()
}

func (s *Service) Count(projectID *int64) (libticket.CountSummary, error) {
	return s.client.Count(projectID)
}

func (s *Service) CreateUser(username, password string) (store.User, error) {
	return s.client.CreateUser(username, password)
}

func (s *Service) SetUserEnabled(username string, enabled bool) error {
	return s.client.SetUserEnabled(username, enabled)
}

func (s *Service) ListUsers() ([]store.User, error) {
	return s.client.ListUsers()
}

func (s *Service) DeleteUser(username string) error {
	return s.client.DeleteUser(username)
}

func (s *Service) CreateProject(request libticket.ProjectCreateRequest) (store.Project, error) {
	return s.client.CreateProject(client.ProjectCreateRequest(request))
}

func (s *Service) ListProjects() ([]store.Project, error) {
	return s.client.ListProjects()
}

func (s *Service) GetProject(id string) (store.Project, error) {
	return s.client.GetProject(id)
}

func (s *Service) UpdateProject(id int64, request libticket.ProjectUpdateRequest) (store.Project, error) {
	return s.client.UpdateProject(id, client.ProjectUpdateRequest(request))
}

func (s *Service) SetProjectEnabled(id int64, enabled bool) (store.Project, error) {
	return s.client.SetProjectEnabled(id, enabled)
}

func (s *Service) CreateTask(request libticket.TaskCreateRequest) (store.Task, error) {
	return s.client.CreateTask(client.TaskCreateRequest(request))
}

func (s *Service) ListTasks(projectID int64) ([]store.Task, error) {
	return s.client.ListTasks(projectID)
}

func (s *Service) ListTasksFiltered(projectID int64, taskType, stage, state, status, search, assignee string, limit int) ([]store.Task, error) {
	return s.client.ListTasksFiltered(projectID, taskType, stage, state, status, search, assignee, limit)
}

func (s *Service) UpdateTask(id int64, request libticket.TaskUpdateRequest) (store.Task, error) {
	return s.client.UpdateTask(id, client.TaskUpdateRequest(request))
}

func (s *Service) DeleteTask(id int64) error {
	return s.client.DeleteTask(id)
}

func (s *Service) SetTaskParent(id, parentID int64) (store.Task, error) {
	return s.client.SetTaskParent(id, parentID)
}

func (s *Service) UnsetTaskParent(id int64) (store.Task, error) {
	return s.client.UnsetTaskParent(id)
}

func (s *Service) GetTask(id int64) (store.Task, error) {
	return s.client.GetTask(id)
}

func (s *Service) CloneTask(id int64) (store.Task, error) {
	return s.client.CloneTask(id)
}

func (s *Service) ListHistory(id int64) ([]store.HistoryEvent, error) {
	return s.client.ListHistory(id)
}

func (s *Service) AddComment(id int64, comment string) (store.Comment, error) {
	return s.client.AddComment(id, comment)
}

func (s *Service) ListComments(id int64) ([]store.Comment, error) {
	return s.client.ListComments(id)
}

func (s *Service) AddDependency(request libticket.DependencyRequest) (store.Dependency, error) {
	return s.client.AddDependency(client.DependencyRequest(request))
}

func (s *Service) RemoveDependency(request libticket.DependencyRequest) error {
	return s.client.RemoveDependency(client.DependencyRequest(request))
}

func (s *Service) ListDependencies(id int64) ([]store.Dependency, error) {
	return s.client.ListDependencies(id)
}

func (s *Service) RequestTask(request libticket.TaskRequest) (libticket.TaskRequestResponse, error) {
	response, err := s.client.RequestTask(client.TaskRequest(request))
	if err != nil {
		return libticket.TaskRequestResponse{}, err
	}
	return libticket.TaskRequestResponse(response), nil
}
