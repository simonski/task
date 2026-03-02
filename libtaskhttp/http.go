package libtaskhttp

import (
	"github.com/simonski/task/internal/client"
	"github.com/simonski/task/internal/config"
	"github.com/simonski/task/internal/store"
	"github.com/simonski/task/libtask"
)

type Service struct {
	client *client.Client
}

func New(cfg config.Config) *Service {
	return &Service{client: client.New(cfg)}
}

func (s *Service) Status() (libtask.StatusResponse, error) {
	status, err := s.client.Status()
	if err != nil {
		return libtask.StatusResponse{}, err
	}
	return libtask.StatusResponse(status), nil
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

func (s *Service) Count(projectID *int64) (libtask.CountSummary, error) {
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

func (s *Service) CreateProject(req libtask.ProjectCreateRequest) (store.Project, error) {
	return s.client.CreateProject(req.Title, req.Description, req.AcceptanceCriteria)
}

func (s *Service) ListProjects() ([]store.Project, error) {
	return s.client.ListProjects()
}

func (s *Service) GetProject(id string) (store.Project, error) {
	return s.client.GetProject(id)
}

func (s *Service) UpdateProject(id int64, req libtask.ProjectUpdateRequest) (store.Project, error) {
	return s.client.UpdateProject(id, client.ProjectUpdateRequest(req))
}

func (s *Service) SetProjectEnabled(id int64, enabled bool) (store.Project, error) {
	return s.client.SetProjectEnabled(id, enabled)
}

func (s *Service) CreateTask(req libtask.TaskCreateRequest) (store.Task, error) {
	return s.client.CreateTask(client.TaskCreateRequest(req))
}

func (s *Service) ListTasks(projectID int64) ([]store.Task, error) {
	return s.client.ListTasks(projectID)
}

func (s *Service) ListTasksFiltered(projectID int64, taskType, status, search, assignee string, limit int) ([]store.Task, error) {
	return s.client.ListTasksFiltered(projectID, taskType, status, search, assignee, limit)
}

func (s *Service) UpdateTask(id int64, req libtask.TaskUpdateRequest) (store.Task, error) {
	return s.client.UpdateTask(id, client.TaskUpdateRequest(req))
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

func (s *Service) AddDependency(req libtask.DependencyRequest) (store.Dependency, error) {
	return s.client.AddDependency(client.DependencyRequest(req))
}

func (s *Service) RemoveDependency(req libtask.DependencyRequest) error {
	return s.client.RemoveDependency(client.DependencyRequest(req))
}

func (s *Service) ListDependencies(id int64) ([]store.Dependency, error) {
	return s.client.ListDependencies(id)
}

func (s *Service) RequestTask(req libtask.TaskRequest) (libtask.TaskRequestResponse, error) {
	response, err := s.client.RequestTask(client.TaskRequest(req))
	if err != nil {
		return libtask.TaskRequestResponse{}, err
	}
	return libtask.TaskRequestResponse(response), nil
}
