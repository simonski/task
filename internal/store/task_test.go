package store

import (
	"errors"
	"testing"
)

func TestCreateUpdateAndListTasks(t *testing.T) {
	db := testDB(t)
	project, err := CreateProject(db, "Customer Portal", "", "", 1)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if _, err := CreateUser(db, "alice", "password123", "user"); err != nil {
		t.Fatalf("CreateUser(alice) error = %v", err)
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
		ProjectID:        project.ID,
		ParentID:         &epic.ID,
		Type:             "task",
		Title:            "Add password reset",
		EstimateEffort:   5,
		EstimateComplete: "2026-04-01T12:00:00Z",
		CreatedBy:        1,
	})
	if err != nil {
		t.Fatalf("CreateTask(task) error = %v", err)
	}
	if task.ParentID == nil || *task.ParentID != epic.ID {
		t.Fatalf("CreateTask().ParentID = %#v, want %d", task.ParentID, epic.ID)
	}
	if task.Stage != StageDesign || task.State != StateIdle {
		t.Fatalf("CreateTask().Lifecycle = %s/%s, want design/idle", task.Stage, task.State)
	}
	if task.EstimateEffort != 5 || task.EstimateComplete != "2026-04-01T12:00:00Z" {
		t.Fatalf("CreateTask() estimates = %#v", task)
	}

	tasks, err := ListTasksByProject(db, project.ID)
	if err != nil {
		t.Fatalf("ListTasksByProject() error = %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("ListTasksByProject() len = %d, want 2", len(tasks))
	}

	updated, err := UpdateTask(db, task.ID, TaskUpdateParams{
		Title:            "Add password reset workflow",
		Description:      "Support email-based reset",
		ParentID:         &epic.ID,
		EstimateEffort:   8,
		EstimateComplete: "2026-04-15T09:00:00Z",
	})
	if err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	if updated.Title != "Add password reset workflow" {
		t.Fatalf("UpdateTask().Title = %q", updated.Title)
	}
	if updated.EstimateEffort != 8 || updated.EstimateComplete != "2026-04-15T09:00:00Z" {
		t.Fatalf("UpdateTask() estimates = %#v", updated)
	}

	statusUpdated, err := UpdateTask(db, task.ID, TaskUpdateParams{
		Title:            updated.Title,
		Description:      updated.Description,
		ParentID:         updated.ParentID,
		Stage:            StageDevelop,
		State:            StateActive,
		Assignee:         "alice",
		ActorUsername:    "admin",
		ActorRole:        "admin",
		EstimateEffort:   updated.EstimateEffort,
		EstimateComplete: updated.EstimateComplete,
	})
	if err != nil {
		t.Fatalf("UpdateTask(stage/state) error = %v", err)
	}
	if statusUpdated.Status != "develop/active" {
		t.Fatalf("UpdateTask().Status = %q, want develop/active", statusUpdated.Status)
	}
	if statusUpdated.Stage != StageDevelop || statusUpdated.State != StateActive {
		t.Fatalf("UpdateTask().Lifecycle = %s/%s, want develop/active", statusUpdated.Stage, statusUpdated.State)
	}

	filtered, err := ListTasks(db, TaskListParams{
		ProjectID: project.ID,
		Type:      "task",
		Status:    "develop/active",
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

func TestCreateOrUpdateTaskEnforcesEpicParentRules(t *testing.T) {
	db := testDB(t)
	project, err := CreateProject(db, "Customer Portal", "", "", 1)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	taskParent, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		Type:      "task",
		Title:     "Regular task",
		CreatedBy: 1,
	})
	if err != nil {
		t.Fatalf("CreateTask(task) error = %v", err)
	}
	if _, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		ParentID:  &taskParent.ID,
		Type:      "epic",
		Title:     "Invalid epic",
		CreatedBy: 1,
	}); err == nil || err.Error() != "epic parent must be an epic" {
		t.Fatalf("CreateTask(epic with non-epic parent) error = %v, want epic parent must be an epic", err)
	}

	epicParent, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		Type:      "epic",
		Title:     "Valid epic",
		CreatedBy: 1,
	})
	if err != nil {
		t.Fatalf("CreateTask(epic) error = %v", err)
	}

	taskChild, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		ParentID:  &epicParent.ID,
		Type:      "task",
		Title:     "Task child",
		CreatedBy: 1,
	})
	if err != nil {
		t.Fatalf("CreateTask(task child) error = %v", err)
	}

	_, err = UpdateTask(db, epicParent.ID, TaskUpdateParams{
		Title:    "Valid epic",
		ParentID: &taskChild.ID,
	})
	if err == nil || err.Error() != "epic parent must be an epic" {
		t.Fatalf("UpdateTask(epic parented by task) error = %v, want epic parent must be an epic", err)
	}
}

