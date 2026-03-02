package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var ErrTaskNotFound = errors.New("task not found")

type Task struct {
	ID                 int64  `json:"task_id"`
	ProjectID          int64  `json:"project_id"`
	ParentID           *int64 `json:"parent_id,omitempty"`
	CloneOf            *int64 `json:"clone_of,omitempty"`
	Type               string `json:"type"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
	Status             string `json:"status"`
	Priority           int    `json:"priority"`
	Order              int    `json:"order"`
	Assignee           string `json:"assignee"`
	Archived           bool   `json:"archived"`
	CreatedBy          int64  `json:"created_by"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

type TaskCreateParams struct {
	ProjectID          int64
	ParentID           *int64
	CloneOf            *int64
	Type               string
	Title              string
	Description        string
	AcceptanceCriteria string
	Priority           int
	Order              int
	Assignee           string
	Status             string
	CreatedBy          int64
}

type TaskUpdateParams struct {
	Title              string
	Description        string
	AcceptanceCriteria string
	ParentID           *int64
	Assignee           string
	Status             string
	Priority           int
	Order              int
	UpdatedBy          int64
	ActorUsername      string
	ActorRole          string
}

type TaskListParams struct {
	ProjectID int64
	Type      string
	Status    string
	Search    string
	Assignee  string
	Limit     int
}

var validStatuses = map[string]bool{
	"notready":   true,
	"open":       true,
	"inprogress": true,
	"complete":   true,
	"fail":       true,
}

type TaskRequestParams struct {
	ProjectID int64
	TaskID    *int64
	Username  string
	UserID    int64
}

func CreateTask(db *sql.DB, params TaskCreateParams) (Task, error) {
	params.Type = normalizeTaskType(params.Type)
	params.Title = strings.TrimSpace(params.Title)
	if params.ProjectID == 0 {
		return Task{}, errors.New("project is required")
	}
	if params.Title == "" {
		return Task{}, errors.New("task title is required")
	}
	if !validTaskType(params.Type) {
		return Task{}, fmt.Errorf("invalid task type %q", params.Type)
	}
	status := defaultStatusForType(params.Type, params.Status)
	priority := params.Priority
	if priority == 0 {
		priority = 1
	}
	order := params.Order

	result, err := db.Exec(`
		INSERT INTO tasks (project_id, parent_id, clone_of, type, title, description, acceptance_criteria, status, priority, sort_order, assignee, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, params.ProjectID, nullableInt64(params.ParentID), nullableInt64(params.CloneOf), params.Type, params.Title, params.Description, strings.TrimSpace(params.AcceptanceCriteria), status, priority, order, strings.TrimSpace(params.Assignee), params.CreatedBy)
	if err != nil {
		return Task{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Task{}, err
	}
	task, err := GetTask(db, id)
	if err != nil {
		return Task{}, err
	}
	if err := AddHistoryEvent(db, task.ProjectID, task.ID, "task_created", map[string]any{
		"type":   task.Type,
		"title":  task.Title,
		"status": task.Status,
	}, params.CreatedBy); err != nil {
		return Task{}, err
	}
	return task, nil
}

func UpdateTask(db *sql.DB, id int64, params TaskUpdateParams) (Task, error) {
	title := strings.TrimSpace(params.Title)
	if title == "" {
		return Task{}, errors.New("task title is required")
	}
	current, err := GetTask(db, id)
	if err != nil {
		return Task{}, err
	}
	assignee := strings.TrimSpace(params.Assignee)
	if err := validateTaskAssignmentChange(current.Assignee, assignee, params.ActorUsername, params.ActorRole); err != nil {
		return Task{}, err
	}
	if assignee != "" {
		target, err := GetUserByUsername(db, assignee)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return Task{}, errors.New("user not found")
			}
			return Task{}, err
		}
		if !target.Enabled {
			return Task{}, errors.New("user is disabled")
		}
	}

	status := current.Status
	if strings.TrimSpace(params.Status) != "" {
		status = normalizeStatus(params.Status)
		if !validStatus(status) {
			return Task{}, fmt.Errorf("invalid status %q", params.Status)
		}
		if status != current.Status {
			if strings.TrimSpace(current.Assignee) != strings.TrimSpace(params.ActorUsername) {
				return Task{}, ErrForbidden
			}
			if isClosedStatus(current.Status) {
				return Task{}, errors.New("closed ticket cannot be reopened")
			}
		}
	}

	result, err := db.Exec(`
		UPDATE tasks
		SET title = ?, description = ?, acceptance_criteria = ?, parent_id = ?, assignee = ?, status = ?, priority = ?, sort_order = ?, updated_at = CURRENT_TIMESTAMP
		WHERE task_id = ?
	`, title, params.Description, strings.TrimSpace(params.AcceptanceCriteria), nullableInt64(params.ParentID), assignee, status, params.Priority, params.Order, id)
	if err != nil {
		return Task{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Task{}, err
	}
	if affected == 0 {
		return Task{}, ErrTaskNotFound
	}
	task, err := GetTask(db, id)
	if err != nil {
		return Task{}, err
	}
	if err := AddHistoryEvent(db, task.ProjectID, task.ID, "task_updated", map[string]any{
		"title":       task.Title,
		"description": task.Description,
		"acceptance_criteria": task.AcceptanceCriteria,
		"assignee":    task.Assignee,
		"status":      task.Status,
		"priority":    task.Priority,
		"order":       task.Order,
		"parent_id":   task.ParentID,
	}, params.UpdatedBy); err != nil {
		return Task{}, err
	}
	return task, nil
}

func ListTasksByProject(db *sql.DB, projectID int64) ([]Task, error) {
	return ListTasks(db, TaskListParams{ProjectID: projectID})
}

func ListTasks(db *sql.DB, params TaskListParams) ([]Task, error) {
	if params.ProjectID == 0 {
		return nil, errors.New("project is required")
	}

	query := `
		SELECT task_id, project_id, parent_id, clone_of, type, title, description, acceptance_criteria, status, priority, sort_order, assignee, archived, COALESCE(created_by, 0), created_at, updated_at
		FROM tasks
		WHERE project_id = ?
	`
	args := []any{params.ProjectID}
	if taskType := normalizeOptional(params.Type); taskType != "" {
		query += ` AND type = ?`
		args = append(args, taskType)
	}
	if status := normalizeOptional(params.Status); status != "" {
		if !validStatus(status) {
			return nil, fmt.Errorf("invalid status %q", params.Status)
		}
		query += ` AND status = ?`
		args = append(args, status)
	}
	if search := strings.TrimSpace(params.Search); search != "" {
		query += ` AND (LOWER(title) LIKE ? OR LOWER(description) LIKE ?)`
		needle := "%" + strings.ToLower(search) + "%"
		args = append(args, needle, needle)
	}
	if assignee := strings.TrimSpace(params.Assignee); assignee != "" {
		query += ` AND assignee = ?`
		args = append(args, assignee)
	}
	query += ` ORDER BY created_at, task_id`
	if params.Limit < 0 {
		return nil, errors.New("limit must be zero or greater")
	}
	if params.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, params.Limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func SearchTasks(db *sql.DB, projectID int64, query string) ([]Task, error) {
	return ListTasks(db, TaskListParams{
		ProjectID: projectID,
		Search:    query,
	})
}

func GetTaskByProject(db *sql.DB, projectID, id int64) (Task, error) {
	row := db.QueryRow(`
		SELECT task_id, project_id, parent_id, clone_of, type, title, description, acceptance_criteria, status, priority, sort_order, assignee, archived, COALESCE(created_by, 0), created_at, updated_at
		FROM tasks
		WHERE project_id = ? AND task_id = ?
	`, projectID, id)
	task, err := scanTask(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, ErrTaskNotFound
		}
		return Task{}, err
	}
	return task, nil
}

func GetTask(db *sql.DB, id int64) (Task, error) {
	row := db.QueryRow(`
		SELECT task_id, project_id, parent_id, clone_of, type, title, description, acceptance_criteria, status, priority, sort_order, assignee, archived, COALESCE(created_by, 0), created_at, updated_at
		FROM tasks
		WHERE task_id = ?
	`, id)

	task, err := scanTask(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, ErrTaskNotFound
		}
		return Task{}, err
	}
	return task, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanTask(s scanner) (Task, error) {
	var task Task
	var parentID sql.NullInt64
	var cloneOf sql.NullInt64
	var archived int
	if err := s.Scan(
		&task.ID,
		&task.ProjectID,
		&parentID,
		&cloneOf,
		&task.Type,
		&task.Title,
		&task.Description,
		&task.AcceptanceCriteria,
		&task.Status,
		&task.Priority,
		&task.Order,
		&task.Assignee,
		&archived,
		&task.CreatedBy,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return Task{}, err
	}
	if parentID.Valid {
		task.ParentID = &parentID.Int64
	}
	if cloneOf.Valid {
		task.CloneOf = &cloneOf.Int64
	}
	task.Archived = archived == 1
	return task, nil
}

func normalizeTaskType(taskType string) string {
	taskType = strings.TrimSpace(strings.ToLower(taskType))
	if taskType == "" {
		return "task"
	}
	return taskType
}

func normalizeStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return "open"
	}
	switch status {
	case "ready":
		return "open"
	case "in_progress":
		return "inprogress"
	case "done":
		return "complete"
	}
	return status
}

func validStatus(status string) bool {
	return validStatuses[status]
}

func normalizeOptional(v string) string {
	return strings.TrimSpace(strings.ToLower(v))
}

func validTaskType(taskType string) bool {
	switch taskType {
	case "task", "bug", "epic":
		return true
	default:
		return false
	}
}

func nullableInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

func defaultStatusForType(taskType, requested string) string {
	if requested = normalizeOptional(requested); requested != "" {
		return requested
	}
	return "open"
}

func isClosedStatus(status string) bool {
	return status == "complete" || status == "fail"
}

func validateTaskAssignmentChange(currentAssignee, nextAssignee, actorUsername, actorRole string) error {
	currentAssignee = strings.TrimSpace(currentAssignee)
	nextAssignee = strings.TrimSpace(nextAssignee)
	actorUsername = strings.TrimSpace(actorUsername)
	actorRole = strings.TrimSpace(actorRole)

	if currentAssignee == nextAssignee {
		return nil
	}
	if actorRole == "admin" {
		return nil
	}
	if actorUsername == "" {
		return errors.New("username is required for assignment changes")
	}
	if nextAssignee == actorUsername {
		if currentAssignee != "" && currentAssignee != actorUsername {
			return fmt.Errorf("task is already assigned to %s", currentAssignee)
		}
		return nil
	}
	if nextAssignee == "" {
		if currentAssignee != actorUsername {
			if currentAssignee == "" {
				return errors.New("task is not assigned to you")
			}
			return fmt.Errorf("task is assigned to %s", currentAssignee)
		}
		return nil
	}
	return ErrAdminRequired
}

func RequestTask(db *sql.DB, params TaskRequestParams) (Task, string, error) {
	username := strings.TrimSpace(params.Username)
	if username == "" {
		return Task{}, "", errors.New("username is required")
	}

	if task, ok, err := findAssignedTaskForUser(db, params.ProjectID, username, "inprogress"); err != nil {
		return Task{}, "", err
	} else if ok {
		return task, "ASSIGNED", nil
	}
	if task, ok, err := findAssignedTaskForUser(db, params.ProjectID, username, "open"); err != nil {
		return Task{}, "", err
	} else if ok {
		return task, "ASSIGNED", nil
	}

	if params.TaskID != nil {
		task, err := GetTask(db, *params.TaskID)
		if err != nil {
			return Task{}, "", err
		}
		if params.ProjectID != 0 && task.ProjectID != params.ProjectID {
			return Task{}, "REJECTED", nil
		}
		if strings.TrimSpace(task.Assignee) == username {
			return task, "ASSIGNED", nil
		}
		if strings.TrimSpace(task.Assignee) != "" {
			return Task{}, "REJECTED", nil
		}
		if task.Status != "open" {
			return Task{}, "REJECTED", nil
		}
		assigned, err := UpdateTask(db, task.ID, TaskUpdateParams{
			Title:              task.Title,
			Description:        task.Description,
			AcceptanceCriteria: task.AcceptanceCriteria,
			ParentID:           task.ParentID,
			Assignee:           username,
			Status:             task.Status,
			Priority:           task.Priority,
			Order:              task.Order,
			UpdatedBy:          params.UserID,
			ActorUsername:      username,
			ActorRole:          "user",
		})
		if err != nil {
			return Task{}, "", err
		}
		return assigned, "ASSIGNED", nil
	}

	task, ok, err := findUnassignedOpenTask(db, params.ProjectID)
	if err != nil {
		return Task{}, "", err
	}
	if !ok {
		return Task{}, "NO-WORK", nil
	}
	assigned, err := UpdateTask(db, task.ID, TaskUpdateParams{
		Title:              task.Title,
		Description:        task.Description,
		AcceptanceCriteria: task.AcceptanceCriteria,
		ParentID:           task.ParentID,
		Assignee:           username,
		Status:             task.Status,
		Priority:           task.Priority,
		Order:              task.Order,
		UpdatedBy:          params.UserID,
		ActorUsername:      username,
		ActorRole:          "user",
	})
	if err != nil {
		return Task{}, "", err
	}
	return assigned, "ASSIGNED", nil
}

func findAssignedTaskForUser(db *sql.DB, projectID int64, username, status string) (Task, bool, error) {
	query := `
		SELECT task_id, project_id, parent_id, clone_of, type, title, description, acceptance_criteria, status, priority, sort_order, assignee, archived, COALESCE(created_by, 0), created_at, updated_at
		FROM tasks
		WHERE assignee = ? AND status = ?
	`
	args := []any{username, status}
	if projectID != 0 {
		query += ` AND project_id = ?`
		args = append(args, projectID)
	}
	query += ` ORDER BY created_at, task_id LIMIT 1`
	task, err := scanTask(db.QueryRow(query, args...))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, false, nil
		}
		return Task{}, false, err
	}
	return task, true, nil
}

func findUnassignedOpenTask(db *sql.DB, projectID int64) (Task, bool, error) {
	if projectID == 0 {
		return Task{}, false, errors.New("project is required")
	}
	task, err := scanTask(db.QueryRow(`
		SELECT task_id, project_id, parent_id, clone_of, type, title, description, acceptance_criteria, status, priority, sort_order, assignee, archived, COALESCE(created_by, 0), created_at, updated_at
		FROM tasks
		WHERE project_id = ? AND status = 'open' AND TRIM(COALESCE(assignee, '')) = ''
		ORDER BY created_at, task_id
		LIMIT 1
	`, projectID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, false, nil
		}
		return Task{}, false, err
	}
	return task, true, nil
}

func CloneTask(db *sql.DB, id, createdBy int64) (Task, error) {
	original, err := GetTask(db, id)
	if err != nil {
		return Task{}, err
	}
	cloned, err := cloneTaskRecursive(db, original, nil, createdBy)
	if err != nil {
		return Task{}, err
	}
	return cloned, nil
}

func cloneTaskRecursive(db *sql.DB, original Task, parentID *int64, createdBy int64) (Task, error) {
	cloned, err := CreateTask(db, TaskCreateParams{
		ProjectID:          original.ProjectID,
		ParentID:           parentID,
		CloneOf:            &original.ID,
		Type:               original.Type,
		Title:              original.Title,
		Description:        original.Description,
		AcceptanceCriteria: original.AcceptanceCriteria,
		Priority:           original.Priority,
		Order:              original.Order,
		Assignee:           "",
		Status:             "notready",
		CreatedBy:          createdBy,
	})
	if err != nil {
		return Task{}, err
	}
	if original.Type != "epic" {
		return cloned, nil
	}
	children, err := ListTasks(db, TaskListParams{ProjectID: original.ProjectID})
	if err != nil {
		return Task{}, err
	}
	for _, child := range children {
		if child.ParentID != nil && *child.ParentID == original.ID {
			if _, err := cloneTaskRecursive(db, child, &cloned.ID, createdBy); err != nil {
				return Task{}, err
			}
		}
	}
	return cloned, nil
}
