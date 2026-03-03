# Design

## Product Summary

`task` is a lightweight ticket and project management system delivered as a single Go binary.

It is designed for small teams that want low-friction task tracking without separate infrastructure for the API, database, and web UI. The product combines a server, a terminal-first CLI, and an embedded web application around one shared data model.

The system has three interfaces:

1. A server that owns persistence, authentication, and collaboration.
2. A CLI for fast, explicit terminal workflows.
3. An embedded web application for browsing, editing, and status management.

The repository also contains a static `VERSION` file. `make build` increments the patch version before compiling the binary and copies that value into the embedded build asset used by `task version`.

Client-side files are stored under `$TASK_HOME`, which defaults to `~/.config/task`.

## Product Principles

1. The server defines the single system of record and the shared data model used by both remote and local workflows.
2. The CLI and web app use the same API semantics and data model.
3. Common operations should be fast and predictable from the terminal.
4. Projects should support lightweight hierarchy through epics and child tasks.
5. Every meaningful change should be traceable through history and comments.

## Primary Users And Workflows

The primary user is a small software team managing projects, epics, tasks, bugs.

The first release must support these workflows end to end:

1. Initialize a local SQLite-backed workspace.
2. Store passwords as Argon2id hashes in SQLite.
3. Start the server and embedded web app from the same binary.
4. Create and manage users.
5. Authenticate from the CLI and the web app.
6. Create and select projects.
7. Add work items such as tasks, bugs, and epics.
8. List, filter, search, and inspect items.
9. Optionally organize work beneath a parent task or epic.
10. Review item history and comments.
11. Manage work visually in the web app, including status-based board views.

## Domain Model

### User

- `user_id`
- `username`
- `password_hash`
- `role`
- `display_name`
- `enabled`
- `created_at`

Roles in the first release:

- `admin`
- `user`

Notes:

- administrators can create, enable, and disable users
- regular users can log in and manage project work according to API permissions

### Project

- `project_id`
- `title`
- `description`
- `created_at`
- `created_by`
- `status`

Projects are the top-level container for work items.

### Task

`Task` is the main work artifact. All item types share one core model.

- `task_id`
- `project_id`
- `parent_id`
- `type`
- `title`
- `description`
- `acceptance_criteria`
- `status`
- `priority`
- `estimate_effort`
- `estimate_complete`
- `assignee`
- `comments`
- `created_at`
- `created_by`
- `updated_at`
- `archived`

Supported `type` values in the first release:

- `epic`
- `task`
- `bug`

Model notes:

- `parent_id` is nullable and supports hierarchical work
- tasks are orphaned when `parent_id` is null
- task creation accepts either a positional title or `-title`
- `acceptance_criteria` is captured directly on the task record
- `estimate_effort` is an integer assessment of task effort
- `estimate_complete` is the estimated delivery datetime and should use RFC3339 format
- `comments` are exposed on task detail reads as an array of `{author, date, text}` ordered most recent first

CLI creation defaults:

- `task add`, `task create`, and `task new` are the same command
- `task list` and `task ls` are the same command
- if `-type` / `-t` is omitted, the type defaults to `task`
- if `-priority` / `-p` is omitted, the priority defaults to `1`
- if `-assignee` / `-a` is omitted, the assignee is blank
- if `-description` / `-d` is omitted, the description is blank
- if `-ac` is omitted, the acceptance criteria is blank
- if `-estimate_effort` is omitted, it defaults to `0`
- if `-estimate_complete` is omitted, it is blank
- if `-parent` is omitted, the task is created without a parent
- if `-project` is omitted, the active project is used

### History

Append-only audit log for important changes.

- `id`
- `project_id`
- `task_id`
- `event_type`
- `payload`
- `created_at`
- `created_by`

Typical history events:

- task created
- task updated
- status changed
- assignee changed
- parent changed
- comment added

## Functional Scope

### Workspace Initialization

The product must support local initialization of a SQLite database from the CLI.

The bootstrap command is `task initdb`.

`task initdb` must:

1. create the schema in a new SQLite database
2. create an `admin` account
3. create a default project

Representative flow:

```bash
task initdb -f task.db --force -password secret
```

Bootstrap defaults:

- admin username is always `admin`
- if `-f` is omitted, the SQLite database is created at `$TASK_HOME/task.db`
- admin password comes from `-password` when supplied
- if `-password` is omitted, the CLI generates a random password and prints it to stdout
- if `--force` is supplied, any existing SQLite database file is overwritten
- the default project is created automatically during initialization

### Server

The server is the system of record.

Responsibilities:

- manage SQLite persistence
- expose the HTTP API for CLI and web use
- enforce authentication and authorization
- serve the embedded web application
- support multi-user access
- provide near-real-time refresh for connected clients

The default local server should listen on `http://localhost:8080`.

If `task server` is run without `-f`, it must open the SQLite database at `$TASK_HOME/task.db`.

If `task server` is run with `-v`, it must print verbose request and response details to stdout.

### Authentication And User Management

The first release must support:

1. administrator bootstrap during initialization
2. user creation by administrators
3. user listing by administrators
4. user deletion by administrators
5. enable and disable user accounts
6. login and logout from CLI and web
7. user/session status inspection from the CLI

Representative commands:

```bash
task onboard
task version
task user create --username alice --password secret
task user ls
task user delete --username alice
task user enable --username alice
task user disable --username alice

task register
task login
task status
task logout
```

`task onboard` must append the embedded `cmd/task/AGENTS.md` template into `${CWD}/AGENTS.md`, creating that file if it does not exist.

`task status` must always print the current effective configuration first, then perform a mode-appropriate connectivity check.

In REMOTE mode it must print at least:

- `mode: remote`
- `server: <TASK_SERVER>`
- `username: <configured username or blank>`
- `authenticated: true|false`

The REMOTE connectivity check is:

- call the remote status endpoint

The REMOTE result must then print:

- `connection: success` in green if the server responds successfully
- `connection: failure` in red if the server cannot be contacted or returns an error

In LOCAL mode it must print at least:

- `mode: local`
- `db_path: <resolved database path>`
- `db_exists: true|false`

The LOCAL connectivity check is:

- if the database file exists, open it and verify the schema is usable

A usable schema means:

- the required application tables exist and can be queried

The LOCAL result must then print:

- `connection: success` in green if the database can be opened and the schema is valid
- `connection: failure` in red if the database is missing, cannot be opened, or the schema is invalid

If the database does not exist in LOCAL mode, `task status` must also print:

- `hint: run task initdb`

If `-nocolor` is set, the same output must be printed without ANSI colors.

`task count` must query the server and print aggregate counts for users and work item types. Without a project filter it must also print the project count. With `-project_id <id>` it must scope work item counts to that project.

The CLI must resolve credentials from `-username` and `-password` first, then `TASK_USERNAME` and `TASK_PASSWORD`, and finally default to OS `whoami` and `password`.

The CLI must resolve the server URL from `-url` first, then `TASK_SERVER`, then saved config, and finally default to `http://localhost:8080`.

The CLI must expose `task version`, which prints the semantic version embedded into the binary at build time.

`task initdb` is separate from the login and registration flows: it only creates `admin`, does not consume `TASK_USERNAME`, and does not read `TASK_PASSWORD`.

Admin-only user-management requests must be rejected by the server when the caller is authenticated but not an admin. Those requests must return HTTP 403 with an error explaining that the user is not an admin.

When `task` is run without arguments, the CLI should print a colored ASCII-art `TASK` banner above the main usage text.

When `task server` starts, it should print the same colored ASCII-art `TASK` banner before the startup message.

Below that banner, `task server` must print the embedded version and the resolved task database path.

The CLI stores non-sensitive client defaults in `$TASK_HOME/config.json` and session credentials in `$TASK_HOME/credentials.json`.

`task login` must:

1. check `$TASK_HOME/credentials.json` first and reuse that session if it is still valid
2. check the `username` in `$TASK_HOME/config.json`
3. check `-username` and `-password`, then `TASK_USERNAME` and `TASK_PASSWORD`
4. prompt for any missing values
5. when prompting, use the discovered values as editable defaults
6. print `invalid credentials` on an invalid-login response before prompting for a retry
7. when prompting for a password in an interactive terminal, echo `*` characters instead of the raw password
8. on success, write the session token to `$TASK_HOME/credentials.json`
9. on success, update the `username` and `server_url` keys in `$TASK_HOME/config.json`

`task register` must create the account but must not create or persist a logged-in session.

`task logout` must remove `$TASK_HOME/credentials.json`.

### Project Management

Users must be able to:

1. create projects
2. list projects
3. inspect project details
4. select an active project for CLI defaults

Representative commands:

```bash
task project create -description "Portal backlog" -ac "Launch criteria" "Customer Portal"
task project list
task project ls
task project use 2
task project get 2
task project
task project 2 update -title "Customer Portal"
task project 2 update -description "Portal backlog"
task project 2 update -ac "Launch criteria"
task project 2 enable
task project 2 disable
```

`task project list` should show at least the project id, title, and status, and indicate which project is current in the local CLI context.

All `task <command> create` commands must return to STDOUT the newly created ID, if they succeed.

The selected project should be remembered locally by the CLI.

### Work Item Capture

Creating work should be low-friction.

Users must be able to create tasks, bugs, and epics.

Representative commands:

```bash
task add "Customers can reset their password."
task create "Customers can reset their password."
task new "Customers can reset their password."
task bug "Reset token fails after first use."
task epic "Authentication"
task create -t task -p 1 -a alice -d "Add audit event" "Add password reset audit event"
```

Behavior notes:

- `task add`, `task create`, and `task new` are aliases
- `task list` and `task ls` are aliases
- `task list -n <limit>` applies a server-side limit, with `0` meaning no limit
- task creation defaults are `type=task`, `priority=1`, blank assignee, blank description, blank parent, and current project
- `-ac` stores acceptance criteria on the task
- each item records project, creator, timestamps, status, and revision history

### Review And Search

Users must be able to:

1. list all items in the active project
2. filter by type
3. filter by status
4. search across titles and descriptions within the active project by default
5. inspect full item detail
6. list orphaned items with no parent

Representative commands:

```bash
task list
task ls
task list --type bug
task list --status open
task search "password reset"
task search "password reset" -allprojects
task get 42
task orphans
```

`task search` should search the active project by default. If `-allprojects` is supplied, it should search across all projects.

The CLI should support `-json` on client-facing commands and pretty-print the response JSON.

`task get <id>` should print a flat detail view with the fields `ID`, `Type`, `Description`, `ParentID`, `CloneOf` when present, `ProjectID`, `Title`, `Assignee`, `Order`, `EstimateEffort`, `EstimateComplete`, `DependsOn`, `Status`, `Priority`, `Created`, `LastModified`, `Acceptance Criteria`, and a `Comments` section ordered most recent first.

`task list` should render a readable table that includes at least the id, type, status, assignee, priority, and title.

### Workflow And Status Management

The system should support task progression through status changes.

The first release supports this default status set:

- `notready`
- `open`
- `inprogress`
- `complete`
- `fail`

The CLI and web app must both support easy status changes.

Assignment workflows must support:

- `task assign <id> <name>` for admins
- `task unassign <id> <name>` for admins
- `task dependency add <id> <dependency-id[,dependency-id...]>`
- `task dependency remove <id> <dependency-id[,dependency-id...]>`
- `task request [<id>]` for the caller
- `task claim <id>` for the caller
- `task unclaim <id>` for the caller
- `task set-parent <id> <parent-id>`
- `task unset-parent <id>`
- `task list -u <name>` / `task ls -u <name>` for assignee filtering
- `task open <id>`
- `task ready <id>` as an alias for `task open <id>`
- `task inprogress <id>`
- `task complete <id>`
- `task fail <id>`
- `task update <id> -status <status>`
- `task update <id> -title <title>`
- `task update <id> -description <description>`
- `task update <id> -ac <acceptance-criteria>`
- `task update <id> -priority <priority>`
- `task update <id> -order <order>`
- `task update <id> -parent_id <parent-id>`
- `task update <id> -estimate_effort <effort>`
- `task update <id> -estimate_complete <rfc3339-datetime>`

Assignment rules:

- the server must reject admin-only assignment calls made by non-admin users
- `task assign` and `task unassign` must fail if the named target user does not exist
- `task assign` and `task unassign` must fail if the named target user is disabled
- `task request <id>` must return `{"status":"REJECTED"}` when the requested task cannot be assigned
- `task request` must return `{"status":"NO-WORK"}` when no assignable work exists
- successful request responses must return `{"status":"ASSIGNED","task":...}`
- if the caller already has an assigned `inprogress` task, that task is returned
- otherwise, if the caller has assigned `open` work, the oldest assigned `open` task is returned
- otherwise, `task request` assigns the oldest unassigned `open` task in the active project
- `task claim` must fail if the task is already assigned to another user
- `task unclaim` must fail if the caller is not the current assignee
- a non-admin user must not be able to override another user assignment through the generic task update API

