package main

import (
	"bufio"
	"crypto/rand"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	osuser "os/user"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/simonski/task/internal/client"
	"github.com/simonski/task/internal/config"
	"github.com/simonski/task/internal/server"
	"github.com/simonski/task/internal/store"
	"golang.org/x/term"
)

type commandHelp struct {
	usage   string
	details []string
	example string
}

var (
	loginPromptInput  io.Reader = os.Stdin
	loginPromptOutput io.Writer = os.Stdout
	outputJSON        bool
	noColorOutput     bool
	runAgentCommand   = defaultRunAgentCommand
)

var bannerLines = []string{
	"████████╗ █████╗ ███████╗██╗  ██╗",
	"╚══██╔══╝██╔══██╗██╔════╝██║ ██╔╝",
	"   ██║   ███████║███████╗█████╔╝ ",
	"   ██║   ██╔══██║╚════██║██╔═██╗ ",
	"   ██║   ██║  ██║███████║██║  ██╗",
	"   ╚═╝   ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝",
}

var bannerColors = []string{
	"\x1b[31m",
	"\x1b[33m",
	"\x1b[32m",
	"\x1b[36m",
	"\x1b[34m",
	"\x1b[35m",
}

//go:embed VERSION
var embeddedVersion string

//go:embed AGENTS.md
var embeddedAgents string

var helpIndex = map[string]commandHelp{
	"onboard": {
		usage:   "task onboard",
		details: []string{"Appends the embedded onboarding template to `${CWD}/AGENTS.md`.", "Creates `${CWD}/AGENTS.md` if it does not already exist."},
		example: "task onboard",
	},
	"initdb": {
		usage:   "task initdb [-f <db-path>] [--force] [-password <password>]",
		details: []string{"Creates a new SQLite database, bootstraps the fixed `admin` account, and creates the default project.", "If `-f` is omitted, the database is created at `$TASK_HOME/task.db`.", "If `-password` is omitted, a random admin password is generated and printed to stdout.", "If `--force` is supplied, any existing database file is overwritten."},
		example: "task initdb -f $TASK_HOME/task.db --force -password secret",
	},
	"server": {
		usage:   "task server [-f <db-path>] [-addr :8080] [-v]",
		details: []string{"Starts the HTTP API server and the embedded web UI.", "If `-f` is omitted, the server uses `$TASK_HOME/task.db`.", "If `-v` is supplied, requests and responses are printed verbosely to stdout."},
		example: "task server -f $TASK_HOME/task.db -addr :8080 -v",
	},
	"version": {
		usage:   "task version",
		details: []string{"Prints the semantic version embedded into the binary from the build-time `VERSION` file."},
		example: "task version",
	},
	"login": {
		usage:   "task login [-username <name>] [-password <password>] [-url <server-url>]",
		details: []string{"Logs into the configured server and stores the session token in `$TASK_HOME/credentials.json`.", "Login resolution order: valid `$TASK_HOME/credentials.json`, then `username` in `$TASK_HOME/config.json`, then `-username` / `-password`, then `TASK_USERNAME` / `TASK_PASSWORD`, then prompts.", "If prompting is needed, discovered values are used as editable defaults.", "URL resolution: `-url`, then `TASK_URL`, then configured URL, then `http://localhost:8080`."},
		example: "task login -username simon -password secret -url http://localhost:8080",
	},
	"register": {
		usage:   "task register [-username <name>] [-password <password>] [-url <server-url>]",
		details: []string{"Creates a user account on the configured server but does not log the user in.", "Credential resolution: `-username`, then `TASK_USERNAME`, then OS `whoami`; `-password`, then `TASK_PASSWORD`, then `password`."},
		example: "task register -username simon -password secret",
	},
	"logout": {
		usage:   "task logout [-url <server-url>]",
		details: []string{"Logs out from the configured server and removes `$TASK_HOME/credentials.json`."},
		example: "task logout",
	},
	"status": {
		usage:   "task status [-url <server-url>]",
		details: []string{"Pings the server and shows authentication state, server version, and client version.", "Warns when the server version differs from the client version."},
		example: "task status",
	},
	"help": {
		usage:   "task help <command>",
		details: []string{"Shows command-specific help when available.", "Without a command, prints the root usage summary."},
		example: "task help dependency",
	},
	"count": {
		usage:   "task count [-project_id <id>] [-url <server-url>]",
		details: []string{"Counts users and work items by type.", "With `-project_id`, counts work items within that project and omits the global project total."},
		example: "task count -project_id 1",
	},
	"req": {
		usage:   "task req -f <file1,file2,...> -o <output-file> [-agent <agent>]",
		details: []string{"Reads the listed input files, sends a requirements-breakdown prompt to an agent, and writes the agent output to the requested output file.", "Default agent is `codex`, which is invoked as `codex exec <prompt>`. Other agents are invoked as `<agent> -p <prompt>`."},
		example: "task req -f README.md,docs/DESIGN.md -o requirements.md",
	},
	"project": {
		usage:   "task project <create|list|get|use>|<id> <update|enable|disable>",
		details: []string{"Manages projects and the active project context used by subsequent commands.", "Projects are addressed by numeric id."},
		example: "task project 3 update -title \"Customer Portal\"",
	},
	"list": {
		usage:   "task list|ls [--type <type>] [--status <status>] [-u <user>] [-n <limit>] [-url <server-url>]",
		details: []string{"Lists tasks in the active project with optional type, status, assignee, and limit filters.", "`-n` is applied server-side. `0` means no limit."},
		example: "task list --type bug --status open -u alice -n 20",
	},
	"orphans": {
		usage:   "task orphans [-url <server-url>]",
		details: []string{"Lists tasks in the active project that do not have a parent task or epic."},
		example: "task orphans",
	},
	"get": {
		usage:   "task get <id> [-url <server-url>]",
		details: []string{"Shows a single task with comments and history.", "Output uses subtle color unless `-nocolor` is supplied."},
		example: "task get 42",
	},
	"show": {
		usage:   "task show <id> [-url <server-url>]",
		details: []string{"Alias for `task get`."},
		example: "task show 42",
	},
	"search": {
		usage:   "task search \"query\" [-url <server-url>]",
		details: []string{"Searches task titles and descriptions within the active project."},
		example: "task search \"password reset\"",
	},
	"update": {
		usage:   "task update <id> -status <status>",
		details: []string{"Updates a task.", "Current CLI support is focused on `-status` and accepts `notready`, `open`, `inprogress`, `complete`, and `fail`."},
		example: "task update 42 -status inprogress",
	},
	"open": {
		usage:   "task open <id>",
		details: []string{"Sets the task status to `open`."},
		example: "task open 42",
	},
	"ready": {
		usage:   "task ready <id>",
		details: []string{"Alias for `task open <id>`.", "Use this when marking a task as ready for work."},
		example: "task ready 42",
	},
	"inprogress": {
		usage:   "task inprogress <id>",
		details: []string{"Sets the task status to `inprogress`."},
		example: "task inprogress 42",
	},
	"complete": {
		usage:   "task complete <id>",
		details: []string{"Sets the task status to `complete`."},
		example: "task complete 42",
	},
	"fail": {
		usage:   "task fail <id>",
		details: []string{"Sets the task status to `fail`."},
		example: "task fail 42",
	},
	"add": {
		usage:   "task add|create|new [-title <title>] [-t <type>] [-p <priority>] [-a <assignee>] [-d <description>] [-ac <criteria>] [-parent <id>] [-project <project>] [title words]",
		details: []string{"Creates a task-like entity in the active project.", "Positional title words and `-title` are equivalent ways to set the title.", "Defaults: `type=task`, `priority=1`, blank assignee, blank description, blank acceptance criteria, blank parent, current project."},
		example: "task add \"Customers can reset their password.\"",
	},
	"comment": {
		usage:   "task comment add <id> \"comment\" [-url <server-url>]",
		details: []string{"Adds a comment to a task and records a corresponding history event."},
		example: "task comment add 42 \"Need product sign-off.\"",
	},
	"clone": {
		usage:   "task clone|cp <id>",
		details: []string{"Clones a task or epic.", "Cloned items are unassigned, set to `notready`, and keep a `clone_of` reference to the source item. Cloning an epic also clones its child tasks."},
		example: "task clone 42",
	},
	"assign": {
		usage:   "task assign <id> <name>",
		details: []string{"Admin-only command that assigns a task to a user.", "The target user must exist and be enabled."},
		example: "task assign 42 alice",
	},
	"unassign": {
		usage:   "task unassign <id> <name>",
		details: []string{"Admin-only command that clears a task assignment from the named user.", "The named user must exist and be enabled."},
		example: "task unassign 42 alice",
	},
	"claim": {
		usage:   "task claim <id>",
		details: []string{"Assigns the caller to the task.", "Fails if the task is already assigned to another user."},
		example: "task claim 42",
	},
	"unclaim": {
		usage:   "task unclaim <id>",
		details: []string{"Clears the caller's assignment from the task.", "Fails unless the caller is the current assignee."},
		example: "task unclaim 42",
	},
	"add-dependency": {
		usage:   "task add-dependency <id> <dependency-id[,dependency-id...]>",
		details: []string{"Adds one or more `depends_on` links from the task to the listed task IDs.", "Comma-separated dependency IDs are supported."},
		example: "task add-dependency 4 1,2,3",
	},
	"remove-dependency": {
		usage:   "task remove-dependency <id> <dependency-id[,dependency-id...]>",
		details: []string{"Removes one or more `depends_on` links from the task to the listed task IDs.", "Comma-separated dependency IDs are supported."},
		example: "task remove-dependency 4 2",
	},
	"dependency": {
		usage:   "task dependency <add|remove> <id> <dependency-id[,dependency-id...]>",
		details: []string{"Manages `depends_on` links for a task.", "`add` creates dependency links; `remove` deletes them."},
		example: "task dependency add 4 1,2,3",
	},
	"request": {
		usage:   "task request [<id>]",
		details: []string{"Requests work for the current user.", "With an id, the server attempts to assign that specific task. Without an id, it assigns the oldest unassigned `open` task in the active project, unless the user already has assigned work to resume."},
		example: "task request 42",
	},
	"user": {
		usage:   "task user <create|ls|list|rm|delete|enable|disable>",
		details: []string{"Admin-only user management commands.", "If a non-admin user calls these commands, the server returns 403 with `user is not an admin`."},
		example: "task user create --username alice --password secret",
	},
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		if strings.HasPrefix(err.Error(), "no such command") {
			fmt.Fprintln(os.Stderr, err.Error())
		} else {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
		os.Exit(1)
	}
}

