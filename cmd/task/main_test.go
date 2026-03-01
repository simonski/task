package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/simonski/task/internal/store"
)

func TestRenderRootUsageShowsMainCommandsOnly(t *testing.T) {
	usage := renderRootUsage()

	for _, want := range []string{
		"████████╗",
		"USAGE",
		"CLIENT COMMANDS",
		"ADMIN COMMANDS",
		"initdb",
		"server",
		"version",
		"login",
		"project",
		"help",
	} {
		if !strings.Contains(usage, want) {
			t.Fatalf("root usage missing %q:\n%s", want, usage)
		}
	}

	if !strings.Contains(usage, "\x1b[31m") {
		t.Fatalf("root usage should contain ANSI color banner:\n%s", usage)
	}

	for _, unwanted := range []string{
		"accept requirement",
		"spec export markdown",
	} {
		if strings.Contains(usage, unwanted) {
			t.Fatalf("root usage should not include detailed subcommand %q:\n%s", unwanted, usage)
		}
	}

	clientOrder := []string{
		"  add",
		"  claim",
		"  comment",
		"  count",
		"  dependency",
		"  get",
		"  help",
		"  list",
		"  login",
		"  logout",
		"  onboard",
		"  orphans",
		"  project",
		"  register",
		"  search",
		"  status",
		"  unclaim",
		"  version",
	}
	last := -1
	for _, item := range clientOrder {
		idx := strings.Index(usage, item)
		if idx == -1 {
			t.Fatalf("root usage missing ordered client command %q:\n%s", item, usage)
		}
		if idx <= last {
			t.Fatalf("root usage client commands not alphabetical around %q:\n%s", item, usage)
		}
		last = idx
	}

	adminOrder := []string{"  assign", "  initdb", "  server", "  unassign", "  user"}
	last = -1
	for _, item := range adminOrder {
		idx := strings.Index(usage, item)
		if idx == -1 {
			t.Fatalf("root usage missing ordered admin command %q:\n%s", item, usage)
		}
		if idx <= last {
			t.Fatalf("root usage admin commands not alphabetical around %q:\n%s", item, usage)
		}
		last = idx
	}

	for _, unwanted := range []string{"ALIASES", "create,new", "del,delete", "  ls", "  show"} {
		if strings.Contains(usage, unwanted) {
			t.Fatalf("root usage should not include aliases %q:\n%s", unwanted, usage)
		}
	}
}

func TestParseIDListSupportsCommaSeparatedValues(t *testing.T) {
	ids, err := parseIDList("1, 2,3")
	if err != nil {
		t.Fatalf("parseIDList() error = %v", err)
	}
	if len(ids) != 3 || ids[0] != 1 || ids[1] != 2 || ids[2] != 3 {
		t.Fatalf("parseIDList() = %#v", ids)
	}
}

func TestRenderBannerContainsTaskArtAndColors(t *testing.T) {
	banner := renderBanner()
	if !strings.Contains(banner, "████████╗") {
		t.Fatalf("banner missing TASK art:\n%s", banner)
	}
	if !strings.Contains(banner, "\x1b[35m") {
		t.Fatalf("banner missing rainbow ANSI colors:\n%s", banner)
	}
}

func TestRenderCommandHelpIncludesUsageAndExample(t *testing.T) {
	help := renderCommandHelp("initdb")

	for _, want := range []string{
		"USAGE",
		"task initdb",
		"DETAILS",
		"EXAMPLE",
		"task initdb -f $TASK_HOME/task.db --force -password secret",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("command help missing %q:\n%s", want, help)
		}
	}
}