### Hierarchy

Projects must support lightweight hierarchy through parent-child relationships.

The first release should support:

1. creating epics
2. attaching tasks and bugs to an epic via `parent_id`
3. tracking the active epic in the CLI for faster entry
4. browsing hierarchy in the web UI

### History And Comments

Users must be able to inspect how an item changed over time.

The first release must include:

1. append-only history events for important changes
2. comments attached to items
3. `task history <id>` in the CLI for event output
4. item detail pages in the web app that surface history and comments

Representative commands:

```bash
task history 17
task comment add 17 "Waiting on API changes."
```

## CLI Design

The CLI is the fastest interface for expert users.

Requirements:

- use the same HTTP API as the web app
- never bypass the server or SQLite
- support explicit and scriptable commands
- maintain local defaults for current project, credentials, and active epic where useful

Representative command set:

```bash
task project create "Customer Portal"
task project use 2

task epic "Authentication"
task add "Customers can reset their password."
task bug "Reset token expires immediately."
task list
task get 42
task search "password reset"
task history 42
```

The CLI should support only the aliases that are part of the documented command surface.

## Web Application

The web application is embedded into the Go binary with `go:embed`.

Requirements:

- single-page application
- operationally lightweight
- collaborative and multi-user aware
- no manual page refresh required for normal use
- project switcher
- status-based board view
- item detail view with history and comments

The web UI should make these activities easy:

- switch between projects
- add and edit items
- view hierarchy
- manage status on a board
- inspect history and comments

## Persistence And Architecture

### Storage

- SQLite is the only database in the first release.
- SQLite remains the persistence layer behind the server data model; local mode uses the same data model and validation rules as the server-backed flow.

Suggested storage areas:

1. users
2. sessions
3. projects
4. tasks
5. history_events
6. comments

### Application Shape

The implementation should be organized around shared domain concepts rather than separate one-off logic in each interface.

Suggested layers:

1. domain models and validation
2. application services for auth, projects, tasks, comments, and history
3. HTTP handlers and API contracts
4. SQLite repositories
5. CLI commands and web UI clients consuming the API

## Non-Goals For The First Release

Avoid overbuilding the initial product.

Non-goals:

- multiple database backends
- direct client access to SQLite
- heavyweight enterprise workflow configuration
- advanced portfolio planning
- deeply nested issue taxonomies beyond simple parent-child hierarchy

## Quality Gates

The repository should provide at least these checks:

```bash
make build
make test
make test-go
make test-playwright
```

`make build` must increment the patch component of the semantic version stored in `VERSION` before running the Go build.

Changes are not complete until the relevant automated checks pass.

## Success Criteria

The product is successful if a user can:

1. initialize a local workspace and start the server
2. create users and authenticate successfully
3. create and switch projects quickly
4. add tasks, bugs, and epics with minimal friction
5. inspect work through list, search, detail, history, and comments
6. manage work visually through the web interface


## Task Status

The status of a task is either
    notready
    open
    inprogress
    complete
    fail

This is set using 
    `task open N`
    `task ready N`
    `task inprogress N`
    `task complete N`
    `task fail N`
or
    `task update N -status <status>`
    `task update N -title <title>`
    `task update N -description <description>`
    `task update N -ac <acceptance-criteria>`
    `task update N -priority <priority>`
    `task update N -order <order>`
    `task update N -parent_id <parent-id>`
    `task update N -estimate_effort <effort>`
    `task update N -estimate_complete <rfc3339-datetime>`

## Requesting Tasks

A user can makes a request to work on a specific task

    `task request N`

It is either assigned the task it requested, or it is rejected. If assigned, the task is updated to have this user name and the response is `{"status":"ASSIGNED","task":...}`. If not, the response is `{"status":"REJECTED"}`.

Or a user may request ANY task

    task request

It is either assigned a task, or no work is available. If assigned, the task is updated to have this user name and the response is `{"status":"ASSIGNED","task":...}`. If not, the response is `{"status":"NO-WORK"}`.

If the user has already been assigned a task, the task that is inprogress is returned. If the user has been assigned a task that is ready, then the oldest task that is assigned is then returned.

    