func TestRequestTask(t *testing.T) {
	db := testDB(t)
	project, err := CreateProject(db, "Customer Portal", "", "", 1)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if _, err := CreateUser(db, "alice", "password123", "user"); err != nil {
		t.Fatalf("CreateUser(alice) error = %v", err)
	}

	notReady, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		Type:      "task",
		Title:     "Blocked setup",
		Stage:     StageDesign,
		State:     StateIdle,
		CreatedBy: 1,
	})
	if err != nil {
		t.Fatalf("CreateTask(design/idle) error = %v", err)
	}
	openTask, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		Type:      "task",
		Title:     "Open task",
		Stage:     StageDevelop,
		State:     StateIdle,
		CreatedBy: 1,
	})
	if err != nil {
		t.Fatalf("CreateTask(develop/idle) error = %v", err)
	}

	assigned, status, err := RequestTask(db, TaskRequestParams{
		ProjectID: project.ID,
		Username:  "alice",
		UserID:    2,
	})
	if err != nil {
		t.Fatalf("RequestTask(any) error = %v", err)
	}
	if status != "ASSIGNED" || assigned.ID != openTask.ID {
		t.Fatalf("RequestTask(any) = %#v, %q", assigned, status)
	}

	assignedAgain, status, err := RequestTask(db, TaskRequestParams{
		ProjectID: project.ID,
		Username:  "alice",
		UserID:    2,
	})
	if err != nil {
		t.Fatalf("RequestTask(existing open) error = %v", err)
	}
	if status != "ASSIGNED" || assignedAgain.ID != openTask.ID {
		t.Fatalf("RequestTask(existing open) = %#v, %q", assignedAgain, status)
	}

	inProgress, err := UpdateTask(db, openTask.ID, TaskUpdateParams{
		Title:         assigned.Title,
		Description:   assigned.Description,
		ParentID:      assigned.ParentID,
		Assignee:      "alice",
		Stage:         StageDevelop,
		State:         StateActive,
		UpdatedBy:     2,
		ActorUsername: "alice",
		ActorRole:     "user",
	})
	if err != nil {
		t.Fatalf("UpdateTask(develop/active) error = %v", err)
	}

	requested, status, err := RequestTask(db, TaskRequestParams{
		ProjectID: project.ID,
		TaskID:    &notReady.ID,
		Username:  "alice",
		UserID:    2,
	})
	if err != nil {
		t.Fatalf("RequestTask(existing inprogress) error = %v", err)
	}
	if status != "ASSIGNED" || requested.ID != inProgress.ID {
		t.Fatalf("RequestTask(existing inprogress) = %#v, %q", requested, status)
	}

	if _, err := CreateUser(db, "bob", "password123", "user"); err != nil {
		t.Fatalf("CreateUser(bob) error = %v", err)
	}
	rejected, status, err := RequestTask(db, TaskRequestParams{
		ProjectID: project.ID,
		TaskID:    &notReady.ID,
		Username:  "bob",
		UserID:    3,
	})
	if err != nil {
		t.Fatalf("RequestTask(rejected) error = %v", err)
	}
	if status != "REJECTED" || rejected.ID != 0 {
		t.Fatalf("RequestTask(rejected) = %#v, %q", rejected, status)
	}

	noWork, status, err := RequestTask(db, TaskRequestParams{
		ProjectID: project.ID,
		Username:  "bob",
		UserID:    3,
	})
	if err != nil {
		t.Fatalf("RequestTask(no-work) error = %v", err)
	}
	if status != "NO-WORK" || noWork.ID != 0 {
		t.Fatalf("RequestTask(no-work) = %#v, %q", noWork, status)
	}
}