func TestRunOnboardAppendsEmbeddedAgentsTemplate(t *testing.T) {
	tempDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir(tempDir) error = %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	target := filepath.Join(tempDir, "AGENTS.md")
	if err := os.WriteFile(target, []byte("existing"), 0o644); err != nil {
		t.Fatalf("WriteFile(existing AGENTS.md) error = %v", err)
	}

	if err := runOnboard(nil); err != nil {
		t.Fatalf("runOnboard() error = %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile(AGENTS.md) error = %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "existing\n# Agent Instructions") {
		t.Fatalf("runOnboard() did not append embedded template correctly:\n%s", content)
	}
}

func TestRenderServerHelpIncludesTaskHomeDefault(t *testing.T) {
	help := renderCommandHelp("server")
	for _, want := range []string{
		"task server [-f <db-path>] [-addr :8080]",
		"If `-f` is omitted, the server uses `$TASK_HOME/task.db`.",
		"task server -f $TASK_HOME/task.db -addr :8080",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("server help missing %q:\n%s", want, help)
		}
	}
}

func TestRenderUserHelpIncludesAdmin403Message(t *testing.T) {
	help := renderCommandHelp("user")
	for _, want := range []string{
		"task user <create|ls|list|rm|delete|enable|disable>",
		"user is not an admin",
		"task user create --username alice --password secret",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("user help missing %q:\n%s", want, help)
		}
	}
}

func TestResolveCredentialsUsesFlagsEnvAndDefaults(t *testing.T) {
	t.Setenv("TASK_USERNAME", "env-user")
	t.Setenv("TASK_PASSWORD", "env-pass")

	username, password, err := resolveCredentials("", "", true)
	if err != nil {
		t.Fatalf("resolveCredentials(env) error = %v", err)
	}
	if username != "env-user" || password != "env-pass" {
		t.Fatalf("resolveCredentials(env) = %q/%q", username, password)
	}

	username, password, err = resolveCredentials("flag-user", "flag-pass", true)
	if err != nil {
		t.Fatalf("resolveCredentials(flags) error = %v", err)
	}
	if username != "flag-user" || password != "flag-pass" {
		t.Fatalf("resolveCredentials(flags) = %q/%q", username, password)
	}

	t.Setenv("TASK_USERNAME", "")
	t.Setenv("TASK_PASSWORD", "")
	username, password, err = resolveCredentials("", "", true)
	if err != nil {
		t.Fatalf("resolveCredentials(defaults) error = %v", err)
	}
	if password != "password" {
		t.Fatalf("resolveCredentials(default password) = %q", password)
	}
	if username == "" {
		t.Fatal("resolveCredentials(default username) returned empty username")
	}
}

func TestExtractURLOverride(t *testing.T) {
	args, override, err := extractURLOverride([]string{"login", "-username", "simon", "-url", "http://example.test:9000"})
	if err != nil {
		t.Fatalf("extractURLOverride() error = %v", err)
	}
	if override != "http://example.test:9000" {
		t.Fatalf("extractURLOverride() override = %q", override)
	}
	got := strings.Join(args, " ")
	if got != "login -username simon" {
		t.Fatalf("extractURLOverride() args = %q", got)
	}
}

func TestEmbeddedVersionMatchesBuildVersionFile(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("VERSION"))
	if err != nil {
		t.Fatalf("ReadFile(VERSION) error = %v", err)
	}
	if strings.TrimSpace(embeddedVersion) != strings.TrimSpace(string(data)) {
		t.Fatalf("embeddedVersion = %q, want %q", strings.TrimSpace(embeddedVersion), strings.TrimSpace(string(data)))
	}
}

func TestRunInitDBGeneratesPasswordWhenMissing(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TASK_HOME", tempDir)
	dbPath := filepath.Join(tempDir, "task.db")

	output := captureStdout(t, func() {
		if err := runInitDB([]string{"-f", dbPath}); err != nil {
			t.Fatalf("runInitDB() error = %v", err)
		}
	})

	if !strings.Contains(output, "admin user: admin") {
		t.Fatalf("runInitDB() output missing admin user:\n%s", output)
	}
	if !strings.Contains(output, "admin password: ") {
		t.Fatalf("runInitDB() output missing password:\n%s", output)
	}
	if !strings.Contains(output, "generated because -password was not provided") {
		t.Fatalf("runInitDB() output missing generated-password note:\n%s", output)
	}
}

