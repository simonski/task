package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/simonski/task/internal/store"
)

func registerAPI(mux *http.ServeMux, db *sql.DB, version string) {
	mux.HandleFunc("/api/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var ping int
		if err := db.QueryRow("SELECT 1").Scan(&ping); err != nil {
			writeError(w, http.StatusInternalServerError, "database unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": version})
	})

	mux.HandleFunc("/api/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var req credentialsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		user, err := store.RegisterUser(db, req.Username, req.Password)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, user)
	})

	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var req credentialsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		user, err := store.AuthenticateUser(db, req.Username, req.Password)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrInvalidCredentials):
				writeError(w, http.StatusUnauthorized, err.Error())
			case errors.Is(err, store.ErrForbidden):
				writeError(w, http.StatusForbidden, err.Error())
			default:
				writeError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}
		token, err := store.CreateSession(db, user.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, authResponse{Token: token, User: user})
	})

	mux.HandleFunc("/api/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		token := bearerToken(r)
		if err := store.DeleteSession(db, token); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
	})

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		user, err := userFromRequest(db, r)
		if err != nil {
			if errors.Is(err, store.ErrUnauthorized) {
				writeJSON(w, http.StatusOK, map[string]any{
					"status":         "ok",
					"authenticated":  false,
					"server_version": version,
				})
				return
			}
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":         "ok",
			"authenticated":  true,
			"server_version": version,
			"user":           user,
		})
	})

	mux.HandleFunc("/api/count", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if _, err := requireUser(db, r); err != nil {
			writeAuthError(w, err)
			return
		}
		var projectID *int64
		if raw := strings.TrimSpace(r.URL.Query().Get("project_id")); raw != "" {
			var parsed int64
			if _, err := fmt.Sscan(raw, &parsed); err != nil {
				writeError(w, http.StatusBadRequest, "project_id must be numeric")
				return
			}
			projectID = &parsed
		}
		summary, err := store.CountEverything(db, projectID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, summary)
	})

	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if _, err := requireAdmin(db, r); err != nil {
				writeAuthError(w, err)
				return
			}
			users, err := store.ListUsers(db)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, users)
		case http.MethodPost:
			if _, err := requireAdmin(db, r); err != nil {
				writeAuthError(w, err)
				return
			}
			var req credentialsRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "invalid json body")
				return
			}
			user, err := store.CreateUser(db, req.Username, req.Password, "user")
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, user)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	mux.HandleFunc("/api/users/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodDelete {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if _, err := requireAdmin(db, r); err != nil {
			writeAuthError(w, err)
			return
		}

		trimmed := strings.TrimPrefix(r.URL.Path, "/api/users/")
		parts := strings.Split(trimmed, "/")
		if r.Method == http.MethodDelete {
			if len(parts) != 1 || strings.TrimSpace(parts[0]) == "" {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			if err := store.DeleteUser(db, parts[0]); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					writeError(w, http.StatusNotFound, "user not found")
					return
				}
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
			return
		}

		if len(parts) != 2 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		username, action := parts[0], parts[1]
		var enabled bool
		switch action {
		case "enable":
			enabled = true
		case "disable":
			enabled = false
		default:
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		if err := store.SetUserEnabled(db, username, enabled); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusNotFound, "user not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": action + "d"})
	})

	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if _, err := requireUser(db, r); err != nil {
				writeAuthError(w, err)
				return
			}
			projects, err := store.ListProjects(db)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, projects)
		case http.MethodPost:
			user, err := requireUser(db, r)
			if err != nil {
				writeAuthError(w, err)
				return
			}
			var req projectRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "invalid json body")
				return
			}
			project, err := store.CreateProject(db, req.Title, req.Description, req.AcceptanceCriteria, user.ID)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, project)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	mux.HandleFunc("/api/projects/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := requireUser(db, r); err != nil {
			writeAuthError(w, err)
			return
		}

		trimmed := strings.TrimPrefix(r.URL.Path, "/api/projects/")
		parts := strings.Split(trimmed, "/")
		if len(parts) == 2 && parts[1] == "tasks" && r.Method == http.MethodGet {
			project, err := store.GetProject(db, parts[0])
			if err != nil {
				if errors.Is(err, store.ErrProjectNotFound) {
					writeError(w, http.StatusNotFound, err.Error())
					return
				}
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			limit := 0
			if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
				if _, err := fmt.Sscan(raw, &limit); err != nil {
					writeError(w, http.StatusBadRequest, "limit must be numeric")
					return
				}
			}
			tasks, err := store.ListTasks(db, store.TaskListParams{
				ProjectID: project.ID,
				Type:      r.URL.Query().Get("type"),
				Status:    r.URL.Query().Get("status"),
				Search:    r.URL.Query().Get("q"),
				Assignee:  r.URL.Query().Get("assignee"),
				Limit:     limit,
			})
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, tasks)
			return
		}

		if len(parts) == 2 && r.Method == http.MethodPost {
			if _, err := requireAdmin(db, r); err != nil {
				writeAuthError(w, err)
				return
			}
			var id int64
			if _, err := fmt.Sscan(parts[0], &id); err != nil {
				writeError(w, http.StatusNotFound, "project not found")
				return
			}
			var enabled bool
			switch parts[1] {
			case "enable":
				enabled = true
			case "disable":
				enabled = false
			default:
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			project, err := store.SetProjectStatus(db, id, enabled)
			if err != nil {
				if errors.Is(err, store.ErrProjectNotFound) {
					writeError(w, http.StatusNotFound, err.Error())
					return
				}
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, project)
			return
		}

		if len(parts) != 1 {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		switch r.Method {
		case http.MethodGet:
			project, err := store.GetProject(db, parts[0])
			if err != nil {
				if errors.Is(err, store.ErrProjectNotFound) {
					writeError(w, http.StatusNotFound, err.Error())
					return
				}
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, project)
		case http.MethodPut:
			if _, err := requireAdmin(db, r); err != nil {
				writeAuthError(w, err)
				return
			}
			var id int64
			if _, err := fmt.Sscan(parts[0], &id); err != nil {
				writeError(w, http.StatusNotFound, "project not found")
				return
			}
			var req projectRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "invalid json body")
				return
			}
			project, err := store.UpdateProject(db, id, req.Title, req.Description, req.AcceptanceCriteria)
			if err != nil {
				if errors.Is(err, store.ErrProjectNotFound) {
					writeError(w, http.StatusNotFound, err.Error())
					return
				}
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, project)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		user, err := requireUser(db, r)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		var req taskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		task, err := store.CreateTask(db, store.TaskCreateParams{
			ProjectID:          req.ProjectID,
			ParentID:           req.ParentID,
			Type:               req.Type,
			Title:              req.Title,
			Description:        req.Description,
			AcceptanceCriteria: req.AcceptanceCriteria,
			Priority:           req.Priority,
			Assignee:           req.Assignee,
			Status:             req.Status,
			CreatedBy:          user.ID,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, task)
	})

	mux.HandleFunc("/api/tasks/request", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		user, err := requireUser(db, r)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		var req taskAssignRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		task, status, err := store.RequestTask(db, store.TaskRequestParams{
			ProjectID: req.ProjectID,
			TaskID:    req.TaskID,
			Username:  user.Username,
			UserID:    user.ID,
		})
		if err != nil {
			if errors.Is(err, store.ErrTaskNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		payload := map[string]any{"status": status}
		if status == "ASSIGNED" {
			payload["task"] = task
		}
		writeJSON(w, http.StatusOK, payload)
	})

	mux.HandleFunc("/api/tasks/", func(w http.ResponseWriter, r *http.Request) {
		user, err := requireUser(db, r)
		if err != nil {
			writeAuthError(w, err)
			return
		}

		trimmed := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
		parts := strings.Split(trimmed, "/")
		var id int64
		if _, err := fmt.Sscan(parts[0], &id); err != nil {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}

		if len(parts) == 2 && parts[1] == "history" && r.Method == http.MethodGet {
			events, err := store.ListHistoryEvents(db, id)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, events)
			return
		}

		if len(parts) == 2 && parts[1] == "comments" {
			switch r.Method {
			case http.MethodGet:
				comments, err := store.ListComments(db, id)
				if err != nil {
					writeError(w, http.StatusInternalServerError, err.Error())
					return
				}
				writeJSON(w, http.StatusOK, comments)
			case http.MethodPost:
				var req commentRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					writeError(w, http.StatusBadRequest, "invalid json body")
					return
				}
				comment, err := store.AddComment(db, id, user.ID, req.Comment)
				if err != nil {
					writeError(w, http.StatusBadRequest, err.Error())
					return
				}
				task, err := store.GetTask(db, id)
				if err == nil {
					_ = store.AddHistoryEvent(db, task.ProjectID, id, "comment_added", map[string]any{
						"comment_id": comment.ID,
					}, user.ID)
				}
				writeJSON(w, http.StatusCreated, comment)
			default:
				writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
			return
		}

		if len(parts) == 2 && parts[1] == "dependencies" && r.Method == http.MethodGet {
			dependencies, err := store.ListDependencies(db, id)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, dependencies)
			return
		}

		if len(parts) == 2 && parts[1] == "clone" && r.Method == http.MethodPost {
			cloned, err := store.CloneTask(db, id, user.ID)
			if err != nil {
				if errors.Is(err, store.ErrTaskNotFound) {
					writeError(w, http.StatusNotFound, err.Error())
					return
				}
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, cloned)
			return
		}

		switch r.Method {
		case http.MethodGet:
			task, err := store.GetTask(db, id)
			if err != nil {
				if errors.Is(err, store.ErrTaskNotFound) {
					writeError(w, http.StatusNotFound, err.Error())
					return
				}
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, task)
		case http.MethodPut:
			var req taskRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "invalid json body")
				return
			}
			task, err := store.UpdateTask(db, id, store.TaskUpdateParams{
				Title:              req.Title,
				Description:        req.Description,
				AcceptanceCriteria: req.AcceptanceCriteria,
				ParentID:           req.ParentID,
				Assignee:           req.Assignee,
				Status:             req.Status,
				Priority:           req.Priority,
				Order:              req.Order,
				UpdatedBy:          user.ID,
				ActorUsername:      user.Username,
				ActorRole:          user.Role,
			})
			if err != nil {
				if errors.Is(err, store.ErrTaskNotFound) {
					writeError(w, http.StatusNotFound, err.Error())
					return
				}
				if errors.Is(err, store.ErrAdminRequired) || errors.Is(err, store.ErrForbidden) {
					writeAuthError(w, err)
					return
				}
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, task)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	mux.HandleFunc("/api/dependencies", func(w http.ResponseWriter, r *http.Request) {
		user, err := requireUser(db, r)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		switch r.Method {
		case http.MethodPost:
			var req dependencyRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "invalid json body")
				return
			}
			dependency, err := store.AddDependency(db, req.ProjectID, req.TaskID, req.DependsOn, user.ID)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, dependency)
		case http.MethodDelete:
			var projectID, taskID, dependsOn int64
			if _, err := fmt.Sscan(strings.TrimSpace(r.URL.Query().Get("project_id")), &projectID); err != nil {
				writeError(w, http.StatusBadRequest, "project_id must be numeric")
				return
			}
			if _, err := fmt.Sscan(strings.TrimSpace(r.URL.Query().Get("task_id")), &taskID); err != nil {
				writeError(w, http.StatusBadRequest, "task_id must be numeric")
				return
			}
			if _, err := fmt.Sscan(strings.TrimSpace(r.URL.Query().Get("depends_on")), &dependsOn); err != nil {
				writeError(w, http.StatusBadRequest, "depends_on must be numeric")
				return
			}
			if err := store.DeleteDependency(db, projectID, taskID, dependsOn); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					writeError(w, http.StatusNotFound, "dependency not found")
					return
				}
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})
}

type credentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type projectRequest struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
}

type taskRequest struct {
	ProjectID          int64  `json:"project_id"`
	ParentID           *int64 `json:"parent_id"`
	Type               string `json:"type"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
	Status             string `json:"status"`
	Priority           int    `json:"priority"`
	Order              int    `json:"order"`
	Assignee           string `json:"assignee"`
}

type commentRequest struct {
	Comment string `json:"comment"`
}

type dependencyRequest struct {
	ProjectID int64 `json:"project_id"`
	TaskID    int64 `json:"task_id"`
	DependsOn int64 `json:"depends_on"`
}

type taskAssignRequest struct {
	ProjectID int64  `json:"project_id"`
	TaskID    *int64 `json:"task_id"`
}

type authResponse struct {
	Token string     `json:"token"`
	User  store.User `json:"user"`
}

func userFromRequest(db *sql.DB, r *http.Request) (store.User, error) {
	return store.GetUserByToken(db, bearerToken(r))
}

func requireUser(db *sql.DB, r *http.Request) (store.User, error) {
	return userFromRequest(db, r)
}

func requireAdmin(db *sql.DB, r *http.Request) (store.User, error) {
	user, err := requireUser(db, r)
	if err != nil {
		return store.User{}, err
	}
	if user.Role != "admin" {
		return store.User{}, store.ErrAdminRequired
	}
	return user, nil
}

func bearerToken(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, store.ErrAdminRequired):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, store.ErrForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}
