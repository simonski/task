package libtask

import (
	"database/sql"
	"errors"
	"os"
	osuser "os/user"
	"strings"

	"github.com/simonski/task/internal/config"
	"github.com/simonski/task/internal/store"
)

type LocalService struct {
	cfg config.Config
}

func NewLocal(cfg config.Config) *LocalService {
	return &LocalService{cfg: cfg}
}

func (s *LocalService) Status() (StatusResponse, error) {
	db, err := s.openDB()
	if err != nil {
		return StatusResponse{}, err
	}
	defer db.Close()
	user, err := s.localUser(db)
	if err != nil {
		return StatusResponse{}, err
	}
	return StatusResponse{
		Status:        "ok",
		Authenticated: true,
		User:          &user,
	}, nil
}

func (s *LocalService) Register(username, password string) (store.User, error) {
	return store.User{}, errors.New("task register requires TASK_MODE=remote")
}

func (s *LocalService) Login(username, password string) (store.User, string, error) {
	return store.User{}, "", errors.New("task login requires TASK_MODE=remote")
}

func (s *LocalService) Logout() error {
	return errors.New("task logout requires TASK_MODE=remote")
}

func (s *LocalService) Count(projectID *int64) (CountSummary, error) {
	db, err := s.openDB()
	if err != nil {
		return CountSummary{}, err
	}
	defer db.Close()
	return store.CountEverything(db, projectID)
}

func (s *LocalService) CreateUser(username, password string) (store.User, error) {
	db, err := s.openDB()
	if err != nil {
		return store.User{}, err
	}
	defer db.Close()
	return store.CreateUser(db, username, password, "user")
}

func (s *LocalService) SetUserEnabled(username string, enabled bool) error {
	db, err := s.openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	return store.SetUserEnabled(db, username, enabled)
}

func (s *LocalService) ListUsers() ([]store.User, error) {
	db, err := s.openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return store.ListUsers(db)
}

func (s *LocalService) DeleteUser(username string) error {
	db, err := s.openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	return store.DeleteUser(db, username)
}

func (s *LocalService) CreateProject(req ProjectCreateRequest) (store.Project, error) {
	db, err := s.openDB()
	if err != nil {
		return store.Project{}, err
	}
	defer db.Close()
	user, err := s.localUser(db)
	if err != nil {
		return store.Project{}, err
	}
	return store.CreateProject(db, req.Title, req.Description, req.AcceptanceCriteria, user.ID)
}

func (s *LocalService) ListProjects() ([]store.Project, error) {
	db, err := s.openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return store.ListProjects(db)
}

func (s *LocalService) GetProject(id string) (store.Project, error) {
	db, err := s.openDB()
	if err != nil {
		return store.Project{}, err
	}
	defer db.Close()
	return store.GetProject(db, id)
}

func (s *LocalService) UpdateProject(id int64, req ProjectUpdateRequest) (store.Project, error) {
	db, err := s.openDB()
	if err != nil {
		return store.Project{}, err
	}
	defer db.Close()
	return store.UpdateProject(db, id, req.Title, req.Description, req.AcceptanceCriteria)
}

func (s *LocalService) SetProjectEnabled(id int64, enabled bool) (store.Project, error) {
	db, err := s.openDB()
	if err != nil {
		return store.Project{}, err
	}
	defer db.Close()
	return store.SetProjectStatus(db, id, enabled)
}

func (s *LocalService) CreateTask(req TaskCreateRequest) (store.Task, error) {
	db, err := s.openDB()
	if err != nil {
		return store.Task{}, err
	}
	defer db.Close()
	user, err := s.localUser(db)
	if err != nil {
		return store.Task{}, err
	}
	return store.CreateTask(db, store.TaskCreateParams{
		ProjectID:          req.ProjectID,
		ParentID:           req.ParentID,
		CloneOf:            req.CloneOf,
		Type:               req.Type,
		Title:              req.Title,
		Description:        req.Description,
		AcceptanceCriteria: req.AcceptanceCriteria,
		Priority:           req.Priority,
		Assignee:           req.Assignee,
		CreatedBy:          user.ID,
	})
}

func (s *LocalService) ListTasks(projectID int64) ([]store.Task, error) {
	return s.ListTasksFiltered(projectID, "", "", "", "", 0)
}