func TestRunInitDBUsesTaskHomeWhenFIsOmitted(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TASK_HOME", tempDir)

	if err := runInitDB([]string{"-password", "secret"}); err != nil {
		t.Fatalf("runInitDB() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(tempDir, "task.db")); err != nil {
		t.Fatalf("expected default db at TASK_HOME/task.db: %v", err)
	}
}

func TestRunInitDBForceOverwritesExistingDatabase(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TASK_HOME", tempDir)
	dbPath := filepath.Join(tempDir, "task.db")

	if err := runInitDB([]string{"-f", dbPath, "-password", "first-pass"}); err != nil {
		t.Fatalf("first runInitDB() error = %v", err)
	}
	if err := runInitDB([]string{"-f", dbPath, "-password", "second-pass"}); err == nil {
		t.Fatal("second runInitDB() without --force = nil, want error")
	}
	if err := runInitDB([]string{"-f", dbPath, "--force", "-password", "second-pass"}); err != nil {
		t.Fatalf("forced runInitDB() error = %v", err)
	}
}

func TestPromptForCredentials(t *testing.T) {
	username, password, err := promptForCredentials(strings.NewReader("alice\nsecret\n"), ioDiscard{}, "", "")
	if err != nil {
		t.Fatalf("promptForCredentials() error = %v", err)
	}
	if username != "alice" || password != "secret" {
		t.Fatalf("promptForCredentials() = %q/%q", username, password)
	}
}

func TestPromptForCredentialsUsesDefaultsWhenInputIsEmpty(t *testing.T) {
	username, password, err := promptForCredentials(strings.NewReader("\n\n"), ioDiscard{}, "alice", "secret")
	if err != nil {
		t.Fatalf("promptForCredentials(defaults) error = %v", err)
	}
	if username != "alice" || password != "secret" {
		t.Fatalf("promptForCredentials(defaults) = %q/%q", username, password)
	}
}

func TestLoginRetryStoresCredentialsSeparatelyAndLogoutRemovesThem(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TASK_HOME", tempDir)
	credsPath := filepath.Join(tempDir, "credentials.json")
	t.Setenv("TASK_URL", "")

	var loginAttempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/login":
			var req map[string]string
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("Decode(login) error = %v", err)
			}
			attempt := atomic.AddInt32(&loginAttempts, 1)
			if attempt == 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"invalid credentials"}`))
				return
			}
			if req["username"] != "alice" || req["password"] != "secret" {
				t.Fatalf("retry login payload = %#v", req)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token":"session-token","user":{"username":"alice","role":"user"}}`))
		case "/api/logout":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"logged_out"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("TASK_URL", server.URL)

	oldIn := loginPromptInput
	oldOut := loginPromptOutput
	loginPromptInput = strings.NewReader("alice\nsecret\n")
	loginPromptOutput = ioDiscard{}
	t.Cleanup(func() {
		loginPromptInput = oldIn
		loginPromptOutput = oldOut
	})

	output := captureStdout(t, func() {
		if err := runLogin([]string{"-username", "alice", "-password", "wrong"}); err != nil {
			t.Fatalf("runLogin() error = %v", err)
		}
	})
	if !strings.Contains(output, "invalid credentials") {
		t.Fatalf("runLogin() output missing invalid credentials:\n%s", output)
	}
	if !strings.Contains(output, "logged in as alice") {
		t.Fatalf("runLogin() output missing success:\n%s", output)
	}

	configData, err := os.ReadFile(filepath.Join(tempDir, "config.json"))
	if err != nil {
		t.Fatalf("ReadFile(config.json) error = %v", err)
	}
	if strings.Contains(string(configData), "session-token") {
		t.Fatalf("config.json should not contain session token:\n%s", string(configData))
	}
	if !strings.Contains(string(configData), `"username": "alice"`) {
		t.Fatalf("config.json should contain username alice:\n%s", string(configData))
	}
	if !strings.Contains(string(configData), `"server_url": "`+server.URL+`"`) {
		t.Fatalf("config.json should contain resolved server URL %q:\n%s", server.URL, string(configData))
	}
	credData, err := os.ReadFile(credsPath)
	if err != nil {
		t.Fatalf("ReadFile(credentials.json) error = %v", err)
	}
	if !strings.Contains(string(credData), "session-token") {
		t.Fatalf("credentials.json missing session token:\n%s", string(credData))
	}

	if err := runLogout(nil); err != nil {
		t.Fatalf("runLogout() error = %v", err)
	}
	if _, err := os.Stat(credsPath); !os.IsNotExist(err) {
		t.Fatalf("credentials.json should be removed after logout, err=%v", err)
	}
}

