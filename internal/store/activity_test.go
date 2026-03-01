package store

import "testing"

func TestHistoryAndComments(t *testing.T) {
	db := testDB(t)
	project, err := CreateProject(db, "Customer Portal", "", 1)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	task, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		Type:      "task",
		Title:     "Add login",
		CreatedBy: 1,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	events, err := ListHistoryEvents(db, task.ID)
	if err != nil {
		t.Fatalf("ListHistoryEvents() error = %v", err)
	}
	if len(events) == 0 || events[0].EventType != "task_created" {
		t.Fatalf("history after create = %#v", events)
	}

	_, err = UpdateTask(db, task.ID, TaskUpdateParams{
		Title:       task.Title,
		Description: "Updated description",
		ParentID:    task.ParentID,
		Status:      "in_progress",
		UpdatedBy:   1,
	})
	if err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}

	events, err = ListHistoryEvents(db, task.ID)
	if err != nil {
		t.Fatalf("ListHistoryEvents(after update) error = %v", err)
	}
	if len(events) < 2 {
		t.Fatalf("history length = %d, want at least 2", len(events))
	}

	comment, err := AddComment(db, task.ID, 1, "Waiting on API changes.")
	if err != nil {
		t.Fatalf("AddComment() error = %v", err)
	}
	if comment.Comment != "Waiting on API changes." {
		t.Fatalf("AddComment().Comment = %q", comment.Comment)
	}

	comments, err := ListComments(db, task.ID)
	if err != nil {
		t.Fatalf("ListComments() error = %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("ListComments() len = %d, want 1", len(comments))
	}
}
