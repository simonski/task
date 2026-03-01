package store

import "testing"

func TestCreateUpdateAndListTasks(t *testing.T) {
	db := testDB(t)
	project, err := CreateProject(db, "Customer Portal", "", 1)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	epic, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		Type:      "epic",
		Title:     "Authentication",
		CreatedBy: 1,
	})
	if err != nil {
		t.Fatalf("CreateTask(epic) error = %v", err)
	}

	task, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		ParentID:  &epic.ID,
		Type:      "task",
		Title:     "Add password reset",
		CreatedBy: 1,
	})
	if err != nil {
		t.Fatalf("CreateTask(task) error = %v", err)
	}
	if task.ParentID == nil || *task.ParentID != epic.ID {
		t.Fatalf("CreateTask().ParentID = %#v, want %d", task.ParentID, epic.ID)
	}

	tasks, err := ListTasksByProject(db, project.ID)
	if err != nil {
		t.Fatalf("ListTasksByProject() error = %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("ListTasksByProject() len = %d, want 2", len(tasks))
	}

	updated, err := UpdateTask(db, task.ID, TaskUpdateParams{
		Title:       "Add password reset workflow",
		Description: "Support email-based reset",
		ParentID:    &epic.ID,
	})
	if err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	if updated.Title != "Add password reset workflow" {
		t.Fatalf("UpdateTask().Title = %q", updated.Title)
	}

	statusUpdated, err := UpdateTask(db, task.ID, TaskUpdateParams{
		Title:       updated.Title,
		Description: updated.Description,
		ParentID:    updated.ParentID,
		Status:      "in_progress",
	})
	if err != nil {
		t.Fatalf("UpdateTask(status) error = %v", err)
	}
	if statusUpdated.Status != "in_progress" {
		t.Fatalf("UpdateTask().Status = %q, want in_progress", statusUpdated.Status)
	}

	filtered, err := ListTasks(db, TaskListParams{
		ProjectID: project.ID,
		Type:      "task",
		Status:    "in_progress",
		Search:    "password",
	})
	if err != nil {
		t.Fatalf("ListTasks(filtered) error = %v", err)
	}
	if len(filtered) != 1 || filtered[0].ID != task.ID {
		t.Fatalf("ListTasks(filtered) = %#v", filtered)
	}

	limited, err := ListTasks(db, TaskListParams{
		ProjectID: project.ID,
		Limit:     1,
	})
	if err != nil {
		t.Fatalf("ListTasks(limited) error = %v", err)
	}
	if len(limited) != 1 {
		t.Fatalf("ListTasks(limited) len = %d, want 1", len(limited))
	}

	got, err := GetTaskByProject(db, project.ID, task.ID)
	if err != nil {
		t.Fatalf("GetTaskByProject() error = %v", err)
	}
	if got.ID != task.ID {
		t.Fatalf("GetTaskByProject().ID = %d, want %d", got.ID, task.ID)
	}
}

func TestUpdateTaskAssignmentRulesForNonAdmin(t *testing.T) {
	db := testDB(t)
	project, err := CreateProject(db, "Customer Portal", "", 1)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	task, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		Type:      "task",
		Title:     "Add password reset",
		CreatedBy: 1,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if _, err := CreateUser(db, "alice", "password123", "user"); err != nil {
		t.Fatalf("CreateUser(alice) error = %v", err)
	}
	if _, err := CreateUser(db, "bob", "password123", "user"); err != nil {
		t.Fatalf("CreateUser(bob) error = %v", err)
	}

	claimed, err := UpdateTask(db, task.ID, TaskUpdateParams{
		Title:         task.Title,
		Description:   task.Description,
		ParentID:      task.ParentID,
		Assignee:      "alice",
		ActorUsername: "alice",
		ActorRole:     "user",
	})
	if err != nil {
		t.Fatalf("UpdateTask(claim self) error = %v", err)
	}
	if claimed.Assignee != "alice" {
		t.Fatalf("UpdateTask(claim self).Assignee = %q, want alice", claimed.Assignee)
	}

	if _, err := UpdateTask(db, task.ID, TaskUpdateParams{
		Title:         claimed.Title,
		Description:   claimed.Description,
		ParentID:      claimed.ParentID,
		Assignee:      "bob",
		ActorUsername: "bob",
		ActorRole:     "user",
	}); err == nil || err.Error() != "task is already assigned to alice" {
		t.Fatalf("UpdateTask(claim assigned) error = %v, want task is already assigned to alice", err)
	}

	if _, err := UpdateTask(db, task.ID, TaskUpdateParams{
		Title:         claimed.Title,
		Description:   claimed.Description,
		ParentID:      claimed.ParentID,
		Assignee:      "",
		ActorUsername: "bob",
		ActorRole:     "user",
	}); err == nil || err.Error() != "task is assigned to alice" {
		t.Fatalf("UpdateTask(unclaim other) error = %v, want task is assigned to alice", err)
	}
}

func TestUpdateTaskAssignRequiresExistingEnabledUser(t *testing.T) {
	db := testDB(t)
	project, err := CreateProject(db, "Customer Portal", "", 1)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	task, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		Type:      "task",
		Title:     "Add password reset",
		CreatedBy: 1,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if _, err := CreateUser(db, "alice", "password123", "user"); err != nil {
		t.Fatalf("CreateUser(alice) error = %v", err)
	}
	if _, err := UpdateTask(db, task.ID, TaskUpdateParams{
		Title:         task.Title,
		Description:   task.Description,
		ParentID:      task.ParentID,
		Assignee:      "nobody",
		ActorUsername: "admin",
		ActorRole:     "admin",
	}); err == nil || err.Error() != "user not found" {
		t.Fatalf("UpdateTask(assign missing user) error = %v, want user not found", err)
	}
	if err := SetUserEnabled(db, "alice", false); err != nil {
		t.Fatalf("SetUserEnabled(false) error = %v", err)
	}
	if _, err := UpdateTask(db, task.ID, TaskUpdateParams{
		Title:         task.Title,
		Description:   task.Description,
		ParentID:      task.ParentID,
		Assignee:      "alice",
		ActorUsername: "admin",
		ActorRole:     "admin",
	}); err == nil || err.Error() != "user is disabled" {
		t.Fatalf("UpdateTask(assign disabled user) error = %v, want user is disabled", err)
	}
}