func TestRunLoginUsesValidStoredCredentialsFirst(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TASK_HOME", tempDir)
	t.Setenv("TASK_URL", "")

	if err := os.WriteFile(filepath.Join(tempDir, "config.json"), []byte(`{"username":"alice"}`), 0o600); err != nil {
		t.Fatalf("WriteFile(config.json) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "credentials.json"), []byte(`{"token":"stored-token"}`), 0o600); err != nil {
		t.Fatalf("WriteFile(credentials.json) error = %v", err)
	}

	var loginCalls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/status":
			if r.Header.Get("Authorization") != "Bearer stored-token" {
				t.Fatalf("status auth header = %q", r.Header.Get("Authorization"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","authenticated":true,"user":{"username":"alice","role":"user"}}`))
		case "/api/login":
			atomic.AddInt32(&loginCalls, 1)
			t.Fatal("runLogin should not call /api/login when stored credentials are valid")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("TASK_URL", server.URL)

	output := captureStdout(t, func() {
		if err := runLogin(nil); err != nil {
			t.Fatalf("runLogin() error = %v", err)
		}
	})
	if !strings.Contains(output, "logged in as alice") {
		t.Fatalf("runLogin() output = %q", output)
	}
	if atomic.LoadInt32(&loginCalls) != 0 {
		t.Fatalf("unexpected login calls = %d", loginCalls)
	}
	configData, err := os.ReadFile(filepath.Join(tempDir, "config.json"))
	if err != nil {
		t.Fatalf("ReadFile(config.json) error = %v", err)
	}
	if !strings.Contains(string(configData), `"server_url": "`+server.URL+`"`) {
		t.Fatalf("config.json should contain resolved server URL %q:\n%s", server.URL, string(configData))
	}
}

func TestPrintTaskDetailsIncludesAcceptanceCriteria(t *testing.T) {
	output := captureStdout(t, func() {
		printTaskDetails(store.Task{
			ID:                 42,
			Title:              "Example Task",
			Type:               "task",
			Status:             "open",
			Description:        "Example description",
			ProjectID:          7,
			Priority:           1,
			CreatedAt:          "2026-03-01 12:00:00",
			UpdatedAt:          "2026-03-02 09:30:00",
			AcceptanceCriteria: "- does the thing\n- handles the edge case",
		}, nil)
	})

	for _, want := range []string{
		"ID           : 42",
		"Type         : task",
		"Description  : Example description",
		"ParentID     : ",
		"ProjectID    : 7",
		"Title        : Example Task",
		"Assignee     : ",
		"Order        : ",
		"DependsOn    : []",
		"Status       : open",
		"Priority     : 1",
		"Created      : 2026-03-01 12:00:00",
		"LastModified : 2026-03-02 09:30:00",
		"Closed       : ",
		"Acceptance Criteria : - does the thing",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("printTaskDetails() missing %q:\n%s", want, output)
		}
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = old })

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("ReadFrom() error = %v", err)
	}
	return buf.String()
}