func (s *LocalService) ListTasksFiltered(projectID int64, taskType, status, search, assignee string, limit int) ([]store.Task, error) {
	db, err := s.openDB()
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

func (s *LocalService) UpdateTask(id int64, req TaskUpdateRequest) (store.Task, error) {
	db, err := s.openDB()
	if err != nil {
		return store.Task{}, err
	}
	defer db.Close()
	user, err := s.localUser(db)
	if err != nil {
		return store.Task{}, err
	}
	return store.UpdateTask(db, id, store.TaskUpdateParams{
		Title:         req.Title,
		Description:   req.Description,
		ParentID:      req.ParentID,
		Assignee:      req.Assignee,
		Status:        req.Status,
		UpdatedBy:     user.ID,
		ActorUsername: user.Username,
		ActorRole:     user.Role,
	})
}

func (s *LocalService) SetTaskParent(id, parentID int64) (store.Task, error) {
	current, err := s.GetTask(id)
	if err != nil {
		return store.Task{}, err
	}
	return s.UpdateTask(id, TaskUpdateRequest{
		Title:       current.Title,
		Description: current.Description,
		ParentID:    &parentID,
		Assignee:    current.Assignee,
		Status:      current.Status,
	})
}

func (s *LocalService) UnsetTaskParent(id int64) (store.Task, error) {
	current, err := s.GetTask(id)
	if err != nil {
		return store.Task{}, err
	}
	return s.UpdateTask(id, TaskUpdateRequest{
		Title:       current.Title,
		Description: current.Description,
		ParentID:    nil,
		Assignee:    current.Assignee,
		Status:      current.Status,
	})
}

func (s *LocalService) GetTask(id int64) (store.Task, error) {
	db, err := s.openDB()
	if err != nil {
		return store.Task{}, err
	}
	defer db.Close()
	return store.GetTask(db, id)
}

func (s *LocalService) CloneTask(id int64) (store.Task, error) {
	db, err := s.openDB()
	if err != nil {
		return store.Task{}, err
	}
	defer db.Close()
	user, err := s.localUser(db)
	if err != nil {
		return store.Task{}, err
	}
	return store.CloneTask(db, id, user.ID)
}

func (s *LocalService) ListHistory(id int64) ([]store.HistoryEvent, error) {
	db, err := s.openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return store.ListHistoryEvents(db, id)
}

func (s *LocalService) AddComment(id int64, comment string) (store.Comment, error) {
	db, err := s.openDB()
	if err != nil {
		return store.Comment{}, err
	}
	defer db.Close()
	user, err := s.localUser(db)
	if err != nil {
		return store.Comment{}, err
	}
	return store.AddComment(db, id, user.ID, comment)
}

func (s *LocalService) ListComments(id int64) ([]store.Comment, error) {
	db, err := s.openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return store.ListComments(db, id)
}

func (s *LocalService) AddDependency(req DependencyRequest) (store.Dependency, error) {
	db, err := s.openDB()
	if err != nil {
		return store.Dependency{}, err
	}
	defer db.Close()
	user, err := s.localUser(db)
	if err != nil {
		return store.Dependency{}, err
	}
	return store.AddDependency(db, req.ProjectID, req.TaskID, req.DependsOn, user.ID)
}

func (s *LocalService) RemoveDependency(req DependencyRequest) error {
	db, err := s.openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	return store.DeleteDependency(db, req.ProjectID, req.TaskID, req.DependsOn)
}

func (s *LocalService) ListDependencies(id int64) ([]store.Dependency, error) {
	db, err := s.openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return store.ListDependencies(db, id)
}

func (s *LocalService) RequestTask(req TaskRequest) (TaskRequestResponse, error) {
	db, err := s.openDB()
	if err != nil {
		return TaskRequestResponse{}, err
	}
	defer db.Close()
	user, err := s.localUser(db)
	if err != nil {
		return TaskRequestResponse{}, err
	}
	task, status, err := store.RequestTask(db, store.TaskRequestParams{
		ProjectID: req.ProjectID,
		TaskID:    req.TaskID,
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

func (s *LocalService) openDB() (*sql.DB, error) {
	path, err := config.ResolveDatabasePath()
	if err != nil {
		return nil, err
	}
	return store.Open(path)
}

func (s *LocalService) localUser(db *sql.DB) (store.User, error) {
	username := LocalUsername()
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
	return store.CreateUser(db, username, "local-mode", "admin")
}

func LocalUsername() string {
	user, err := osuser.Current()
	if err == nil && strings.TrimSpace(user.Username) != "" {
		parts := strings.Split(user.Username, `\`)
		return parts[len(parts)-1]
	}
	if value := strings.TrimSpace(os.Getenv("USER")); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("USERNAME")); value != "" {
		return value
	}
	return "user"
}