func run(args []string) error {
	trimmedArgs, urlOverride, err := extractURLOverride(args)
	if err != nil {
		return err
	}
	trimmedArgs, outputJSON, noColorOutput, err = extractOutputFlags(trimmedArgs)
	if err != nil {
		return err
	}
	if urlOverride != "" {
		if err := os.Setenv("TASK_URL", urlOverride); err != nil {
			return err
		}
	}
	if len(trimmedArgs) == 0 {
		fmt.Print(renderRootUsage())
		return nil
	}

	switch trimmedArgs[0] {
	case "help", "-h", "--help":
		return runHelp(trimmedArgs[1:])
	case "onboard":
		return runOnboard(trimmedArgs[1:])
	case "init":
		return errors.New("use `task initdb`")
	case "initdb":
		return runInitDB(trimmedArgs[1:])
	case "server":
		return runServer(trimmedArgs[1:])
	case "version":
		return runVersion(trimmedArgs[1:])
	case "register":
		return runRegister(trimmedArgs[1:])
	case "login":
		return runLogin(trimmedArgs[1:])
	case "logout":
		return runLogout(trimmedArgs[1:])
	case "status":
		return runStatus(trimmedArgs[1:])
	case "count":
		return runCount(trimmedArgs[1:])
	case "req":
		return runReq(trimmedArgs[1:])
	case "user":
		return runUser(trimmedArgs[1:])
	case "project":
		return runProject(trimmedArgs[1:])
	case "ls":
		return runList(trimmedArgs[1:])
	case "list":
		return runList(trimmedArgs[1:])
	case "orphans":
		return runOrphans(trimmedArgs[1:])
	case "get", "show":
		return runGet(trimmedArgs[1:])
	case "search":
		return runSearch(trimmedArgs[1:])
	case "update":
		return runUpdate(trimmedArgs[1:])
	case "set-status":
		return runSetStatus(trimmedArgs[1:])
	case "open":
		return runTaskStatusAlias(trimmedArgs[1:], "open", "open")
	case "ready":
		return runTaskStatusAlias(trimmedArgs[1:], "open", "ready")
	case "inprogress":
		return runTaskStatusAlias(trimmedArgs[1:], "inprogress", "inprogress")
	case "complete":
		return runTaskStatusAlias(trimmedArgs[1:], "complete", "complete")
	case "fail":
		return runTaskStatusAlias(trimmedArgs[1:], "fail", "fail")
	case "assign":
		return runAssign(trimmedArgs[1:])
	case "unassign":
		return runUnassign(trimmedArgs[1:])
	case "claim":
		return runClaim(trimmedArgs[1:])
	case "unclaim":
		return runUnclaim(trimmedArgs[1:])
	case "add-dependency":
		return runDependencyCommand(trimmedArgs[1:], true)
	case "remove-dependency":
		return runDependencyCommand(trimmedArgs[1:], false)
	case "dependency":
		return runDependency(trimmedArgs[1:])
	case "request":
		return runRequest(trimmedArgs[1:])
	case "history":
		return runHistory(trimmedArgs[1:])
	case "comment":
		return runComment(trimmedArgs[1:])
	case "clone", "cp":
		return runClone(trimmedArgs[1:])
	case "curate":
		return runCurate(trimmedArgs[1:])
	case "review":
		return runReview(trimmedArgs[1:])
	case "accept":
		return runRequirementStatus("accepted", trimmedArgs[1:])
	case "reject":
		return runRequirementStatus("rejected", trimmedArgs[1:])
	case "revise":
		return runRevise(trimmedArgs[1:])
	case "decision":
		return runDecision(trimmedArgs[1:])
	case "conversation":
		return runConversation(trimmedArgs[1:])
	case "add", "create", "new":
		return runTaskCreate(trimmedArgs[1:])
	case "note":
		return runTypedTaskCreate("note", trimmedArgs[1:])
	case "question":
		return runTypedTaskCreate("question", trimmedArgs[1:])
	case "bug":
		return runTypedTaskCreate("bug", trimmedArgs[1:])
	case "epic":
		return runTypedTaskCreate("epic", trimmedArgs[1:])
	case "config":
		return runConfig(trimmedArgs[1:])
	default:
		return fmt.Errorf("no such command %q", trimmedArgs[0])
	}
}

func runHelp(args []string) error {
	if len(args) == 0 {
		fmt.Print(renderRootUsage())
		return nil
	}
	if !hasCommandHelp(args[0]) {
		return fmt.Errorf("no such command %q", args[0])
	}
	fmt.Print(renderCommandHelp(args[0]))
	return nil
}

