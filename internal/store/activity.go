package store

import (
	"database/sql"
	"encoding/json"
)

type HistoryEvent struct {
	ID        int64  `json:"id"`
	ProjectID int64  `json:"project_id"`
	TaskID    int64  `json:"task_id"`
	EventType string `json:"event_type"`
	Payload   string `json:"payload"`
	CreatedBy int64  `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

type Comment struct {
	ID        int64  `json:"id"`
	ItemID    int64  `json:"item_id"`
	UserID    int64  `json:"user_id"`
	Comment   string `json:"comment"`
	CreatedAt string `json:"created_at"`
}

func AddHistoryEvent(db *sql.DB, projectID, taskID int64, eventType string, payload any, createdBy int64) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		INSERT INTO history_events (project_id, task_id, event_type, payload, created_by)
		VALUES (?, ?, ?, ?, ?)
	`, projectID, taskID, eventType, string(data), nullableUserID(createdBy))
	return err
}

func ListHistoryEvents(db *sql.DB, taskID int64) ([]HistoryEvent, error) {
	rows, err := db.Query(`
		SELECT id, project_id, task_id, event_type, payload, COALESCE(created_by, 0), created_at
		FROM history_events
		WHERE task_id = ?
		ORDER BY id
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []HistoryEvent
	for rows.Next() {
		var event HistoryEvent
		if err := rows.Scan(&event.ID, &event.ProjectID, &event.TaskID, &event.EventType, &event.Payload, &event.CreatedBy, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func AddComment(db *sql.DB, taskID, userID int64, comment string) (Comment, error) {
	result, err := db.Exec(`
		INSERT INTO comments (item_id, user_id, comment)
		VALUES (?, ?, ?)
	`, taskID, userID, comment)
	if err != nil {
		return Comment{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Comment{}, err
	}
	row := db.QueryRow(`
		SELECT id, item_id, user_id, comment, created_at
		FROM comments
		WHERE id = ?
	`, id)
	var c Comment
	if err := row.Scan(&c.ID, &c.ItemID, &c.UserID, &c.Comment, &c.CreatedAt); err != nil {
		return Comment{}, err
	}
	return c, nil
}

func ListComments(db *sql.DB, taskID int64) ([]Comment, error) {
	rows, err := db.Query(`
		SELECT id, item_id, user_id, comment, created_at
		FROM comments
		WHERE item_id = ?
		ORDER BY id
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.ItemID, &c.UserID, &c.Comment, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func nullableUserID(userID int64) any {
	if userID == 0 {
		return nil
	}
	return userID
}
