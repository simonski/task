package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/simonski/ticket/internal/password"
)

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		db.Close()
		return nil, err
	}
	if err := createSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func Init(path, adminUsername, adminPassword string) error {
	if adminUsername == "" || adminPassword == "" {
		return errors.New("admin username and password are required")
	}

	if path != ":memory:" {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("database already exists at %s", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
			return err
		}
	}

	db, err := Open(path)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := createSchema(db); err != nil {
		return err
	}

	hash, err := password.Hash(adminPassword)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO users (username, password_hash, role, display_name, enabled)
		VALUES (?, ?, 'admin', ?, 1)
	`, adminUsername, hash, adminUsername)
	if err != nil {
		return err
	}

	if _, err := CreateProject(db, "Default Project", "Bootstrap project created during initdb.", "", 1); err != nil {
		return err
	}
	return nil
}

func createSchema(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS users (
	user_id INTEGER PRIMARY KEY AUTOINCREMENT,
	username TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	role TEXT NOT NULL,
	display_name TEXT NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 1,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
	session_id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	token TEXT NOT NULL UNIQUE,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	expires_at TEXT,
	FOREIGN KEY(user_id) REFERENCES users(user_id)
);

CREATE TABLE IF NOT EXISTS projects (
	project_id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	acceptance_criteria TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'active',
	created_by INTEGER,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(created_by) REFERENCES users(user_id)
);

CREATE TABLE IF NOT EXISTS tasks (
	task_id INTEGER PRIMARY KEY AUTOINCREMENT,
	project_id INTEGER NOT NULL,
	parent_id INTEGER,
	clone_of INTEGER,
	type TEXT NOT NULL,
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	acceptance_criteria TEXT NOT NULL DEFAULT '',
	stage TEXT NOT NULL DEFAULT 'design',
	state TEXT NOT NULL DEFAULT 'idle',
	status TEXT NOT NULL DEFAULT 'open',
	priority INTEGER NOT NULL DEFAULT 3,
	sort_order INTEGER NOT NULL DEFAULT 0,
	estimate_effort INTEGER NOT NULL DEFAULT 0,
	estimate_complete TEXT NOT NULL DEFAULT '',
	assignee TEXT NOT NULL DEFAULT '',
	archived INTEGER NOT NULL DEFAULT 0,
	created_by INTEGER,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(project_id) REFERENCES projects(project_id),
	FOREIGN KEY(parent_id) REFERENCES tasks(task_id),
	FOREIGN KEY(clone_of) REFERENCES tasks(task_id),
	FOREIGN KEY(created_by) REFERENCES users(user_id)
);

CREATE TABLE IF NOT EXISTS history_events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	project_id INTEGER NOT NULL,
	task_id INTEGER NOT NULL,
	event_type TEXT NOT NULL,
	payload TEXT NOT NULL DEFAULT '{}',
	created_by INTEGER,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(project_id) REFERENCES projects(project_id),
	FOREIGN KEY(task_id) REFERENCES tasks(task_id),
	FOREIGN KEY(created_by) REFERENCES users(user_id)
);

CREATE TABLE IF NOT EXISTS comments (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	item_id INTEGER NOT NULL,
	user_id INTEGER NOT NULL,
	comment TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(item_id) REFERENCES tasks(task_id),
	FOREIGN KEY(user_id) REFERENCES users(user_id)
);

CREATE TABLE IF NOT EXISTS dependencies (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	project_id INTEGER NOT NULL,
	task_id INTEGER NOT NULL,
	depends_on INTEGER NOT NULL,
	created_by INTEGER,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(project_id) REFERENCES projects(project_id),
	FOREIGN KEY(task_id) REFERENCES tasks(task_id),
	FOREIGN KEY(depends_on) REFERENCES tasks(task_id),
	FOREIGN KEY(created_by) REFERENCES users(user_id)
);
`

	if _, err := db.Exec(schema); err != nil {
		return err
	}
	return migrateSchema(db)
}

func migrateSchema(db *sql.DB) error {
	if !columnExists(db, "tasks", "sort_order") {
		if _, err := db.Exec(`ALTER TABLE tasks ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	if !columnExists(db, "tasks", "estimate_effort") {
		if _, err := db.Exec(`ALTER TABLE tasks ADD COLUMN estimate_effort INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	if !columnExists(db, "tasks", "estimate_complete") {
		if _, err := db.Exec(`ALTER TABLE tasks ADD COLUMN estimate_complete TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	if !columnExists(db, "tasks", "stage") {
		if _, err := db.Exec(`ALTER TABLE tasks ADD COLUMN stage TEXT NOT NULL DEFAULT 'design'`); err != nil {
			return err
		}
	}
	if !columnExists(db, "tasks", "state") {
		if _, err := db.Exec(`ALTER TABLE tasks ADD COLUMN state TEXT NOT NULL DEFAULT 'idle'`); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`
		UPDATE tasks
		SET
			stage = CASE
				WHEN status = 'notready' THEN 'design'
				WHEN status = 'open' THEN 'develop'
				WHEN status = 'inprogress' THEN 'develop'
				WHEN status = 'complete' THEN 'done'
				WHEN status = 'fail' THEN 'test'
				ELSE COALESCE(NULLIF(TRIM(stage), ''), 'design')
			END,
			state = CASE
				WHEN status IN ('notready', 'open') THEN 'idle'
				WHEN status = 'inprogress' THEN 'active'
				WHEN status IN ('complete', 'fail') THEN 'complete'
				ELSE COALESCE(NULLIF(TRIM(state), ''), 'idle')
			END
		WHERE COALESCE(NULLIF(TRIM(stage), ''), '') = '' OR COALESCE(NULLIF(TRIM(state), ''), '') = ''
		   OR stage NOT IN ('design', 'develop', 'test', 'done')
		   OR state NOT IN ('idle', 'active', 'complete')
	`); err != nil {
		return err
	}
	return nil
}

func columnExists(db *sql.DB, tableName, columnName string) bool {
	rows, err := db.Query(`PRAGMA table_info(` + tableName + `)`)
	if err != nil {
		return false
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notNull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dflt, &pk); err != nil {
			return false
		}
		if name == columnName {
			return true
		}
	}
	return false
}
