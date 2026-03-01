package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var ErrProjectNotFound = errors.New("project not found")

type Project struct {
	ID                 int64  `json:"project_id"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
	Status             string `json:"status"`
	CreatedBy          int64  `json:"created_by"`
	CreatedAt          string `json:"created_at"`
}

func CreateProject(db *sql.DB, title, description, acceptanceCriteria string, createdBy int64) (Project, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Project{}, errors.New("project title is required")
	}

	result, err := db.Exec(`
		INSERT INTO projects (title, description, acceptance_criteria, created_by)
		VALUES (?, ?, ?, ?)
	`, title, strings.TrimSpace(description), strings.TrimSpace(acceptanceCriteria), createdBy)
	if err != nil {
		return Project{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Project{}, err
	}
	return GetProjectByID(db, id)
}

func ListProjects(db *sql.DB) ([]Project, error) {
	rows, err := db.Query(`
		SELECT project_id, title, description, acceptance_criteria, status, COALESCE(created_by, 0), created_at
		FROM projects
		ORDER BY created_at, project_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var project Project
		if err := rows.Scan(&project.ID, &project.Title, &project.Description, &project.AcceptanceCriteria, &project.Status, &project.CreatedBy, &project.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	return projects, rows.Err()
}

func GetProject(db *sql.DB, rawID string) (Project, error) {
	if strings.TrimSpace(rawID) == "" {
		return Project{}, ErrProjectNotFound
	}
	var id int64
	if _, err := fmt.Sscan(rawID, &id); err != nil {
		return Project{}, ErrProjectNotFound
	}
	return GetProjectByID(db, id)
}

func GetProjectByID(db *sql.DB, id int64) (Project, error) {
	row := db.QueryRow(`
		SELECT project_id, title, description, acceptance_criteria, status, COALESCE(created_by, 0), created_at
		FROM projects
		WHERE project_id = ?
	`, id)
	var project Project
	if err := row.Scan(&project.ID, &project.Title, &project.Description, &project.AcceptanceCriteria, &project.Status, &project.CreatedBy, &project.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Project{}, ErrProjectNotFound
		}
		return Project{}, err
	}
	return project, nil
}

func UpdateProject(db *sql.DB, id int64, title, description, acceptanceCriteria string) (Project, error) {
	current, err := GetProjectByID(db, id)
	if err != nil {
		return Project{}, err
	}
	nextTitle := strings.TrimSpace(title)
	if nextTitle == "" {
		nextTitle = current.Title
	}
	_, err = db.Exec(`
		UPDATE projects
		SET title = ?, description = ?, acceptance_criteria = ?
		WHERE project_id = ?
	`, nextTitle, description, acceptanceCriteria, id)
	if err != nil {
		return Project{}, err
	}
	return GetProjectByID(db, id)
}

func SetProjectStatus(db *sql.DB, id int64, enabled bool) (Project, error) {
	status := "disabled"
	if enabled {
		status = "active"
	}
	result, err := db.Exec(`UPDATE projects SET status = ? WHERE project_id = ?`, status, id)
	if err != nil {
		return Project{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Project{}, err
	}
	if affected == 0 {
		return Project{}, ErrProjectNotFound
	}
	return GetProjectByID(db, id)
}