func TestUpdateTaskAssignmentRulesForNonAdmin(t *testing.T) {
	db := testDB(t)
	project, err := CreateProject(db, "Customer Portal", "", "", 1)
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
	project, err := CreateProject(db, "Customer Portal", "", "", 1)
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

func TestUpdateTaskStatusRequiresAssignee(t *testing.T) {
	db := testDB(t)
	project, err := CreateProject(db, "Customer Portal", "", "", 1)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if _, err := CreateUser(db, "alice", "password123", "user"); err != nil {
		t.Fatalf("CreateUser(alice) error = %v", err)
	}
	task, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		Type:      "task",
		Title:     "Status-owned task",
		CreatedBy: 1,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if _, err := UpdateTask(db, task.ID, TaskUpdateParams{
		Title:         task.Title,
		Description:   task.Description,
		ParentID:      task.ParentID,
		Assignee:      "",
		Stage:         StageDevelop,
		State:         StateActive,
		UpdatedBy:     2,
		ActorUsername: "alice",
		ActorRole:     "user",
	}); err == nil || err.Error() != "active ticket requires assignee" {
		t.Fatalf("UpdateTask(status unassigned) error = %v, want active ticket requires assignee", err)
	}
}

func TestUpdateTaskStatusAllowsAdminBypass(t *testing.T) {
	db := testDB(t)
	project, err := CreateProject(db, "Customer Portal", "", "", 1)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	task, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		Type:      "task",
		Title:     "Admin-bypass task",
		CreatedBy: 1,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if _, err := CreateUser(db, "alice", "password123", "user"); err != nil {
		t.Fatalf("CreateUser(alice) error = %v", err)
	}
	updated, err := UpdateTask(db, task.ID, TaskUpdateParams{
		Title:         task.Title,
		Description:   task.Description,
		ParentID:      task.ParentID,
		Assignee:      "alice",
		Stage:         StageDevelop,
		State:         StateActive,
		UpdatedBy:     1,
		ActorUsername: "admin",
		ActorRole:     "admin",
	})
	if err != nil {
		t.Fatalf("UpdateTask(admin lifecycle bypass) error = %v", err)
	}
	if updated.Status != "develop/active" {
		t.Fatalf("UpdateTask(admin lifecycle bypass).Status = %q, want develop/active", updated.Status)
	}
}

func TestClosedTaskCannotBeReopened(t *testing.T) {
	db := testDB(t)
	project, err := CreateProject(db, "Customer Portal", "", "", 1)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if _, err := CreateUser(db, "alice", "password123", "user"); err != nil {
		t.Fatalf("CreateUser(alice) error = %v", err)
	}
	task, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		Type:      "task",
		Title:     "Closed task",
		Assignee:  "alice",
		Stage:     StageDone,
		State:     StateComplete,
		CreatedBy: 1,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if _, err := UpdateTask(db, task.ID, TaskUpdateParams{
		Title:         task.Title,
		Description:   task.Description,
		ParentID:      task.ParentID,
		Assignee:      "alice",
		Stage:         StageDevelop,
		State:         StateIdle,
		UpdatedBy:     2,
		ActorUsername: "alice",
		ActorRole:     "user",
	}); err == nil || err.Error() != "done ticket cannot be reopened" {
		t.Fatalf("UpdateTask(reopen) error = %v", err)
	}
}

func TestCloneTaskClonesSingleTask(t *testing.T) {
	db := testDB(t)
	project, err := CreateProject(db, "Customer Portal", "", "", 1)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	task, err := CreateTask(db, TaskCreateParams{
		ProjectID:          project.ID,
		Type:               "task",
		Title:              "Original task",
		Description:        "desc",
		AcceptanceCriteria: "ac",
		Assignee:           "alice",
		Stage:              StageDevelop,
		State:              StateActive,
		Priority:           3,
		CreatedBy:          1,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	cloned, err := CloneTask(db, task.ID, 1)
	if err != nil {
		t.Fatalf("CloneTask() error = %v", err)
	}
	if cloned.ID == task.ID || cloned.Status != "design/idle" || cloned.Assignee != "" {
		t.Fatalf("CloneTask() = %#v", cloned)
	}
	if cloned.CloneOf == nil || *cloned.CloneOf != task.ID {
		t.Fatalf("CloneTask().CloneOf = %#v, want %d", cloned.CloneOf, task.ID)
	}
}

func TestDeleteTaskDeletesTaskAndRelatedRows(t *testing.T) {
	db := testDB(t)
	project, err := CreateProject(db, "Customer Portal", "", "", 1)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	task, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		Type:      "task",
		Title:     "Delete me",
		CreatedBy: 1,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	clone, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		CloneOf:   &task.ID,
		Type:      "task",
		Title:     "Clone stays",
		CreatedBy: 1,
	})
	if err != nil {
		t.Fatalf("CreateTask(clone) error = %v", err)
	}
	if _, err := AddComment(db, task.ID, 1, "hello"); err != nil {
		t.Fatalf("AddComment() error = %v", err)
	}
	if err := AddHistoryEvent(db, project.ID, task.ID, "task_updated", map[string]any{"title": task.Title}, 1); err != nil {
		t.Fatalf("AddHistoryEvent() error = %v", err)
	}
	dependency, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		Type:      "task",
		Title:     "Dependency",
		CreatedBy: 1,
	})
	if err != nil {
		t.Fatalf("CreateTask(dependency) error = %v", err)
	}
	if _, err := AddDependency(db, project.ID, task.ID, dependency.ID, 1); err != nil {
		t.Fatalf("AddDependency() error = %v", err)
	}

	if err := DeleteTask(db, task.ID); err != nil {
		t.Fatalf("DeleteTask() error = %v", err)
	}
	if _, err := GetTask(db, task.ID); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("GetTask(deleted) error = %v, want ErrTaskNotFound", err)
	}

	clonedTask, err := GetTask(db, clone.ID)
	if err != nil {
		t.Fatalf("GetTask(clone) error = %v", err)
	}
	if clonedTask.CloneOf != nil {
		t.Fatalf("CloneOf = %#v, want nil after source delete", clonedTask.CloneOf)
	}
	if comments, err := ListComments(db, task.ID); err != nil || len(comments) != 0 {
		t.Fatalf("ListComments(deleted) = %#v, %v", comments, err)
	}
	if history, err := ListHistoryEvents(db, task.ID); err != nil || len(history) != 0 {
		t.Fatalf("ListHistoryEvents(deleted) = %#v, %v", history, err)
	}
	if deps, err := ListDependencies(db, task.ID); err != nil || len(deps) != 0 {
		t.Fatalf("ListDependencies(deleted) = %#v, %v", deps, err)
	}
}

