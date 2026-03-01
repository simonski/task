package store

import "testing"

func TestCountEverything(t *testing.T) {
	db := testDB(t)

	project, err := CreateProject(db, "Customer Portal", "", 1)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	otherProject, err := CreateProject(db, "Internal Tools", "", 1)
	if err != nil {
		t.Fatalf("CreateProject() other error = %v", err)
	}

	if _, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		Type:      "task",
		Title:     "Task A",
		Status:    "open",
		CreatedBy: 1,
	}); err != nil {
		t.Fatalf("CreateTask(task open) error = %v", err)
	}
	if _, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		Type:      "task",
		Title:     "Task B",
		Status:    "done",
		CreatedBy: 1,
	}); err != nil {
		t.Fatalf("CreateTask(task done) error = %v", err)
	}
	if _, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		Type:      "epic",
		Title:     "Epic A",
		Status:    "done",
		CreatedBy: 1,
	}); err != nil {
		t.Fatalf("CreateTask(epic done) error = %v", err)
	}
	if _, err := CreateTask(db, TaskCreateParams{
		ProjectID: otherProject.ID,
		Type:      "bug",
		Title:     "Bug A",
		Status:    "in_progress",
		CreatedBy: 1,
	}); err != nil {
		t.Fatalf("CreateTask(bug in progress) error = %v", err)
	}

	all, err := CountEverything(db, nil)
	if err != nil {
		t.Fatalf("CountEverything(all) error = %v", err)
	}
	if all.Users != 1 {
		t.Fatalf("CountEverything(all).Users = %d, want 1", all.Users)
	}
	if all.Projects != 3 {
		t.Fatalf("CountEverything(all).Projects = %d, want 3", all.Projects)
	}
	if len(all.Types) != 3 {
		t.Fatalf("CountEverything(all).Types len = %d, want 3", len(all.Types))
	}

	projectOnly, err := CountEverything(db, &project.ID)
	if err != nil {
		t.Fatalf("CountEverything(project) error = %v", err)
	}
	if projectOnly.Projects != 0 {
		t.Fatalf("CountEverything(project).Projects = %d, want 0", projectOnly.Projects)
	}
	if len(projectOnly.Types) != 2 {
		t.Fatalf("CountEverything(project).Types len = %d, want 2", len(projectOnly.Types))
	}
	if projectOnly.Types[0].Type != "task" || projectOnly.Types[0].Total != 2 {
		t.Fatalf("CountEverything(project).Types[0] = %#v", projectOnly.Types[0])
	}
	if projectOnly.Types[0].Statuses["done"] != 1 || projectOnly.Types[0].Statuses["open"] != 1 {
		t.Fatalf("CountEverything(project).Types[0].Statuses = %#v", projectOnly.Types[0].Statuses)
	}
}
