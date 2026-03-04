package libticket

import (
	"database/sql"
	"errors"
	"os"
	osuser "os/user"
	"strings"

	"github.com/simonski/ticket/internal/config"
	"github.com/simonski/ticket/internal/store"
)

func resolveRequestLifecycle(status, stage, state string) (string, string, error) {
	if strings.TrimSpace(stage) != "" || strings.TrimSpace(state) != "" {
		return stage, state, nil
	}
	if strings.TrimSpace(status) == "" {
		return stage, state, nil
	}
	return store.ParseLifecycleStatus(status)
}

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
	return store.User{}, errors.New("ticket register requires TICKET_MODE=remote")
}

func (s *LocalService) Login(username, password string) (store.User, string, error) {
	return store.User{}, "", errors.New("ticket login requires TICKET_MODE=remote")
}

func (s *LocalService) Logout() error {
	return errors.New("ticket logout requires TICKET_MODE=remote")
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

func (s *LocalService) CreateProject(request ProjectCreateRequest) (store.Project, error) {
	db, err := s.openDB()
	if err != nil {
		return store.Project{}, err
	}
	defer db.Close()
	user, err := s.localUser(db)
	if err != nil {
		return store.Project{}, err
	}
	return store.CreateProject(db, request.Title, request.Description, request.AcceptanceCriteria, user.ID)
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

func (s *LocalService) UpdateProject(id int64, request ProjectUpdateRequest) (store.Project, error) {
	db, err := s.openDB()
	if err != nil {
		return store.Project{}, err
	}
	defer db.Close()
	return store.UpdateProject(db, id, request.Title, request.Description, request.AcceptanceCriteria)
}

func (s *LocalService) SetProjectEnabled(id int64, enabled bool) (store.Project, error) {
	db, err := s.openDB()
	if err != nil {
		return store.Project{}, err
	}
	defer db.Close()
	return store.SetProjectStatus(db, id, enabled)
}

func (s *LocalService) CreateTask(request TaskCreateRequest) (store.Task, error) {
	db, err := s.openDB()
	if err != nil {
		return store.Task{}, err
	}
	defer db.Close()
	user, err := s.localUser(db)
	if err != nil {
		return store.Task{}, err
	}
	stage, state, err := resolveRequestLifecycle(request.Status, request.Stage, request.State)
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
		Stage:              stage,
		State:              state,
		CreatedBy:          user.ID,
	})
}

func (s *LocalService) ListTasks(projectID int64) ([]store.Task, error) {
	return s.ListTasksFiltered(projectID, "", "", "", "", "", "", 0)
}

func (s *LocalService) ListTasksFiltered(projectID int64, taskType, stage, state, status, search, assignee string, limit int) ([]store.Task, error) {
	db, err := s.openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return store.ListTasks(db, store.TaskListParams{
		ProjectID: projectID,
		Type:      taskType,
		Stage:     stage,
		State:     state,
		Status:    status,
		Search:    search,
		Assignee:  assignee,
		Limit:     limit,
	})
}

func (s *LocalService) UpdateTask(id int64, request TaskUpdateRequest) (store.Task, error) {
	db, err := s.openDB()
	if err != nil {
		return store.Task{}, err
	}
	defer db.Close()
	user, err := s.localUser(db)
	if err != nil {
		return store.Task{}, err
	}
	stage, state, err := resolveRequestLifecycle(request.Status, request.Stage, request.State)
	if err != nil {
		return store.Task{}, err
	}
	return store.UpdateTask(db, id, store.TaskUpdateParams{
		Title:              request.Title,
		Description:        request.Description,
		AcceptanceCriteria: request.AcceptanceCriteria,
		ParentID:           request.ParentID,
		Assignee:           request.Assignee,
		Stage:              stage,
		State:              state,
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

func (s *LocalService) DeleteTask(id int64) error {
	db, err := s.openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	return store.DeleteTask(db, id)
}

func (s *LocalService) SetTaskParent(id, parentID int64) (store.Task, error) {
	current, err := s.GetTask(id)
	if err != nil {
		return store.Task{}, err
	}
	return s.UpdateTask(id, TaskUpdateRequest{
		Title:              current.Title,
		Description:        current.Description,
		AcceptanceCriteria: current.AcceptanceCriteria,
		ParentID:           &parentID,
		Assignee:           current.Assignee,
		Stage:              current.Stage,
		State:              current.State,
		Priority:           current.Priority,
		Order:              current.Order,
		EstimateEffort:     current.EstimateEffort,
		EstimateComplete:   current.EstimateComplete,
	})
}

func (s *LocalService) UnsetTaskParent(id int64) (store.Task, error) {
	current, err := s.GetTask(id)
	if err != nil {
		return store.Task{}, err
	}
	return s.UpdateTask(id, TaskUpdateRequest{
		Title:              current.Title,
		Description:        current.Description,
		AcceptanceCriteria: current.AcceptanceCriteria,
		ParentID:           nil,
		Assignee:           current.Assignee,
		Stage:              current.Stage,
		State:              current.State,
		Priority:           current.Priority,
		Order:              current.Order,
		EstimateEffort:     current.EstimateEffort,
		EstimateComplete:   current.EstimateComplete,
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

func (s *LocalService) AddDependency(request DependencyRequest) (store.Dependency, error) {
	db, err := s.openDB()
	if err != nil {
		return store.Dependency{}, err
	}
	defer db.Close()
	user, err := s.localUser(db)
	if err != nil {
		return store.Dependency{}, err
	}
	return store.AddDependency(db, request.ProjectID, request.TaskID, request.DependsOn, user.ID)
}

func (s *LocalService) RemoveDependency(request DependencyRequest) error {
	db, err := s.openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	return store.DeleteDependency(db, request.ProjectID, request.TaskID, request.DependsOn)
}

func (s *LocalService) ListDependencies(id int64) ([]store.Dependency, error) {
	db, err := s.openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return store.ListDependencies(db, id)
}

func (s *LocalService) RequestTask(request TaskRequest) (TaskRequestResponse, error) {
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
