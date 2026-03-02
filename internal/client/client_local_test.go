package client

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/simonski/task/internal/config"
	"github.com/simonski/task/internal/store"
)

func TestLocalModeClientUsesSQLiteDirectly(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TASK_MODE", "local")
	t.Setenv("TASK_HOME", tempDir)

	dbPath := filepath.Join(tempDir, "task.db")
	if err := store.Init(dbPath, "admin", "secret"); err != nil {
		t.Fatalf("store.Init() error = %v", err)
	}

	api := New(config.Config{})
	projects, err := api.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) != 1 || projects[0].ID != 1 {
		t.Fatalf("ListProjects() = %#v", projects)
	}

	task, err := api.CreateTask(TaskCreateRequest{
		ProjectID: 1,
		Type:      "task",
		Title:     "Local task",
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if strings.TrimSpace(task.Assignee) != "" || task.Status != "open" {
		t.Fatalf("CreateTask() = %#v", task)
	}

	requested, err := api.RequestTask(TaskRequest{ProjectID: 1})
	if err != nil {
		t.Fatalf("RequestTask() error = %v", err)
	}
	if requested.Status != "ASSIGNED" || requested.Task == nil {
		t.Fatalf("RequestTask() = %#v", requested)
	}

	updated, err := api.UpdateTask(task.ID, TaskUpdateRequest{
		Title:       task.Title,
		Description: task.Description,
		ParentID:    task.ParentID,
		Assignee:    requested.Task.Assignee,
		Status:      "inprogress",
	})
	if err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	if updated.Status != "inprogress" {
		t.Fatalf("UpdateTask().Status = %q, want inprogress", updated.Status)
	}

	parent, err := api.CreateTask(TaskCreateRequest{
		ProjectID: 1,
		Type:      "epic",
		Title:     "Parent epic",
	})
	if err != nil {
		t.Fatalf("CreateTask(parent) error = %v", err)
	}
	reparented, err := api.SetTaskParent(task.ID, parent.ID)
	if err != nil {
		t.Fatalf("SetTaskParent() error = %v", err)
	}
	if reparented.ParentID == nil || *reparented.ParentID != parent.ID {
		t.Fatalf("SetTaskParent() = %#v", reparented)
	}

	detached, err := api.UnsetTaskParent(task.ID)
	if err != nil {
		t.Fatalf("UnsetTaskParent() error = %v", err)
	}
	if detached.ParentID != nil {
		t.Fatalf("UnsetTaskParent() = %#v", detached)
	}

	comment, err := api.AddComment(task.ID, "hello")
	if err != nil {
		t.Fatalf("AddComment() error = %v", err)
	}
	if comment.Comment != "hello" {
		t.Fatalf("AddComment() = %#v", comment)
	}
}