func TestDeleteTaskFailsWhenTaskHasChildren(t *testing.T) {
	db := testDB(t)
	project, err := CreateProject(db, "Customer Portal", "", "", 1)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	parent, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		Type:      "epic",
		Title:     "Parent",
		CreatedBy: 1,
	})
	if err != nil {
		t.Fatalf("CreateTask(parent) error = %v", err)
	}
	if _, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		ParentID:  &parent.ID,
		Type:      "task",
		Title:     "Child",
		CreatedBy: 1,
	}); err != nil {
		t.Fatalf("CreateTask(child) error = %v", err)
	}

	if err := DeleteTask(db, parent.ID); !errors.Is(err, ErrTaskHasChildren) {
		t.Fatalf("DeleteTask(parent) error = %v, want ErrTaskHasChildren", err)
	}
}

func TestCloneEpicClonesChildren(t *testing.T) {
	db := testDB(t)
	project, err := CreateProject(db, "Customer Portal", "", "", 1)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	epic, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		Type:      "epic",
		Title:     "Epic",
		CreatedBy: 1,
	})
	if err != nil {
		t.Fatalf("CreateTask(epic) error = %v", err)
	}
	child, err := CreateTask(db, TaskCreateParams{
		ProjectID: project.ID,
		ParentID:  &epic.ID,
		Type:      "task",
		Title:     "Child",
		CreatedBy: 1,
	})
	if err != nil {
		t.Fatalf("CreateTask(child) error = %v", err)
	}
	clonedEpic, err := CloneTask(db, epic.ID, 1)
	if err != nil {
		t.Fatalf("CloneTask(epic) error = %v", err)
	}
	tasks, err := ListTasksByProject(db, project.ID)
	if err != nil {
		t.Fatalf("ListTasksByProject() error = %v", err)
	}
	var clonedChild Task
	var found bool
	for _, task := range tasks {
		if task.CloneOf != nil && *task.CloneOf == child.ID {
			clonedChild = task
			found = true
		}
	}
	if !found {
		t.Fatalf("cloned child not found in %#v", tasks)
	}
	if clonedChild.ParentID == nil || *clonedChild.ParentID != clonedEpic.ID {
		t.Fatalf("cloned child parent = %#v, want %d", clonedChild.ParentID, clonedEpic.ID)
	}
}
