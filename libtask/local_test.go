package libtask_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/simonski/task/internal/config"
	"github.com/simonski/task/internal/store"
	"github.com/simonski/task/libtask"
	"github.com/simonski/task/libtasktest"
)

func TestLocalServiceContract(t *testing.T) {
	libtasktest.RunServiceContractTests(t, func(t *testing.T) libtask.Service {
		tempDir := t.TempDir()
		t.Setenv("TASK_MODE", "local")
		t.Setenv("TASK_HOME", tempDir)
		dbPath := filepath.Join(tempDir, "task.db")
		if err := store.Init(dbPath, "admin", "secret"); err != nil {
			t.Fatalf("store.Init() error = %v", err)
		}
		return libtask.NewLocal(config.Config{})
	})
}

func TestLocalServiceStatusCreatesLocalUser(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TASK_MODE", "local")
	t.Setenv("TASK_HOME", tempDir)
	dbPath := filepath.Join(tempDir, "task.db")
	if err := store.Init(dbPath, "admin", "secret"); err != nil {
		t.Fatalf("store.Init() error = %v", err)
	}

	svc := libtask.NewLocal(config.Config{})
	status, err := svc.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !status.Authenticated || status.User == nil {
		t.Fatalf("Status() = %#v", status)
	}
	if status.User.Username != libtask.LocalUsername() {
		t.Fatalf("Status().User.Username = %q, want %q", status.User.Username, libtask.LocalUsername())
	}
}

func TestLocalServiceRemoteAuthCommandsFail(t *testing.T) {
	svc := libtask.NewLocal(config.Config{})

	if _, err := svc.Register("alice", "secret"); err == nil {
		t.Fatal("Register() error = nil, want remote-mode error")
	}
	if _, _, err := svc.Login("alice", "secret"); err == nil {
		t.Fatal("Login() error = nil, want remote-mode error")
	}
	if err := svc.Logout(); err == nil {
		t.Fatal("Logout() error = nil, want remote-mode error")
	}
}

func TestLocalServiceStatusCreatesDatabaseWhenMissing(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TASK_MODE", "local")
	t.Setenv("TASK_HOME", tempDir)
	dbPath := filepath.Join(tempDir, "task.db")

	svc := libtask.NewLocal(config.Config{})
	if _, err := svc.Status(); err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("Status() should create/open local db at %s: %v", dbPath, err)
	}
}

func TestLocalUsernameUsesEnvironmentFallbacks(t *testing.T) {
	t.Setenv("USER", "env-user")
	t.Setenv("USERNAME", "env-username")

	got := libtask.LocalUsername()
	if got == "" {
		t.Fatal("LocalUsername() returned empty username")
	}
}

func TestLocalServiceUsesTaskHomeDatabasePath(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TASK_MODE", "local")
	t.Setenv("TASK_HOME", tempDir)

	dbPath := filepath.Join(tempDir, "task.db")
	if err := store.Init(dbPath, "admin", "secret"); err != nil {
		t.Fatalf("store.Init() error = %v", err)
	}

	svc := libtask.NewLocal(config.Config{})
	projects, err := svc.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) == 0 {
		t.Fatal("ListProjects() returned no projects")
	}
}

func TestLocalServiceSetTaskParent(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TASK_MODE", "local")
	t.Setenv("TASK_HOME", tempDir)
	dbPath := filepath.Join(tempDir, "task.db")
	if err := store.Init(dbPath, "admin", "secret"); err != nil {
		t.Fatalf("store.Init() error = %v", err)
	}

	svc := libtask.NewLocal(config.Config{})
	parent, err := svc.CreateTask(libtask.TaskCreateRequest{ProjectID: 1, Type: "epic", Title: "Parent"})
	if err != nil {
		t.Fatalf("CreateTask(parent) error = %v", err)
	}
	child, err := svc.CreateTask(libtask.TaskCreateRequest{ProjectID: 1, Type: "task", Title: "Child"})
	if err != nil {
		t.Fatalf("CreateTask(child) error = %v", err)
	}

	updated, err := svc.SetTaskParent(child.ID, parent.ID)
	if err != nil {
		t.Fatalf("SetTaskParent() error = %v", err)
	}
	if updated.ParentID == nil || *updated.ParentID != parent.ID {
		t.Fatalf("SetTaskParent() = %#v", updated)
	}

	detached, err := svc.UnsetTaskParent(child.ID)
	if err != nil {
		t.Fatalf("UnsetTaskParent() error = %v", err)
	}
	if detached.ParentID != nil {
		t.Fatalf("UnsetTaskParent() = %#v", detached)
	}
}

func TestLocalServiceUpdateTaskSupportsExpandedFields(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TASK_MODE", "local")
	t.Setenv("TASK_HOME", tempDir)
	dbPath := filepath.Join(tempDir, "task.db")
	if err := store.Init(dbPath, "admin", "secret"); err != nil {
		t.Fatalf("store.Init() error = %v", err)
	}

	svc := libtask.NewLocal(config.Config{})
	parent, err := svc.CreateTask(libtask.TaskCreateRequest{ProjectID: 1, Type: "epic", Title: "Parent"})
	if err != nil {
		t.Fatalf("CreateTask(parent) error = %v", err)
	}
	task, err := svc.CreateTask(libtask.TaskCreateRequest{
		ProjectID:          1,
		Type:               "task",
		Title:              "Child",
		Description:        "old description",
		AcceptanceCriteria: "old ac",
		Priority:           1,
	})
	if err != nil {
		t.Fatalf("CreateTask(task) error = %v", err)
	}
	if _, err := svc.RequestTask(libtask.TaskRequest{ProjectID: 1, TaskID: &task.ID}); err != nil {
		t.Fatalf("RequestTask() error = %v", err)
	}

	updated, err := svc.UpdateTask(task.ID, libtask.TaskUpdateRequest{
		Title:              "Updated Child",
		Description:        "new description",
		AcceptanceCriteria: "new ac",
		ParentID:           &parent.ID,
		Assignee:           task.Assignee,
		Status:             "inprogress",
		Priority:           3,
		Order:              7,
	})
	if err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	if updated.Title != "Updated Child" || updated.Description != "new description" || updated.AcceptanceCriteria != "new ac" || updated.Status != "inprogress" || updated.Priority != 3 || updated.Order != 7 {
		t.Fatalf("UpdateTask() = %#v", updated)
	}
	if updated.ParentID == nil || *updated.ParentID != parent.ID {
		t.Fatalf("UpdateTask() parent = %#v", updated)
	}
}