func runOnboard(args []string) error {
	if len(args) != 0 {
		return errors.New("usage: task onboard")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	target := filepath.Join(cwd, "AGENTS.md")
	var needsLeadingNewline bool
	if info, err := os.Stat(target); err == nil && info.Size() > 0 {
		existing, err := os.ReadFile(target)
		if err != nil {
			return err
		}
		if len(existing) > 0 && existing[len(existing)-1] != '\n' {
			needsLeadingNewline = true
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	f, err := os.OpenFile(target, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	if needsLeadingNewline {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	if _, err := f.WriteString(embeddedAgents); err != nil {
		return err
	}
	if !strings.HasSuffix(embeddedAgents, "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	if outputJSON {
		return printJSON(map[string]string{"status": "ok", "path": target})
	}
	fmt.Printf("appended onboarding template to %s\n", target)
	return nil
}

func runInitDB(args []string) error {
	fs := flag.NewFlagSet("initdb", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	defaultDBPath, err := defaultDatabasePath()
	if err != nil {
		return err
	}
	dbPath := fs.String("f", defaultDBPath, "SQLite database file")
	passwordFlag := fs.String("password", "", "bootstrap password")
	force := fs.Bool("force", false, "overwrite the database file if it exists")

	if err := fs.Parse(args); err != nil {
		return err
	}

	password := strings.TrimSpace(*passwordFlag)
	generated := false
	if password == "" {
		var err error
		password, err = generatePassword(24)
		if err != nil {
			return err
		}
		generated = true
	}

	if *force {
		if err := removeDBFiles(*dbPath); err != nil {
			return err
		}
	}

	if err := store.Init(*dbPath, "admin", password); err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.CurrentProject = "1"
	cfg.Username = "admin"
	cfg.ServerURL = config.ResolveServerURL(cfg)
	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Printf("initialized database at %s\n", *dbPath)
	fmt.Printf("admin user: admin\n")
	fmt.Printf("admin password: %s\n", password)
	fmt.Printf("default project: 1\n")
	if generated {
		fmt.Println("admin password was generated because -password was not provided")
	}
	return nil
}

func runServer(args []string) error {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	defaultDBPath, err := defaultDatabasePath()
	if err != nil {
		return err
	}
	dbPath := fs.String("f", defaultDBPath, "SQLite database file")
	addr := fs.String("addr", ":8080", "HTTP listen address")
	verbose := fs.Bool("v", false, "print verbose request/response logs to stdout")

	if err := fs.Parse(args); err != nil {
		return err
	}

	db, err := store.Open(*dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	srv, err := server.New(*addr, db, strings.TrimSpace(embeddedVersion), *verbose, os.Stdout)
	if err != nil {
		return err
	}

	fmt.Print(renderBanner())
	fmt.Printf("VERSION  %s\n", strings.TrimSpace(embeddedVersion))
	fmt.Printf("TASKDB   %s\n\n", *dbPath)
	fmt.Printf("serving task on http://localhost%s\n", *addr)
	return srv.ListenAndServe()
}

func runVersion(args []string) error {
	if len(args) != 0 {
		return errors.New("usage: task version")
	}
	fmt.Println(strings.TrimSpace(embeddedVersion))
	return nil
}

func runRegister(args []string) error {
	fs := flag.NewFlagSet("register", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	usernameFlag := fs.String("username", "", "username")
	passwordFlag := fs.String("password", "", "password")
	if err := fs.Parse(args); err != nil {
		return err
	}

	username, password, err := resolveCredentials(*usernameFlag, *passwordFlag, true)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	user, err := client.New(cfg).Register(username, password)
	if err != nil {
		return err
	}
	cfg.Username = user.Username
	if err := config.Save(cfg); err != nil {
		return err
	}
	if outputJSON {
		return printJSON(user)
	}
	fmt.Printf("registered user %s\n", user.Username)
	return nil
}

func runLogin(args []string) error {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	usernameFlag := fs.String("username", "", "username")
	passwordFlag := fs.String("password", "", "password")
	if err := fs.Parse(args); err != nil {
		return err
	}

	username, password, err := resolveCredentials(*usernameFlag, *passwordFlag, true)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	api := client.New(cfg)

	if cfg.Token != "" {
		status, err := api.Status()
		if err == nil && status.Authenticated && status.User != nil {
			cfg.Username = status.User.Username
			cfg.ServerURL = config.ResolveServerURL(cfg)
			if err := config.Save(cfg); err != nil {
				return err
			}
			if outputJSON {
				return printJSON(status)
			}
			fmt.Printf("logged in as %s\n", status.User.Username)
			return nil
		}
	}

	username = resolveLoginUsername(cfg.Username, *usernameFlag)
	password = resolveLoginPassword(*passwordFlag)

	if username != "" && password != "" {
		response, err := api.Login(username, password)
		if err == nil {
			return finishLogin(cfg, response)
		}
		if err.Error() != "invalid credentials" {
			return err
		}
		fmt.Println("invalid credentials")
	}

	username, password, err = promptForCredentials(loginPromptInput, loginPromptOutput, username, password)
	if err != nil {
		return err
	}
	response, err := api.Login(username, password)
	if err != nil {
		return err
	}
	return finishLogin(cfg, response)
}

func finishLogin(cfg config.Config, response client.AuthResponse) error {
	cfg.Username = response.User.Username
	cfg.ServerURL = config.ResolveServerURL(cfg)
	if err := config.Save(cfg); err != nil {
		return err
	}
	if err := config.SaveCredentials(config.Credentials{Token: response.Token}); err != nil {
		return err
	}
	if outputJSON {
		return printJSON(response)
	}
	fmt.Printf("logged in as %s\n", response.User.Username)
	return nil
}

func runLogout(args []string) error {
	if len(args) != 0 {
		return errors.New("usage: task logout")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := client.New(cfg).Logout(); err != nil {
		if clearErr := config.ClearCredentials(); clearErr != nil {
			return clearErr
		}
		cfg.Token = ""
		return err
	}
	if err := config.ClearCredentials(); err != nil {
		return err
	}
	cfg.Token = ""
	if err := config.Save(cfg); err != nil {
		return err
	}
	if outputJSON {
		return printJSON(map[string]string{"status": "logged_out"})
	}
	return nil
}

func runStatus(args []string) error {
	if len(args) != 0 {
		return errors.New("usage: task status")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	status, err := client.New(cfg).Status()
	if err != nil {
		return err
	}
	if outputJSON {
		return printJSON(status)
	}
	fmt.Printf("server: %s\n", config.ResolveServerURL(cfg))
	fmt.Printf("status: %s\n", status.Status)
	fmt.Printf("server_version: %s\n", status.ServerVersion)
	fmt.Printf("client_version: %s\n", strings.TrimSpace(embeddedVersion))
	fmt.Printf("authenticated: %t\n", status.Authenticated)
	if status.User != nil {
		fmt.Printf("user: %s (%s)\n", status.User.Username, status.User.Role)
	}
	if status.ServerVersion != "" && status.ServerVersion != strings.TrimSpace(embeddedVersion) {
		fmt.Printf("warning: server version %s differs from client version %s\n", status.ServerVersion, strings.TrimSpace(embeddedVersion))
	}
	return nil
}

func runCount(args []string) error {
	fs := flag.NewFlagSet("count", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	projectID := fs.Int64("project_id", 0, "limit counts to a project id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: task count [-project_id <id>]")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	api := client.New(cfg)
	var projectFilter *int64
	if *projectID != 0 {
		projectFilter = projectID
		if _, err := api.GetProject(fmt.Sprintf("%d", *projectID)); err != nil {
			return err
		}
	}
	summary, err := api.Count(projectFilter)
	if err != nil {
		return err
	}
	if outputJSON {
		return printJSON(summary)
	}
	printCountSummary(summary, projectFilter != nil)
	return nil
}

func runUser(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: task user <create|ls|list|rm|delete|enable|disable>")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	api := client.New(cfg)

	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("user create", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		usernameFlag := fs.String("username", "", "username")
		passwordFlag := fs.String("password", "", "password")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		username, password, err := resolveCredentials(*usernameFlag, *passwordFlag, true)
		if err != nil {
			return err
		}
		user, err := api.CreateUser(username, password)
		if err != nil {
			return err
		}
		if outputJSON {
			return printJSON(user)
		}
		fmt.Printf("created user %s\n", user.Username)
		return nil
	case "rm", "delete", "del":
		fs := flag.NewFlagSet("user "+args[0], flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		username := fs.String("username", "", "username")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *username == "" {
			return errors.New("user rm/delete/del requires -username")
		}
		if err := api.DeleteUser(*username); err != nil {
			return err
		}
		if outputJSON {
			return printJSON(map[string]string{"status": "deleted", "username": *username})
		}
		fmt.Printf("deleted user %s\n", *username)
		return nil
	case "enable", "disable":
		fs := flag.NewFlagSet("user "+args[0], flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		username := fs.String("username", "", "username")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *username == "" {
			return errors.New("user enable/disable requires -username")
		}
		if err := api.SetUserEnabled(*username, args[0] == "enable"); err != nil {
			return err
		}
		if outputJSON {
			return printJSON(map[string]string{"status": args[0] + "d", "username": *username})
		}
		fmt.Printf("%sd user %s\n", args[0], *username)
		return nil
	case "list", "ls":
		users, err := api.ListUsers()
		if err != nil {
			return err
		}
		if outputJSON {
			return printJSON(users)
		}
		for _, user := range users {
			fmt.Printf("%s\t%s\tenabled=%t\n", user.Username, user.Role, user.Enabled)
		}
		return nil
	default:
		return fmt.Errorf("unknown user command %q", args[0])
	}
}

func runProject(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	api := client.New(cfg)

	if len(args) == 0 {
		if cfg.CurrentProject == "" {
			fmt.Println("no active project")
			return nil
		}
		project, err := api.GetProject(cfg.CurrentProject)
		if err != nil {
			return err
		}
		if outputJSON {
			return printJSON(project)
		}
		printProject(project)
		return nil
	}

	if projectID, ok := parseProjectCommandID(args[0]); ok {
		return runProjectByID(api, projectID, args[1:])
	}

	switch args[0] {
	case "create", "add", "new":
		fs := flag.NewFlagSet("project create", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		description := fs.String("description", "", "project description")
		acceptanceCriteria := fs.String("ac", "", "project acceptance criteria")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return errors.New("usage: task project create [-description text] [-ac text] \"Project Title\"")
		}
		project, err := api.CreateProject(fs.Arg(0), *description, *acceptanceCriteria)
		if err != nil {
			return err
		}
		cfg.CurrentProject = strconv.FormatInt(project.ID, 10)
		cfg.CurrentEpicID = 0
		if err := config.Save(cfg); err != nil {
			return err
		}
		if outputJSON {
			return printJSON(project)
		}
		printProject(project)
		return nil
	case "list", "ls":
		projects, err := api.ListProjects()
		if err != nil {
			return err
		}
		if outputJSON {
			return printJSON(projects)
		}
		for _, project := range projects {
			current := ""
			if strconv.FormatInt(project.ID, 10) == cfg.CurrentProject {
				current = "\t(current)"
			}
			fmt.Printf("%d\t%s\t%s%s\n", project.ID, project.Title, project.Status, current)
		}
		return nil
	case "get":
		if len(args) != 2 {
			return errors.New("usage: task project get <id>")
		}
		project, err := api.GetProject(args[1])
		if err != nil {
			return err
		}
		if outputJSON {
			return printJSON(project)
		}
		printProject(project)
		return nil
	case "use":
		if len(args) != 2 {
			return errors.New("usage: task project use <id>")
		}
		project, err := api.GetProject(args[1])
		if err != nil {
			return err
		}
		cfg.CurrentProject = strconv.FormatInt(project.ID, 10)
		cfg.CurrentEpicID = 0
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Printf("using project %d\n", project.ID)
		return nil
	default:
		return fmt.Errorf("unknown project command %q", args[0])
	}
}

func runReq(args []string) error {
	fs := flag.NewFlagSet("req", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	filesArg := fs.String("f", "", "comma-separated input files")
	outputFile := fs.String("o", "", "output file")
	agent := fs.String("agent", "codex", "agent command")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*filesArg) == "" || strings.TrimSpace(*outputFile) == "" {
		return errors.New("usage: task req -f <file1,file2,...> -o <output-file> [-agent <agent>]")
	}

	files := splitCSV(*filesArg)
	if len(files) == 0 {
		return errors.New("at least one input file is required")
	}
	prompt, err := buildReqPrompt(files, *outputFile)
	if err != nil {
		return err
	}
	response, err := runAgentCommand(strings.TrimSpace(*agent), prompt)
	if err != nil {
		return err
	}
	if err := os.WriteFile(*outputFile, []byte(response), 0o644); err != nil {
		return err
	}
	fmt.Print(response)
	if response != "" && !strings.HasSuffix(response, "\n") {
		fmt.Println()
	}
	if outputJSON {
		return printJSON(map[string]string{
			"status": "ok",
			"agent":  strings.TrimSpace(*agent),
			"output": *outputFile,
		})
	}
	fmt.Printf("wrote %s using %s\n", *outputFile, strings.TrimSpace(*agent))
	return nil
}

func splitCSV(raw string) []string {
	var values []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

func buildReqPrompt(files []string, outputFile string) (string, error) {
	var b strings.Builder
	b.WriteString("Write an example breakdown of implementation requirements as ")
	b.WriteString(outputFile)
	b.WriteString(" in the format:\n\n")
	b.WriteString("EPIC: title\n")
	b.WriteString("ID: E1, E2, E3 etc\n")
	b.WriteString("DESCRIPTION: description\n")
	b.WriteString("AC: list of acceptance criteria\n")
	b.WriteString("PRIORITY: 1-N (1 highest, do this first)\n")
	b.WriteString("DEPENDS-ON: E2, E4\n\n")
	b.WriteString("<indent for stories \"in\" the epic (the story ID should increment and be EPIC-STORY)>\n")
	b.WriteString("    STORY: title\n")
	b.WriteString("    ID: E1-S1, E1-2, E1-S3 etc.\n")
	b.WriteString("    DESCRIPTION: description\n")
	b.WriteString("    AC: list of acceptance criteria\n")
	b.WriteString("    PRIORITY: 1-N (1 highest, do this first)\n")
	b.WriteString("    DEPENDS-ON: E1-S2\n\n")
	b.WriteString("Use the following input files as source material:\n\n")
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		b.WriteString("FILE: ")
		b.WriteString(file)
		b.WriteString("\n")
		b.WriteString("-----\n")
		b.Write(data)
		if len(data) == 0 || data[len(data)-1] != '\n' {
			b.WriteString("\n")
		}
		b.WriteString("-----\n\n")
	}
	return b.String(), nil
}

func defaultRunAgentCommand(agent, prompt string) (string, error) {
	if agent == "" {
		return "", errors.New("agent is required")
	}
	var cmd *exec.Cmd
	if agent == "codex" {
		cmd = exec.Command("codex", "exec", prompt)
	} else {
		cmd = exec.Command(agent, "-p", prompt)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return "", err
		}
		return "", fmt.Errorf("%v: %s", err, message)
	}
	return string(output), nil
}

func parseProjectCommandID(raw string) (int64, bool) {
	var id int64
	if _, err := fmt.Sscan(raw, &id); err != nil {
		return 0, false
	}
	return id, true
}

func runProjectByID(api *client.Client, projectID int64, args []string) error {
	if len(args) == 0 {
		project, err := api.GetProject(strconv.FormatInt(projectID, 10))
		if err != nil {
			return err
		}
		if outputJSON {
			return printJSON(project)
		}
		printProject(project)
		return nil
	}
	switch args[0] {
	case "update":
		fs := flag.NewFlagSet("project update", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		title := fs.String("title", "", "project title")
		description := fs.String("description", "", "project description")
		acceptanceCriteria := fs.String("ac", "", "project acceptance criteria")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		current, err := api.GetProject(strconv.FormatInt(projectID, 10))
		if err != nil {
			return err
		}
		nextDescription := current.Description
		nextAC := current.AcceptanceCriteria
		if fs.Lookup("description") != nil && strings.TrimSpace(*description) != "" || containsFlag(args[1:], "-description") {
			nextDescription = *description
		}
		if containsFlag(args[1:], "-ac") {
			nextAC = *acceptanceCriteria
		}
		project, err := api.UpdateProject(projectID, client.ProjectUpdateRequest{
			Title:              *title,
			Description:        nextDescription,
			AcceptanceCriteria: nextAC,
		})
		if err != nil {
			return err
		}
		if outputJSON {
			return printJSON(project)
		}
		printProject(project)
		return nil
	case "enable":
		project, err := api.SetProjectEnabled(projectID, true)
		if err != nil {
			return err
		}
		if outputJSON {
			return printJSON(project)
		}
		printProject(project)
		return nil
	case "disable":
		project, err := api.SetProjectEnabled(projectID, false)
		if err != nil {
			return err
		}
		if outputJSON {
			return printJSON(project)
		}
		printProject(project)
		return nil
	default:
		return fmt.Errorf("unknown project command %q", args[0])
	}
}

func containsFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

func runList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	taskType := fs.String("type", "", "filter by task type")
	status := fs.String("status", "", "filter by task status")
	assignee := fs.String("user", "", "filter by assignee")
	fs.StringVar(assignee, "u", "", "filter by assignee")
	limit := fs.Int("n", 0, "maximum number of tasks to return; 0 means all")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *limit < 0 {
		return errors.New("usage: task list|ls [--type <type>] [--status <status>] [-u <user>] [-n <limit>]")
	}
	_, api, project, err := resolveCurrentProjectClient()
	if err != nil {
		return err
	}
	tasks, err := api.ListTasksFiltered(project.ID, *taskType, *status, "", *assignee, *limit)
	if err != nil {
		return err
	}
	if outputJSON {
		return printJSON(tasks)
	}
	printTaskTable(tasks)
	return nil
}

func runOrphans(args []string) error {
	if len(args) != 0 {
		return errors.New("usage: task orphans")
	}
	_, api, project, err := resolveCurrentProjectClient()
	if err != nil {
		return err
	}
	tasks, err := api.ListTasks(project.ID)
	if err != nil {
		return err
	}
	var orphans []store.Task
	for _, task := range tasks {
		if task.ParentID == nil {
			orphans = append(orphans, task)
		}
	}
	if outputJSON {
		return printJSON(orphans)
	}
	for _, task := range orphans {
		fmt.Printf("%d\t%s\t%s\t%s\n", task.ID, task.Type, task.Status, task.Title)
	}
	return nil
}

func runGet(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: task get <id>")
	}
	var id int64
	if _, err := fmt.Sscan(args[0], &id); err != nil {
		return errors.New("task id must be numeric")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	api := client.New(cfg)
	task, err := api.GetTask(id)
	if err != nil {
		return err
	}
	dependencies, _ := api.ListDependencies(id)
	if outputJSON {
		return printJSON(task)
	}
	printTaskDetails(task, dependencies)
	return nil
}

func runSearch(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: task search \"query\"")
	}
	_, api, project, err := resolveCurrentProjectClient()
	if err != nil {
		return err
	}
	tasks, err := api.ListTasksFiltered(project.ID, "", "", args[0], "", 0)
	if err != nil {
		return err
	}
	if outputJSON {
		return printJSON(tasks)
	}
	for _, task := range tasks {
		fmt.Printf("%d\t%s\t%s\t%s\n", task.ID, task.Type, task.Status, task.Title)
	}
	return nil
}

func runSetStatus(args []string) error {
	if len(args) != 2 {
		return errors.New("usage: task set-status <id> <status>")
	}
	return updateTaskStatus(args[0], args[1])
}

func runTaskStatusAlias(args []string, status, command string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: task %s <id>", command)
	}
	return updateTaskStatus(args[0], status)
}

func runUpdate(args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	status := fs.String("status", "", "task status")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 || strings.TrimSpace(*status) == "" {
		return errors.New("usage: task update <id> -status <status>")
	}
	return updateTaskStatus(fs.Arg(0), *status)
}

func updateTaskStatus(idArg, status string) error {
	var id int64
	if _, err := fmt.Sscan(idArg, &id); err != nil {
		return errors.New("task id must be numeric")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	api := client.New(cfg)
	current, err := api.GetTask(id)
	if err != nil {
		return err
	}
	updated, err := api.UpdateTask(id, client.TaskUpdateRequest{
		Title:       current.Title,
		Description: current.Description,
		ParentID:    current.ParentID,
		Assignee:    current.Assignee,
		Status:      status,
	})
	if err != nil {
		return err
	}
	if outputJSON {
		return printJSON(updated)
	}
	printTask(updated)
	return nil
}

func runAssign(args []string) error {
	if len(args) != 2 {
		return errors.New("usage: task assign <id> <name>")
	}
	return assignTask(args[0], args[1], true)
}

func runUnassign(args []string) error {
	if len(args) != 2 {
		return errors.New("usage: task unassign <id> <name>")
	}
	return unassignTask(args[0], args[1], true)
}

func runClaim(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: task claim <id>")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.Username) == "" {
		return errors.New("no current username; log in first")
	}
	return assignTask(args[0], cfg.Username, false)
}

func runUnclaim(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: task unclaim <id>")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.Username) == "" {
		return errors.New("no current username; log in first")
	}
	return unassignTask(args[0], cfg.Username, false)
}

func assignTask(idArg, assignee string, requireAdmin bool) error {
	var id int64
	if _, err := fmt.Sscan(idArg, &id); err != nil {
		return errors.New("task id must be numeric")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	api := client.New(cfg)
	status, err := api.Status()
	if err != nil {
		return err
	}
	if requireAdmin && (status.User == nil || status.User.Role != "admin") {
		return errors.New("user is not an admin")
	}
	current, err := api.GetTask(id)
	if err != nil {
		return err
	}
	updated, err := api.UpdateTask(id, client.TaskUpdateRequest{
		Title:       current.Title,
		Description: current.Description,
		ParentID:    current.ParentID,
		Assignee:    assignee,
		Status:      current.Status,
	})
	if err != nil {
		return err
	}
	if outputJSON {
		return printJSON(updated)
	}
	if strings.TrimSpace(updated.Assignee) == "" {
		fmt.Printf("unassigned %d\n", updated.ID)
		return nil
	}
	fmt.Printf("assigned %d to %s\n", updated.ID, updated.Assignee)
	return nil
}

func unassignTask(idArg, expectedAssignee string, requireAdmin bool) error {
	var id int64
	if _, err := fmt.Sscan(idArg, &id); err != nil {
		return errors.New("task id must be numeric")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	api := client.New(cfg)
	status, err := api.Status()
	if err != nil {
		return err
	}
	if requireAdmin && (status.User == nil || status.User.Role != "admin") {
		return errors.New("user is not an admin")
	}
	if requireAdmin {
		users, err := api.ListUsers()
		if err != nil {
			return err
		}
		var found bool
		for _, user := range users {
			if user.Username == expectedAssignee {
				found = true
				if !user.Enabled {
					return errors.New("user is disabled")
				}
				break
			}
		}
		if !found {
			return errors.New("user not found")
		}
	}
	current, err := api.GetTask(id)
	if err != nil {
		return err
	}
	if strings.TrimSpace(current.Assignee) != strings.TrimSpace(expectedAssignee) {
		return fmt.Errorf("task is not assigned to %s", expectedAssignee)
	}
	updated, err := api.UpdateTask(id, client.TaskUpdateRequest{
		Title:       current.Title,
		Description: current.Description,
		ParentID:    current.ParentID,
		Assignee:    "",
		Status:      current.Status,
	})
	if err != nil {
		return err
	}
	if outputJSON {
		return printJSON(updated)
	}
	fmt.Printf("unassigned %d from %s\n", updated.ID, expectedAssignee)
	return nil
}

func runHistory(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: task history <id>")
	}
	var id int64
	if _, err := fmt.Sscan(args[0], &id); err != nil {
		return errors.New("task id must be numeric")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	events, err := client.New(cfg).ListHistory(id)
	if err != nil {
		return err
	}
	if outputJSON {
		return printJSON(events)
	}
	if len(events) == 0 {
		fmt.Println("no history")
		return nil
	}
	for _, event := range events {
		fmt.Printf("ID         : %d\n", event.ID)
		fmt.Printf("TaskID     : %d\n", event.TaskID)
		fmt.Printf("Event      : %s\n", event.EventType)
		fmt.Printf("Created    : %s\n", event.CreatedAt)
		fmt.Printf("Created By : %d\n", event.CreatedBy)
		fmt.Printf("Payload    : %s\n\n", event.Payload)
	}
	return nil
}

func runDependencyCommand(args []string, add bool) error {
	command := "add-dependency"
	if !add {
		command = "remove-dependency"
	}
	if len(args) != 2 {
		return fmt.Errorf("usage: task %s <id> <dependency-id[,dependency-id...]>", command)
	}
	var taskID int64
	if _, err := fmt.Sscan(args[0], &taskID); err != nil {
		return errors.New("task id must be numeric")
	}
	dependencyIDs, err := parseIDList(args[1])
	if err != nil {
		return err
	}
	_, api, project, err := resolveCurrentProjectClient()
	if err != nil {
		return err
	}
	for _, depID := range dependencyIDs {
		req := client.DependencyRequest{
			ProjectID: project.ID,
			TaskID:    taskID,
			DependsOn: depID,
		}
		if add {
			if _, err := api.AddDependency(req); err != nil {
				return err
			}
			continue
		}
		if err := api.RemoveDependency(req); err != nil {
			return err
		}
	}
	if outputJSON {
		return printJSON(map[string]any{
			"task_id":      taskID,
			"dependencies": dependencyIDs,
			"action":       map[bool]string{true: "added", false: "removed"}[add],
		})
	}
	action := "added"
	if !add {
		action = "removed"
	}
	fmt.Printf("%s dependencies for %d: %s\n", action, taskID, args[1])
	return nil
}

func runDependency(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: task dependency <add|remove> <id> <dependency-id[,dependency-id...]>")
	}
	switch args[0] {
	case "add":
		return runDependencyCommand(args[1:], true)
	case "remove":
		return runDependencyCommand(args[1:], false)
	default:
		return fmt.Errorf("unknown dependency action %q", args[0])
	}
}

func runRequest(args []string) error {
	if len(args) > 1 {
		return errors.New("usage: task request [<id>]")
	}
	_, api, project, err := resolveCurrentProjectClient()
	if err != nil {
		return err
	}
	req := client.TaskRequest{ProjectID: project.ID}
	if len(args) == 1 {
		var id int64
		if _, err := fmt.Sscan(args[0], &id); err != nil {
			return errors.New("task id must be numeric")
		}
		req.TaskID = &id
	}
	response, err := api.RequestTask(req)
	if err != nil {
		return err
	}
	if outputJSON {
		return printJSON(response)
	}
	if response.Task != nil {
		printTask(*response.Task)
		return nil
	}
	fmt.Println(response.Status)
	return nil
}

func parseIDList(raw string) ([]int64, error) {
	parts := strings.Split(raw, ",")
	if len(parts) == 0 {
		return nil, errors.New("at least one dependency id is required")
	}
	var ids []int64
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, errors.New("dependency ids must be numeric")
		}
		var id int64
		if _, err := fmt.Sscan(part, &id); err != nil {
			return nil, errors.New("dependency ids must be numeric")
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func runComment(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: task comment add <id> \"comment\"")
	}
	switch args[0] {
	case "add":
		if len(args) != 3 {
			return errors.New("usage: task comment add <id> \"comment\"")
		}
		var id int64
		if _, err := fmt.Sscan(args[1], &id); err != nil {
			return errors.New("task id must be numeric")
		}
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		comment, err := client.New(cfg).AddComment(id, args[2])
		if err != nil {
			return err
		}
		if outputJSON {
			return printJSON(comment)
		}
		fmt.Printf("commented on %d: %s\n", comment.ItemID, comment.Comment)
		return nil
	default:
		return fmt.Errorf("unknown comment command %q", args[0])
	}
}

func runClone(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: task clone|cp <id>")
	}
	var id int64
	if _, err := fmt.Sscan(args[0], &id); err != nil {
		return errors.New("task id must be numeric")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	task, err := client.New(cfg).CloneTask(id)
	if err != nil {
		return err
	}
	if outputJSON {
		return printJSON(task)
	}
	printTask(task)
	return nil
}

func runCurate(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: task curate <id> [id...]")
	}
	_, api, project, err := resolveCurrentProjectClient()
	if err != nil {
		return err
	}
	var sourceIDs []int64
	var titles []string
	for _, arg := range args {
		var id int64
		if _, err := fmt.Sscan(arg, &id); err != nil {
			return fmt.Errorf("invalid task id %q", arg)
		}
		task, err := api.GetTask(id)
		if err != nil {
			return err
		}
		sourceIDs = append(sourceIDs, id)
		titles = append(titles, task.Title)
	}
	title := "Curated requirement"
	if len(titles) > 0 {
		title = titles[0]
	}
	requirement, err := api.CreateTask(client.TaskCreateRequest{
		ProjectID:   project.ID,
		Type:        "requirement",
		Title:       title,
		Description: "Curated from source items.",
	})
	if err != nil {
		return err
	}
	printTask(requirement)
	return nil
}

func runReview(args []string) error {
	fs := flag.NewFlagSet("review", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	status := fs.String("status", "proposed", "review status")
	if err := fs.Parse(args); err != nil {
		return err
	}
	_, api, project, err := resolveCurrentProjectClient()
	if err != nil {
		return err
	}
	tasks, err := api.ListTasksFiltered(project.ID, "requirement", *status, "", "", 0)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		fmt.Printf("%d\t%s\t%s\n", task.ID, task.Status, task.Title)
	}
	return nil
}

func runRequirementStatus(status string, args []string) error {
	commandName := map[string]string{"accepted": "accept", "rejected": "reject"}[status]
	if len(args) != 2 || args[0] != "requirement" {
		return fmt.Errorf("usage: task %s requirement <id>", commandName)
	}
	var id int64
	if _, err := fmt.Sscan(args[1], &id); err != nil {
		return errors.New("task id must be numeric")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	api := client.New(cfg)
	current, err := api.GetTask(id)
	if err != nil {
		return err
	}
	updated, err := api.UpdateTask(id, client.TaskUpdateRequest{
		Title:       current.Title,
		Description: current.Description,
		ParentID:    current.ParentID,
		Assignee:    current.Assignee,
		Status:      status,
	})
	if err != nil {
		return err
	}
	printTask(updated)
	return nil
}

func runRevise(args []string) error {
	if len(args) != 2 || args[0] != "requirement" {
		return errors.New("usage: task revise requirement <id>")
	}
	var id int64
	if _, err := fmt.Sscan(args[1], &id); err != nil {
		return errors.New("task id must be numeric")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	api := client.New(cfg)
	current, err := api.GetTask(id)
	if err != nil {
		return err
	}
	updated, err := api.UpdateTask(id, client.TaskUpdateRequest{
		Title:       current.Title + " (revised)",
		Description: current.Description,
		ParentID:    current.ParentID,
		Assignee:    current.Assignee,
		Status:      "proposed",
	})
	if err != nil {
		return err
	}
	printTask(updated)
	return nil
}

func runDecision(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: task decision <add|list>")
	}
	switch args[0] {
	case "add":
		if len(args) != 2 {
			return errors.New("usage: task decision add \"text\"")
		}
		_, api, project, err := resolveCurrentProjectClient()
		if err != nil {
			return err
		}
		task, err := api.CreateTask(client.TaskCreateRequest{
			ProjectID:   project.ID,
			Type:        "decision",
			Title:       args[1],
			Description: args[1],
		})
		if err != nil {
			return err
		}
		printTask(task)
		return nil
	case "list":
		_, api, project, err := resolveCurrentProjectClient()
		if err != nil {
			return err
		}
		tasks, err := api.ListTasksFiltered(project.ID, "decision", "", "", "", 0)
		if err != nil {
			return err
		}
		for _, task := range tasks {
			fmt.Printf("%d\t%s\t%s\n", task.ID, task.Status, task.Title)
		}
		return nil
	default:
		return fmt.Errorf("unknown decision command %q", args[0])
	}
}

func runConversation(args []string) error {
	if len(args) != 2 || args[0] != "show" {
		return errors.New("usage: task conversation show <id>")
	}
	return runHistory(args[1:])
}

func runTypedTaskCreate(taskType string, args []string) error {
	fs := flag.NewFlagSet(taskType, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	description := fs.String("description", "", "task description")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: task %s [-description text] \"Title\"", taskType)
	}
	return createTask(taskCreateOptions{TaskType: taskType, Title: fs.Arg(0), Description: *description})
}

type taskCreateOptions struct {
	TaskType           string
	Title              string
	Description        string
	AcceptanceCriteria string
	Priority           int
	Assignee           string
	ParentID           *int64
	Project            string
}

func runTaskCreate(args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	taskType := fs.String("type", "task", "task type")
	fs.StringVar(taskType, "t", "task", "task type")
	titleFlag := fs.String("title", "", "task title")
	priority := fs.Int("priority", 1, "task priority")
	fs.IntVar(priority, "p", 1, "task priority")
	assignee := fs.String("assignee", "", "task assignee")
	fs.StringVar(assignee, "a", "", "task assignee")
	description := fs.String("description", "", "task description")
	fs.StringVar(description, "d", "", "task description")
	acceptanceCriteria := fs.String("ac", "", "acceptance criteria")
	parent := fs.Int64("parent", 0, "parent task id")
	project := fs.String("project", "", "project id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	title := strings.TrimSpace(*titleFlag)
	if title == "" {
		title = strings.Join(fs.Args(), " ")
	}
	if title == "" {
		return errors.New("usage: task add|create|new [-title title] [-t type] [-p priority] [-a assignee] [-d description] [-ac criteria] [-parent id] [-project project] [title words]")
	}
	opts := taskCreateOptions{
		TaskType:           *taskType,
		Title:              title,
		Description:        *description,
		AcceptanceCriteria: *acceptanceCriteria,
		Priority:           *priority,
		Assignee:           *assignee,
		Project:            *project,
	}
	if *parent != 0 {
		opts.ParentID = parent
	}
	return createTask(opts)
}

func createTask(opts taskCreateOptions) error {
	cfg, api, project, err := resolveCurrentProjectClient()
	if err != nil {
		return err
	}
	if strings.TrimSpace(opts.Project) != "" {
		project, err = api.GetProject(opts.Project)
		if err != nil {
			return err
		}
	}
	task, err := api.CreateTask(client.TaskCreateRequest{
		ProjectID:          project.ID,
		ParentID:           opts.ParentID,
		Type:               opts.TaskType,
		Title:              opts.Title,
		Description:        opts.Description,
		AcceptanceCriteria: opts.AcceptanceCriteria,
		Priority:           opts.Priority,
		Assignee:           opts.Assignee,
	})
	if err != nil {
		return err
	}
	if outputJSON {
		return printJSON(task)
	}
	if task.Type == "epic" {
		cfg.CurrentEpicID = task.ID
		if err := config.Save(cfg); err != nil {
			return err
		}
	}
	fmt.Println(task.ID)
	return nil
}

func runConfig(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: task config <set|get> <key> [value]")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	switch args[0] {
	case "set":
		if len(args) != 3 {
			return errors.New("usage: task config set <key> <value>")
		}
		switch args[1] {
		case "server":
			cfg.ServerURL = args[2]
		default:
			return fmt.Errorf("unknown config key %q", args[1])
		}
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Printf("%s=%s\n", args[1], args[2])
		return nil
	case "get":
		switch args[1] {
		case "server":
			fmt.Println(config.ResolveServerURL(cfg))
			return nil
		default:
			return fmt.Errorf("unknown config key %q", args[1])
		}
	default:
		return fmt.Errorf("unknown config action %q", args[0])
	}
}

func printProject(project store.Project) {
	if outputJSON {
		_ = printJSON(project)
		return
	}
	fmt.Printf("project: %s\n", project.Title)
	fmt.Printf("project_id: %d\n", project.ID)
	fmt.Printf("status: %s\n", project.Status)
	if project.Description != "" {
		fmt.Printf("description: %s\n", project.Description)
	}
	if project.AcceptanceCriteria != "" {
		fmt.Printf("acceptance_criteria: %s\n", project.AcceptanceCriteria)
	}
}

func printTask(task store.Task) {
	if outputJSON {
		_ = printJSON(task)
		return
	}
	fmt.Printf("task: %s\n", task.Title)
	fmt.Printf("id: %d\n", task.ID)
	fmt.Printf("type: %s\n", task.Type)
	fmt.Printf("status: %s\n", task.Status)
	fmt.Printf("project_id: %d\n", task.ProjectID)
	if task.ParentID != nil {
		fmt.Printf("parent_id: %d\n", *task.ParentID)
	}
	if task.CloneOf != nil {
		fmt.Printf("clone_of: %d\n", *task.CloneOf)
	}
	if task.Description != "" {
		fmt.Printf("description: %s\n", task.Description)
	}
}

func printTaskDetails(task store.Task, dependencies []store.Dependency) {
	parentID := ""
	if task.ParentID != nil {
		parentID = fmt.Sprintf("%d", *task.ParentID)
	}
	dependsOn := formatDependsOn(dependencies)
	fmt.Printf("ID           : %d\n", task.ID)
	fmt.Printf("Type         : %s\n", task.Type)
	fmt.Printf("Description  : %s\n", task.Description)
	fmt.Printf("ParentID     : %s\n", parentID)
	if task.CloneOf != nil {
		fmt.Printf("CloneOf      : %d\n", *task.CloneOf)
	}
	fmt.Printf("ProjectID    : %d\n", task.ProjectID)
	fmt.Printf("Title        : %s\n", task.Title)
	fmt.Printf("Assignee     : %s\n", task.Assignee)
	fmt.Printf("Order        : \n")
	fmt.Printf("DependsOn    : %s\n", dependsOn)
	fmt.Printf("Status       : %s\n", task.Status)
	fmt.Printf("Priority     : %d\n", task.Priority)
	fmt.Printf("Created      : %s\n", task.CreatedAt)
	fmt.Printf("LastModified : %s\n", task.UpdatedAt)
	fmt.Printf("Acceptance Criteria : %s\n", task.AcceptanceCriteria)
}

func formatDependsOn(dependencies []store.Dependency) string {
	var ids []string
	for _, dependency := range dependencies {
		ids = append(ids, strconv.FormatInt(dependency.DependsOn, 10))
	}
	if len(ids) == 0 {
		return "[]"
	}
	return "[" + strings.Join(ids, ",") + "]"
}

func heading(label string) {
	if noColorOutput {
		fmt.Printf("%s\n", label)
		return
	}
	fmt.Printf("\x1b[2;36m%s\x1b[0m\n", label)
}

func resolveCurrentProjectClient() (config.Config, *client.Client, store.Project, error) {
	cfg, err := config.Load()
	if err != nil {
		return config.Config{}, nil, store.Project{}, err
	}
	if cfg.CurrentProject == "" {
		return config.Config{}, nil, store.Project{}, errors.New("no active project; use `task project create` or `task project use <id>` first")
	}
	api := client.New(cfg)
	project, err := api.GetProject(cfg.CurrentProject)
	if err != nil {
		return config.Config{}, nil, store.Project{}, err
	}
	return cfg, api, project, nil
}

func resolveCredentials(usernameFlag, passwordFlag string, useEnv bool) (string, string, error) {
	username := strings.TrimSpace(usernameFlag)
	password := strings.TrimSpace(passwordFlag)

	if useEnv {
		if username == "" {
			username = strings.TrimSpace(os.Getenv("TASK_USERNAME"))
		}
		if password == "" {
			password = strings.TrimSpace(os.Getenv("TASK_PASSWORD"))
		}
	}
	if username == "" {
		username = currentOSUser()
	}
	if password == "" {
		password = "password"
	}
	if username == "" || password == "" {
		return "", "", errors.New("username and password are required")
	}
	return username, password, nil
}

func currentOSUser() string {
	user, err := osuser.Current()
	if err == nil && user.Username != "" {
		parts := strings.Split(user.Username, `\`)
		return parts[len(parts)-1]
	}
	if env := os.Getenv("USER"); env != "" {
		return env
	}
	if env := os.Getenv("USERNAME"); env != "" {
		return env
	}
	return "user"
}

func extractURLOverride(args []string) ([]string, string, error) {
	if len(args) == 0 {
		return args, "", nil
	}
	var out []string
	var override string
	for i := 0; i < len(args); i++ {
		if args[i] == "-url" {
			if i+1 >= len(args) {
				return nil, "", errors.New("missing value for -url")
			}
			override = args[i+1]
			i++
			continue
		}
		out = append(out, args[i])
	}
	return out, override, nil
}

func extractOutputFlags(args []string) ([]string, bool, bool, error) {
	var out []string
	var jsonFlag bool
	var nocolor bool
	for _, arg := range args {
		switch arg {
		case "-json":
			jsonFlag = true
		case "-nocolor":
			nocolor = true
		default:
			out = append(out, arg)
		}
	}
	return out, jsonFlag, nocolor, nil
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func generatePassword(length int) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	if length <= 0 {
		return "", errors.New("password length must be positive")
	}
	buf := make([]byte, length)
	random := make([]byte, length)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	for i, b := range random {
		buf[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(buf), nil
}

func removeDBFiles(path string) error {
	for _, suffix := range []string{"", "-shm", "-wal"} {
		candidate := path + suffix
		if err := os.Remove(candidate); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func defaultDatabasePath() (string, error) {
	home, err := config.Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "task.db"), nil
}

func promptForCredentials(in io.Reader, out io.Writer, defaultUsername, defaultPassword string) (string, string, error) {
	reader := bufio.NewReader(in)
	if defaultUsername != "" {
		fmt.Fprintf(out, "username [%s]: ", defaultUsername)
	} else {
		fmt.Fprint(out, "username: ")
	}
	username, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}
	username = strings.TrimSpace(username)
	if username == "" {
		username = defaultUsername
	}
	if defaultPassword != "" {
		fmt.Fprint(out, "password [press enter to use default]: ")
	} else {
		fmt.Fprint(out, "password: ")
	}
	password, err := readPasswordPrompt(reader, in, out)
	if err != nil {
		return "", "", err
	}
	if password == "" {
		password = defaultPassword
	}
	return username, password, nil
}

func readPasswordPrompt(reader *bufio.Reader, in io.Reader, out io.Writer) (string, error) {
	inFile, inOK := in.(*os.File)
	outFile, outOK := out.(*os.File)
	if !inOK || !outOK || !term.IsTerminal(int(inFile.Fd())) || !term.IsTerminal(int(outFile.Fd())) {
		password, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(password), nil
	}

	oldState, err := term.MakeRaw(int(inFile.Fd()))
	if err != nil {
		return "", err
	}
	defer func() {
		_ = term.Restore(int(inFile.Fd()), oldState)
	}()

	var buf []byte
	single := make([]byte, 1)
	for {
		if _, err := inFile.Read(single); err != nil {
			return "", err
		}
		switch single[0] {
		case '\r', '\n':
			fmt.Fprint(out, "\n")
			return string(buf), nil
		case 3:
			fmt.Fprint(out, "^C\n")
			return "", errors.New("interrupt")
		case 8, 127:
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				fmt.Fprint(out, "\b \b")
			}
		default:
			if single[0] >= 32 && single[0] <= 126 {
				buf = append(buf, single[0])
				fmt.Fprint(out, "*")
			}
		}
	}
}

func resolveLoginUsername(configUsername, usernameFlag string) string {
	if strings.TrimSpace(configUsername) != "" {
		return strings.TrimSpace(configUsername)
	}
	if strings.TrimSpace(usernameFlag) != "" {
		return strings.TrimSpace(usernameFlag)
	}
	return strings.TrimSpace(os.Getenv("TASK_USERNAME"))
}

func resolveLoginPassword(passwordFlag string) string {
	if strings.TrimSpace(passwordFlag) != "" {
		return strings.TrimSpace(passwordFlag)
	}
	return strings.TrimSpace(os.Getenv("TASK_PASSWORD"))
}

func renderRootUsage() string {
	return renderBanner() + strings.TrimSpace(`
USAGE
  task <command> [options]

CLIENT COMMANDS
  add         Create a task in the active project
  claim       Assign yourself to a task
  clone       Clone a task or epic
  comment     Add comments to a task
  complete    Set a task status to complete
  count       Count users, projects, and work by type
  dependency  Manage dependency links between tasks
  fail        Set a task status to fail
  get         Show a task with history and comments
  help        Show command help
  inprogress  Set a task status to inprogress
  list        List tasks in the active project
  login       Log into the server
  logout      Clear the local session
  onboard     Append the embedded AGENTS.md template in the current directory
  open        Set a task status to open
  orphans     List tasks with no parent
  project     Manage projects and active project context
  ready       Alias for open
  req         Generate requirements via an external agent
  register    Create a user account on the server
  request     Request work for the current user
  search      Search tasks in the active project
  status      Show server and authentication status
  unclaim     Remove yourself from a task
  update      Update a task
  version     Print the current version from VERSION

ADMIN COMMANDS
  assign      Admin-only task assignment
  initdb      Initialize the database, bootstrap admin, and create the default project
  server      Start the API server and embedded web UI
  unassign    Admin-only task unassignment
  user        Admin-only user management

HELP
  task help <command>
`) + "\n"
}

func printCountSummary(summary store.CountSummary, scopedToProject bool) {
	fmt.Printf("users %d\n", summary.Users)
	if !scopedToProject {
		fmt.Printf("projects %d\n", summary.Projects)
	}
	for _, item := range summary.Types {
		fmt.Printf("%ss %d", item.Type, item.Total)
		if suffix := formatStatusCounts(item.Statuses); suffix != "" {
			fmt.Printf(" (%s)", suffix)
		}
		fmt.Println()
	}
}

func printTaskTable(tasks []store.Task) {
	if len(tasks) == 0 {
		fmt.Println("no tasks")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tSTATUS\tASSIGNEE\tPRIORITY\tTITLE")
	for _, task := range tasks {
		assignee := task.Assignee
		if strings.TrimSpace(assignee) == "" {
			assignee = "-"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\t%s\n", task.ID, task.Type, task.Status, assignee, task.Priority, task.Title)
	}
	_ = w.Flush()
}

func formatStatusCounts(statuses map[string]int) string {
	order := []string{"open", "inprogress", "notready", "complete", "fail"}
	labels := map[string]string{
		"open":       "open",
		"inprogress": "in progress",
		"notready":   "not ready",
		"complete":   "complete",
		"fail":       "fail",
	}
	var parts []string
	for _, status := range order {
		if count := statuses[status]; count > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", count, labels[status]))
		}
	}
	return strings.Join(parts, ", ")
}

func renderBanner() string {
	var b strings.Builder
	for i, line := range bannerLines {
		color := bannerColors[i%len(bannerColors)]
		b.WriteString(color)
		b.WriteString(line)
		b.WriteString("\x1b[0m\n")
	}
	b.WriteString("\n")
	return b.String()
}

func renderCommandHelp(command string) string {
	command = normalizeHelpCommand(command)
	info, ok := helpIndex[command]
	if !ok {
		return renderRootUsage()
	}
	var b strings.Builder
	b.WriteString("USAGE\n  ")
	b.WriteString(info.usage)
	b.WriteString("\n\n")
	if len(info.details) > 0 {
		b.WriteString("DETAILS\n")
		for _, line := range info.details {
			b.WriteString("  ")
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("EXAMPLE\n  ")
	b.WriteString(info.example)
	b.WriteString("\n")
	return b.String()
}

func hasCommandHelp(command string) bool {
	_, ok := helpIndex[normalizeHelpCommand(command)]
	return ok
}

func normalizeHelpCommand(command string) string {
	switch command {
	case "show":
		return "get"
	case "create", "new":
		return "add"
	case "ls":
		return "list"
	case "cp":
		return "clone"
	default:
		return command
	}
}
