package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var ErrProjectNotFound = errors.New("project not found")

type Project struct {
	ID          int64  `json:"project_id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedBy   int64  `json:"created_by"`
	CreatedAt   string `json:"created_at"`
}

func CreateProject(db *sql.DB, title, description string, createdBy int64) (Project, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Project{}, errors.New("project title is required")
	}

	slug := slugify(title)
	result, err := db.Exec(`
		INSERT INTO projects (slug, title, description, created_by)
		VALUES (?, ?, ?, ?)
	`, slug, title, description, createdBy)
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
		SELECT project_id, slug, title, description, status, COALESCE(created_by, 0), created_at
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
		if err := rows.Scan(&project.ID, &project.Slug, &project.Title, &project.Description, &project.Status, &project.CreatedBy, &project.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	return projects, rows.Err()
}

func GetProject(db *sql.DB, slugOrID string) (Project, error) {
	if slugOrID == "" {
		return Project{}, ErrProjectNotFound
	}

	var (
		row *sql.Row
		id  int64
	)
	if _, err := fmt.Sscan(slugOrID, &id); err == nil {
		row = db.QueryRow(`
			SELECT project_id, slug, title, description, status, COALESCE(created_by, 0), created_at
			FROM projects
			WHERE project_id = ?
		`, id)
	} else {
		row = db.QueryRow(`
			SELECT project_id, slug, title, description, status, COALESCE(created_by, 0), created_at
			FROM projects
			WHERE slug = ?
		`, slugOrID)
	}

	var project Project
	if err := row.Scan(&project.ID, &project.Slug, &project.Title, &project.Description, &project.Status, &project.CreatedBy, &project.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Project{}, ErrProjectNotFound
		}
		return Project{}, err
	}
	return project, nil
}

func GetProjectByID(db *sql.DB, id int64) (Project, error) {
	return GetProject(db, fmt.Sprintf("%d", id))
}

func slugify(title string) string {
	title = strings.ToLower(strings.TrimSpace(title))
	var b strings.Builder
	lastDash := false
	for _, r := range title {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "project"
	}
	return slug
}
