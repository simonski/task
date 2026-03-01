OpenAI Codex v0.106.0 (research preview)
--------
workdir: /Users/simon/code/task
model: gpt-5.3-codex
provider: openai
approval: never
sandbox: read-only
reasoning effort: medium
reasoning summaries: none
session id: 019cab8d-1b6b-75b1-b962-9dc5e4bc70dc
--------
user
Write an example breakdown of implementation requirements as test.md in the format:

EPIC: title
ID: E1, E2, E3 etc
DESCRIPTION: description
AC: list of acceptance criteria
PRIORITY: 1-N (1 highest, do this first)
DEPENDS-ON: E2, E4

<indent for stories "in" the epic (the story ID should increment and be EPIC-STORY)>
    STORY: title
    ID: E1-S1, E1-2, E1-S3 etc.
    DESCRIPTION: description
    AC: list of acceptance criteria
    PRIORITY: 1-N (1 highest, do this first)
    DEPENDS-ON: E1-S2

Use the following input files as source material:

FILE: docs/DESIGN.md
-----
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

1. The server is the single system of record.
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
- `assignee`
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

CLI creation defaults:

- `task add`, `task create`, and `task new` are the same command
- `task list` and `task ls` are the same command
- if `-type` / `-t` is omitted, the type defaults to `task`
- if `-priority` / `-p` is omitted, the priority defaults to `1`
- if `-assignee` / `-a` is omitted, the assignee is blank
- if `-description` / `-d` is omitted, the description is blank
- if `-ac` is omitted, the acceptance criteria is blank
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

`task status` must display both the server version and the client version and warn when they differ.

`task count` must query the server and print aggregate counts for users and work item types. Without a project filter it must also print the project count. With `-project_id <id>` it must scope work item counts to that project.

The CLI must resolve credentials from `-username` and `-password` first, then `TASK_USERNAME` and `TASK_PASSWORD`, and finally default to OS `whoami` and `password`.

The CLI must resolve the server URL from `-url` first, then `TASK_URL`, then saved config, and finally default to `http://localhost:8080`.

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
4. search across titles and descriptions
5. inspect full item detail
6. list orphaned items with no parent

Representative commands:

```bash
task list
task ls
task list --type bug
task list --status open
task search "password reset"
task get 42
task orphans
```

The CLI should support `-json` on client-facing commands and pretty-print the response JSON.

`task get <id>` should print a flat detail view with the fields `ID`, `Type`, `Description`, `ParentID`, `ProjectID`, `Title`, `Assignee`, `Order`, `DependsOn`, `Status`, `Priority`, `Created`, `LastModified`, and `Acceptance Criteria`.

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
- `task list -u <name>` / `task ls -u <name>` for assignee filtering
- `task open <id>`
- `task ready <id>` as an alias for `task open <id>`
- `task inprogress <id>`
- `task complete <id>`
- `task fail <id>`
- `task update <id> -status <status>`

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
- Only the server accesses SQLite directly.

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

## Requesting Tasks

A user can makes a request to work on a specific task

    `task request N`

It is either assigned the task it requested, or it is rejected. If assigned, the task is updated to have this user name and the response is `{"status":"ASSIGNED","task":...}`. If not, the response is `{"status":"REJECTED"}`.

Or a user may request ANY task

    task request

It is either assigned a task, or no work is available. If assigned, the task is updated to have this user name and the response is `{"status":"ASSIGNED","task":...}`. If not, the response is `{"status":"NO-WORK"}`.

If the user has already been assigned a task, the task that is inprogress is returned. If the user has been assigned a task that is ready, then the oldest task that is assigned is then returned.

    
-----


mcp startup: no servers

thinking
**Confirming write limitations**
codex
I’m using the design doc as the source of truth and checking the workspace state before drafting the breakdown. After that I’ll either write `test.md` directly or, if the sandbox blocks writes, I’ll give you the exact file contents.
exec
/bin/zsh -lc pwd in /Users/simon/code/taskexec
/bin/zsh -lc "rg --files /Users/simon/code/task | rg '("'^|/)AGENTS'"\\.md"'$|docs/DESIGN'"\\.md"'$'"'" in /Users/simon/code/task succeeded in 51ms:
/Users/simon/code/task
 succeeded in 52ms:
/Users/simon/code/task/docs/DESIGN.md

thinking
**Testing write permission blocking**
codex
I found the source file and I’m drafting the epics from its major implementation areas: bootstrap/auth, projects, work items, workflow, history, and web/UI. I’m also checking whether the workspace allows file writes before I try to create `test.md`.
exec
/bin/zsh -lc 'test -w /Users/simon/code/task && echo writable || echo read-only' in /Users/simon/code/task succeeded in 52ms:
read-only

thinking
**Providing blocked write explanation**
codex
I couldn’t create [test.md](/Users/simon/code/task/test.md) because the workspace is mounted read-only in this session. Below is the exact content I would put in that file.

```md
EPIC: Workspace Bootstrap And Runtime
ID: E1
DESCRIPTION: Initialize the local SQLite workspace, bootstrap the admin account and default project, and start the combined API and embedded web server from the Go binary.
AC:
- `task initdb` creates a new SQLite schema in the requested or default database path.
- `task initdb` creates the `admin` user and a default project.
- `task initdb` respects `--force` overwrite behavior and prints a generated password when `-password` is omitted.
- `task server` uses `$TASK_HOME/task.db` when `-f` is omitted.
- `task server` prints the colored TASK banner, embedded version, and resolved database path on startup.
- `task server -v` prints verbose request and response details.
PRIORITY: 1
DEPENDS-ON:

    STORY: Initialize SQLite workspace from CLI
    ID: E1-S1
    DESCRIPTION: Implement `task initdb` to create schema, bootstrap admin, and create the default project.
    AC:
    - Command accepts `-f`, `--force`, and `-password`.
    - Default database path resolves to `$TASK_HOME/task.db`.
    - Admin username is always `admin`.
    - Existing database is overwritten only when `--force` is supplied.
    PRIORITY: 1
    DEPENDS-ON:

    STORY: Generate and print bootstrap password when omitted
    ID: E1-S2
    DESCRIPTION: Support secure password generation for `task initdb` when no password flag is provided.
    AC:
    - A random password is generated when `-password` is absent.
    - Generated password is printed to stdout exactly once during initialization.
    - `task initdb` does not read `TASK_USERNAME` or `TASK_PASSWORD`.
    PRIORITY: 2
    DEPENDS-ON: E1-S1

    STORY: Start combined server and embedded web app
    ID: E1-S3
    DESCRIPTION: Implement `task server` to expose the HTTP API and embedded SPA from the same binary.
    AC:
    - Default local server listens on `http://localhost:8080`.
    - Server opens the configured SQLite database and becomes the only component accessing SQLite directly.
    - Embedded web assets are served successfully from the server process.
    PRIORITY: 1
    DEPENDS-ON: E1-S1

    STORY: Add startup banner, version, and verbose logging
    ID: E1-S4
    DESCRIPTION: Make server startup and CLI entry output conform to the documented UX requirements.
    AC:
    - Running `task` with no arguments prints the colored ASCII-art TASK banner above usage text.
    - Running `task server` prints the same banner before the startup message.
    - `task server` prints embedded version and resolved DB path under the banner.
    - `task server -v` logs request and response details to stdout.
    PRIORITY: 3
    DEPENDS-ON: E1-S3

EPIC: Authentication And User Management
ID: E2
DESCRIPTION: Provide secure authentication, session persistence, admin-only user management, and CLI credential resolution for server and web access.
AC:
- Passwords are stored as Argon2id hashes in SQLite.
- Admins can create, list, delete, enable, and disable users.
- CLI and web support login and logout.
- CLI resolves credentials and server URL in the documented precedence order.
- Non-admin access to admin-only user management endpoints returns HTTP 403 with a clear error.
- `task status` reports server and client versions and warns when they differ.
PRIORITY: 1
DEPENDS-ON: E1

    STORY: Persist users and password hashes securely
    ID: E2-S1
    DESCRIPTION: Implement user storage with Argon2id password hashing and role support.
    AC:
    - User records include username, password hash, role, display name, enabled flag, and created timestamp.
    - Plaintext passwords are never stored.
    - Login validates credentials against Argon2id hashes.
    PRIORITY: 1
    DEPENDS-ON: E1-S1

    STORY: Implement admin-only user management APIs and CLI
    ID: E2-S2
    DESCRIPTION: Add user create, list, delete, enable, and disable flows for administrators.
    AC:
    - `task user create`, `task user ls`, `task user delete`, `task user enable`, and `task user disable` call the server API.
    - Authenticated non-admin callers receive HTTP 403 with an error explaining they are not an admin.
    - Disabled users cannot authenticate successfully.
    PRIORITY: 1
    DEPENDS-ON: E2-S1

    STORY: Implement login, register, logout, and session reuse
    ID: E2-S3
    DESCRIPTION: Support CLI authentication, stored sessions, and registration behavior.
    AC:
    - `task login` checks stored credentials first, then config username, then flags/env, then prompts.
    - Interactive password prompts mask input with `*`.
    - Invalid login prints `invalid credentials` before retry.
    - Successful login writes session token to `$TASK_HOME/credentials.json` and updates config defaults.
    - `task register` creates an account but does not persist a session.
    - `task logout` removes `$TASK_HOME/credentials.json`.
    PRIORITY: 1
    DEPENDS-ON: E2-S1

    STORY: Implement CLI config, URL resolution, and status inspection
    ID: E2-S4
    DESCRIPTION: Persist non-sensitive defaults locally and expose client/server status information.
    AC:
    - CLI stores non-sensitive defaults in `$TASK_HOME/config.json`.
    - Server URL resolves from flag, env, saved config, then `http://localhost:8080`.
    - Credentials resolve from flag, env, then OS defaults where specified.
    - `task status` prints server version and client version and warns on mismatch.
    PRIORITY: 2
    DEPENDS-ON: E2-S3

    STORY: Implement onboarding and version commands
    ID: E2-S5
    DESCRIPTION: Expose setup helpers and embedded version information in the CLI.
    AC:
    - `task onboard` appends the embedded `cmd/task/AGENTS.md` template into `${CWD}/AGENTS.md`, creating it if needed.
    - `task version` prints the semantic version embedded at build time.
    - `make build` increments the patch component in `VERSION` before compiling.
    PRIORITY: 3
    DEPENDS-ON: E1-S3

EPIC: Project Management
ID: E3
DESCRIPTION: Allow users to create, inspect, list, enable or disable, and locally select projects as the top-level container for work.
AC:
- Users can create, list, inspect, update, enable, and disable projects through the server-backed CLI.
- `task project list` shows at least id, title, and status and indicates the current local project.
- All create commands return the newly created ID to stdout.
- The active project is remembered locally for CLI defaults.
PRIORITY: 2
DEPENDS-ON: E2

    STORY: Create and list projects
    ID: E3-S1
    DESCRIPTION: Implement project creation and list retrieval over the shared API.
    AC:
    - `task project create` accepts title, description, and acceptance criteria.
    - `task project list` and `task project ls` return server-backed project data.
    - Successful project creation prints the new project ID to stdout.
    PRIORITY: 1
    DEPENDS-ON: E2-S3

    STORY: Inspect and update project details
    ID: E3-S2
    DESCRIPTION: Support project detail retrieval and field-level updates.
    AC:
    - `task project get <id>` returns project details.
    - `task project <id> update` supports title, description, and acceptance criteria changes.
    - Project changes are persisted through the server API.
    PRIORITY: 2
    DEPENDS-ON: E3-S1

    STORY: Enable, disable, and select active project
    ID: E3-S3
    DESCRIPTION: Add lifecycle status controls and local active-project selection.
    AC:
    - `task project <id> enable` and `task project <id> disable` update project status.
    - `task project use <id>` stores the selected project in local CLI config.
    - `task project list` visibly marks the current project.
    PRIORITY: 1
    DEPENDS-ON: E3-S1

EPIC: Work Item Capture And Retrieval
ID: E4
DESCRIPTION: Implement shared task, bug, and epic creation plus list, search, detail, and orphan retrieval across the active project.
AC:
- Users can create tasks, bugs, and epics with documented aliases and defaults.
- CLI create commands support title, description, acceptance criteria, priority, assignee, parent, and project selection.
- Users can list, filter, search, inspect, and retrieve orphaned items.
- `-json` is supported on client-facing commands with pretty-printed JSON output.
PRIORITY: 2
DEPENDS-ON: E3

    STORY: Create tasks, bugs, and epics with aliases and defaults
    ID: E4-S1
    DESCRIPTION: Implement low-friction work-item creation commands on the shared task model.
    AC:
    - `task add`, `task create`, and `task new` are aliases.
    - `task bug` creates a task with type `bug`.
    - `task epic` creates a task with type `epic`.
    - Default values apply for type, priority, assignee, description, acceptance criteria, parent, and active project.
    PRIORITY: 1
    DEPENDS-ON: E3-S3

    STORY: List and filter work items
    ID: E4-S2
    DESCRIPTION: Implement tabular item listing with server-side filters and limits.
    AC:
    - `task list` and `task ls` are aliases.
    - `task list --type`, `task list --status`, and `task list -u` apply server-side filtering.
    - `task list -n <limit>` applies a server-side limit, with `0` meaning no limit.
    - Default list output includes id, type, status, assignee, priority, and title.
    PRIORITY: 1
    DEPENDS-ON: E4-S1

    STORY: Search and inspect task detail
    ID: E4-S3
    DESCRIPTION: Provide search over titles and descriptions and a flat detail view for a single item.
    AC:
    - `task search <query>` searches across title and description fields.
    - `task get <id>` prints the required flat detail fields in the documented format.
    - `task get` and `task search` support `-json` pretty-printed output.
    PRIORITY: 2
    DEPENDS-ON: E4-S1

    STORY: List orphaned items
    ID: E4-S4
    DESCRIPTION: Surface work items with no parent to help users organize hierarchy.
    AC:
    - `task orphans` returns only items where `parent_id` is null.
    - Results are scoped to the active project unless an explicit project is supplied.
    - Orphan listing is available in normal table output and `-json` output.
    PRIORITY: 3
    DEPENDS-ON: E4-S2

EPIC: Workflow, Assignment, Dependencies, And Hierarchy
ID: E5
DESCRIPTION: Support status transitions, assignment rules, dependency management, and parent-child relationships for epics and child work.
AC:
- CLI and web support the default status set: `notready`, `open`, `inprogress`, `complete`, `fail`.
- Assignment and claim flows enforce admin and assignee rules.
- Dependency add/remove flows update task relationships.
- Parent-child hierarchy is supported through `parent_id` and an active epic CLI context.
PRIORITY: 2
DEPENDS-ON: E4, E2

    STORY: Implement status transition commands
    ID: E5-S1
    DESCRIPTION: Add explicit commands and generic update support for status progression.
    AC:
    - `task open`, `task ready`, `task inprogress`, `task complete`, and `task fail` update task status correctly.
    - `task ready <id>` behaves as an alias for `task open <id>`.
    - `task update <id> -status <status>` validates against the supported status set.
    PRIORITY: 1
    DEPENDS-ON: E4-S1

    STORY: Implement admin assignment and self-service claim flows
    ID: E5-S2
    DESCRIPTION: Support admin assignment plus request, claim, and unclaim behavior for end users.
    AC:
    - `task assign` and `task unassign` are admin-only and fail for missing or disabled target users.
    - `task claim <id>` fails if another user already owns the task.
    - `task unclaim <id>` fails if the caller is not the current assignee.
    - Non-admin users cannot override another user assignment through the generic update API.
    PRIORITY: 1
    DEPENDS-ON: E2-S2, E5-S1

    STORY: Implement task request routing
    ID: E5-S3
    DESCRIPTION: Add deterministic assignment behavior for `task request` with and without an explicit task id.
    AC:
    - `task request <id>` returns `{"status":"ASSIGNED","task":...}` when assignment succeeds or `{"status":"REJECTED"}` otherwise.
    - `task request` returns current inprogress work first, then oldest assigned open work, then oldest unassigned open work in the active project.
    - `task request` returns `{"status":"NO-WORK"}` when no assignable work exists.
    PRIORITY: 1
    DEPENDS-ON: E5-S2

    STORY: Implement dependency management
    ID: E5-S4
    DESCRIPTION: Support adding and removing task dependencies from the CLI and API.
    AC:
    - `task dependency add <id> <dependency-id[,dependency-id...]>` persists dependencies.
    - `task dependency remove <id> <dependency-id[,dependency-id...]>` removes dependencies.
    - `task get <id>` includes dependency data in its detail view.
    PRIORITY: 2
    DEPENDS-ON: E4-S3

    STORY: Implement epic hierarchy and active epic context
    ID: E5-S5
    DESCRIPTION: Support lightweight parent-child organization for epics, tasks, and bugs.
    AC:
    - Tasks and bugs can be created with a `parent_id` pointing to an epic.
    - CLI stores and uses an active epic context for faster entry where applicable.
    - Hierarchy is retrievable for both CLI detail views and web browsing.
    PRIORITY: 2
    DEPENDS-ON: E4-S1

EPIC: History, Comments, And Auditability
ID: E6
DESCRIPTION: Track meaningful item changes with append-only history and support comments attached to work items.
AC:
- Important task changes generate append-only history events.
- Users can add comments to items.
- CLI exposes `task history <id>` and `task comment add <id>`.
- Item detail views surface history and comments.
PRIORITY: 3
DEPENDS-ON: E4, E5

    STORY: Persist append-only history events
    ID: E6-S1
    DESCRIPTION: Record auditable events for important task changes in the history store.
    AC:
    - History events capture project id, task id, event type, payload, created time, and actor.
    - Task creation, update, status change, assignee change, parent change, and comment add events are stored.
    - History records are append-only.
    PRIORITY: 1
    DEPENDS-ON: E4-S1, E5-S1

    STORY: Add comments to work items
    ID: E6-S2
    DESCRIPTION: Support comment creation and persistence linked to tasks.
    AC:
    - `task comment add <id> "<text>"` creates a comment on the target task.
    - Comment creation also emits a history event.
    - Comments are returned on item detail endpoints.
    PRIORITY: 1
    DEPENDS-ON: E6-S1

    STORY: Expose history in CLI and item detail views
    ID: E6-S3
    DESCRIPTION: Make audit trails visible to users in terminal and web interfaces.
    AC:
    - `task history <id>` returns readable event output.
    - Item detail responses include history and comments.
    - Web item detail pages display both history and comments.
    PRIORITY: 2
    DEPENDS-ON: E6-S2

EPIC: Embedded Web Application
ID: E7
DESCRIPTION: Deliver an embedded single-page web application for collaborative project and task management on top of the same API used by the CLI.
AC:
- The web app is embedded into the Go binary with `go:embed`.
- Users can log in, switch projects, add and edit items, browse hierarchy, manage status on a board, and inspect history and comments.
- Normal use does not require manual page refresh.
- The web app is multi-user aware and reflects shared server state.
PRIORITY: 3
DEPENDS-ON: E1, E2, E3, E4, E5, E6

    STORY: Embed and serve SPA assets from the Go binary
    ID: E7-S1
    DESCRIPTION: Package the web client into the server binary and serve it from the main process.
    AC:
    - Web assets are embedded with `go:embed`.
    - Server serves the SPA entrypoint and static assets correctly.
    - Browser navigation works with SPA routing.
    PRIORITY: 1
    DEPENDS-ON: E1-S3

    STORY: Implement project-aware board and item management UI
    ID: E7-S2
    DESCRIPTION: Build the main work-management UI for project switching, item entry, and status board operations.
    AC:
    - Users can switch between projects in the UI.
    - Users can create and edit tasks, bugs, and epics.
    - Board view groups items by status and supports status changes.
    - Hierarchy is visible in the UI.
    PRIORITY: 1
    DEPENDS-ON: E3-S3, E4-S1, E5-S1, E5-S5

    STORY: Implement item detail, history, comments, and live refresh
    ID: E7-S3
    DESCRIPTION: Provide detailed item inspection with collaborative refresh behavior.
    AC:
    - Item detail pages show core fields, history, and comments.
    - New comments and status changes appear without manual page refresh during normal use.
    - UI reflects updates from other connected users in near real time.
    PRIORITY: 2
    DEPENDS-ON: E6-S3

EPIC: Quality Gates And Release Verification
ID: E8
DESCRIPTION: Establish automated checks and release validation so the product meets the documented build, test, and versioning requirements.
AC:
- Repository provides `make build`, `make test`, `make test-go`, and `make test-playwright`.
- `make build` increments the patch version in `VERSION` before the Go build.
- Changes are not complete until relevant automated checks pass.
PRIORITY: 2
DEPENDS-ON: E1, E2, E3, E4, E5, E6, E7

    STORY: Implement build-time version bump and embedding
    ID: E8-S1
    DESCRIPTION: Wire the build pipeline to increment the patch version and embed it into the binary and served assets.
    AC:
    - `make build` updates `VERSION` by incrementing the patch component.
    - The built binary reports the same embedded version through `task version`.
    - Server startup and status output use the embedded version value.
    PRIORITY: 1
    DEPENDS-ON: E1-S4, E2-S5

    STORY: Add Go automated tests
    ID: E8-S2
    DESCRIPTION: Cover domain, service, repository, and HTTP behavior with automated Go tests.
    AC:
    - `make test-go` runs Go unit and integration tests.
    - Tests cover auth, project management, task workflows, history, and permissions.
    - Key failure cases such as HTTP 403 admin-only rejection are exercised.
    PRIORITY: 1
    DEPENDS-ON: E2-S2, E5-S3, E6-S1

    STORY: Add end-to-end web and CLI verification
    ID: E8-S3
    DESCRIPTION: Validate primary user workflows through integration and browser automation.
    AC:
    - `make test-playwright` covers login, project switch, task creation, board updates, and item detail views.
    - `make test` runs the full automated suite.
    - Release validation confirms the documented success criteria end to end.
    PRIORITY: 2
    DEPENDS-ON: E7-S3, E8-S2
```

If you want, I can also reshape this into a smaller backlog with fewer epics and fewer stories per epic.
tokens used
10,520
I couldn’t create [test.md](/Users/simon/code/task/test.md) because the workspace is mounted read-only in this session. Below is the exact content I would put in that file.

```md
EPIC: Workspace Bootstrap And Runtime
ID: E1
DESCRIPTION: Initialize the local SQLite workspace, bootstrap the admin account and default project, and start the combined API and embedded web server from the Go binary.
AC:
- `task initdb` creates a new SQLite schema in the requested or default database path.
- `task initdb` creates the `admin` user and a default project.
- `task initdb` respects `--force` overwrite behavior and prints a generated password when `-password` is omitted.
- `task server` uses `$TASK_HOME/task.db` when `-f` is omitted.
- `task server` prints the colored TASK banner, embedded version, and resolved database path on startup.
- `task server -v` prints verbose request and response details.
PRIORITY: 1
DEPENDS-ON:

    STORY: Initialize SQLite workspace from CLI
    ID: E1-S1
    DESCRIPTION: Implement `task initdb` to create schema, bootstrap admin, and create the default project.
    AC:
    - Command accepts `-f`, `--force`, and `-password`.
    - Default database path resolves to `$TASK_HOME/task.db`.
    - Admin username is always `admin`.
    - Existing database is overwritten only when `--force` is supplied.
    PRIORITY: 1
    DEPENDS-ON:

    STORY: Generate and print bootstrap password when omitted
    ID: E1-S2
    DESCRIPTION: Support secure password generation for `task initdb` when no password flag is provided.
    AC:
    - A random password is generated when `-password` is absent.
    - Generated password is printed to stdout exactly once during initialization.
    - `task initdb` does not read `TASK_USERNAME` or `TASK_PASSWORD`.
    PRIORITY: 2
    DEPENDS-ON: E1-S1

    STORY: Start combined server and embedded web app
    ID: E1-S3
    DESCRIPTION: Implement `task server` to expose the HTTP API and embedded SPA from the same binary.
    AC:
    - Default local server listens on `http://localhost:8080`.
    - Server opens the configured SQLite database and becomes the only component accessing SQLite directly.
    - Embedded web assets are served successfully from the server process.
    PRIORITY: 1
    DEPENDS-ON: E1-S1

    STORY: Add startup banner, version, and verbose logging
    ID: E1-S4
    DESCRIPTION: Make server startup and CLI entry output conform to the documented UX requirements.
    AC:
    - Running `task` with no arguments prints the colored ASCII-art TASK banner above usage text.
    - Running `task server` prints the same banner before the startup message.
    - `task server` prints embedded version and resolved DB path under the banner.
    - `task server -v` logs request and response details to stdout.
    PRIORITY: 3
    DEPENDS-ON: E1-S3

EPIC: Authentication And User Management
ID: E2
DESCRIPTION: Provide secure authentication, session persistence, admin-only user management, and CLI credential resolution for server and web access.
AC:
- Passwords are stored as Argon2id hashes in SQLite.
- Admins can create, list, delete, enable, and disable users.
- CLI and web support login and logout.
- CLI resolves credentials and server URL in the documented precedence order.
- Non-admin access to admin-only user management endpoints returns HTTP 403 with a clear error.
- `task status` reports server and client versions and warns when they differ.
PRIORITY: 1
DEPENDS-ON: E1

    STORY: Persist users and password hashes securely
    ID: E2-S1
    DESCRIPTION: Implement user storage with Argon2id password hashing and role support.
    AC:
    - User records include username, password hash, role, display name, enabled flag, and created timestamp.
    - Plaintext passwords are never stored.
    - Login validates credentials against Argon2id hashes.
    PRIORITY: 1
    DEPENDS-ON: E1-S1

    STORY: Implement admin-only user management APIs and CLI
    ID: E2-S2
    DESCRIPTION: Add user create, list, delete, enable, and disable flows for administrators.
    AC:
    - `task user create`, `task user ls`, `task user delete`, `task user enable`, and `task user disable` call the server API.
    - Authenticated non-admin callers receive HTTP 403 with an error explaining they are not an admin.
    - Disabled users cannot authenticate successfully.
    PRIORITY: 1
    DEPENDS-ON: E2-S1

    STORY: Implement login, register, logout, and session reuse
    ID: E2-S3
    DESCRIPTION: Support CLI authentication, stored sessions, and registration behavior.
    AC:
    - `task login` checks stored credentials first, then config username, then flags/env, then prompts.
    - Interactive password prompts mask input with `*`.
    - Invalid login prints `invalid credentials` before retry.
    - Successful login writes session token to `$TASK_HOME/credentials.json` and updates config defaults.
    - `task register` creates an account but does not persist a session.
    - `task logout` removes `$TASK_HOME/credentials.json`.
    PRIORITY: 1
    DEPENDS-ON: E2-S1

    STORY: Implement CLI config, URL resolution, and status inspection
    ID: E2-S4
    DESCRIPTION: Persist non-sensitive defaults locally and expose client/server status information.
    AC:
    - CLI stores non-sensitive defaults in `$TASK_HOME/config.json`.
    - Server URL resolves from flag, env, saved config, then `http://localhost:8080`.
    - Credentials resolve from flag, env, then OS defaults where specified.
    - `task status` prints server version and client version and warns on mismatch.
    PRIORITY: 2
    DEPENDS-ON: E2-S3

    STORY: Implement onboarding and version commands
    ID: E2-S5
    DESCRIPTION: Expose setup helpers and embedded version information in the CLI.
    AC:
    - `task onboard` appends the embedded `cmd/task/AGENTS.md` template into `${CWD}/AGENTS.md`, creating it if needed.
    - `task version` prints the semantic version embedded at build time.
    - `make build` increments the patch component in `VERSION` before compiling.
    PRIORITY: 3
    DEPENDS-ON: E1-S3

EPIC: Project Management
ID: E3
DESCRIPTION: Allow users to create, inspect, list, enable or disable, and locally select projects as the top-level container for work.
AC:
- Users can create, list, inspect, update, enable, and disable projects through the server-backed CLI.
- `task project list` shows at least id, title, and status and indicates the current local project.
- All create commands return the newly created ID to stdout.
- The active project is remembered locally for CLI defaults.
PRIORITY: 2
DEPENDS-ON: E2

    STORY: Create and list projects
    ID: E3-S1
    DESCRIPTION: Implement project creation and list retrieval over the shared API.
    AC:
    - `task project create` accepts title, description, and acceptance criteria.
    - `task project list` and `task project ls` return server-backed project data.
    - Successful project creation prints the new project ID to stdout.
    PRIORITY: 1
    DEPENDS-ON: E2-S3

    STORY: Inspect and update project details
    ID: E3-S2
    DESCRIPTION: Support project detail retrieval and field-level updates.
    AC:
    - `task project get <id>` returns project details.
    - `task project <id> update` supports title, description, and acceptance criteria changes.
    - Project changes are persisted through the server API.
    PRIORITY: 2
    DEPENDS-ON: E3-S1

    STORY: Enable, disable, and select active project
    ID: E3-S3
    DESCRIPTION: Add lifecycle status controls and local active-project selection.
    AC:
    - `task project <id> enable` and `task project <id> disable` update project status.
    - `task project use <id>` stores the selected project in local CLI config.
    - `task project list` visibly marks the current project.
    PRIORITY: 1
    DEPENDS-ON: E3-S1

EPIC: Work Item Capture And Retrieval
ID: E4
DESCRIPTION: Implement shared task, bug, and epic creation plus list, search, detail, and orphan retrieval across the active project.
AC:
- Users can create tasks, bugs, and epics with documented aliases and defaults.
- CLI create commands support title, description, acceptance criteria, priority, assignee, parent, and project selection.
- Users can list, filter, search, inspect, and retrieve orphaned items.
- `-json` is supported on client-facing commands with pretty-printed JSON output.
PRIORITY: 2
DEPENDS-ON: E3

    STORY: Create tasks, bugs, and epics with aliases and defaults
    ID: E4-S1
    DESCRIPTION: Implement low-friction work-item creation commands on the shared task model.
    AC:
    - `task add`, `task create`, and `task new` are aliases.
    - `task bug` creates a task with type `bug`.
    - `task epic` creates a task with type `epic`.
    - Default values apply for type, priority, assignee, description, acceptance criteria, parent, and active project.
    PRIORITY: 1
    DEPENDS-ON: E3-S3

    STORY: List and filter work items
    ID: E4-S2
    DESCRIPTION: Implement tabular item listing with server-side filters and limits.
    AC:
    - `task list` and `task ls` are aliases.
    - `task list --type`, `task list --status`, and `task list -u` apply server-side filtering.
    - `task list -n <limit>` applies a server-side limit, with `0` meaning no limit.
    - Default list output includes id, type, status, assignee, priority, and title.
    PRIORITY: 1
    DEPENDS-ON: E4-S1

    STORY: Search and inspect task detail
    ID: E4-S3
    DESCRIPTION: Provide search over titles and descriptions and a flat detail view for a single item.
    AC:
    - `task search <query>` searches across title and description fields.
    - `task get <id>` prints the required flat detail fields in the documented format.
    - `task get` and `task search` support `-json` pretty-printed output.
    PRIORITY: 2
    DEPENDS-ON: E4-S1

    STORY: List orphaned items
    ID: E4-S4
    DESCRIPTION: Surface work items with no parent to help users organize hierarchy.
    AC:
    - `task orphans` returns only items where `parent_id` is null.
    - Results are scoped to the active project unless an explicit project is supplied.
    - Orphan listing is available in normal table output and `-json` output.
    PRIORITY: 3
    DEPENDS-ON: E4-S2

EPIC: Workflow, Assignment, Dependencies, And Hierarchy
ID: E5
DESCRIPTION: Support status transitions, assignment rules, dependency management, and parent-child relationships for epics and child work.
AC:
- CLI and web support the default status set: `notready`, `open`, `inprogress`, `complete`, `fail`.
- Assignment and claim flows enforce admin and assignee rules.
- Dependency add/remove flows update task relationships.
- Parent-child hierarchy is supported through `parent_id` and an active epic CLI context.
PRIORITY: 2
DEPENDS-ON: E4, E2

    STORY: Implement status transition commands
    ID: E5-S1
    DESCRIPTION: Add explicit commands and generic update support for status progression.
    AC:
    - `task open`, `task ready`, `task inprogress`, `task complete`, and `task fail` update task status correctly.
    - `task ready <id>` behaves as an alias for `task open <id>`.
    - `task update <id> -status <status>` validates against the supported status set.
    PRIORITY: 1
    DEPENDS-ON: E4-S1

    STORY: Implement admin assignment and self-service claim flows
    ID: E5-S2
    DESCRIPTION: Support admin assignment plus request, claim, and unclaim behavior for end users.
    AC:
    - `task assign` and `task unassign` are admin-only and fail for missing or disabled target users.
    - `task claim <id>` fails if another user already owns the task.
    - `task unclaim <id>` fails if the caller is not the current assignee.
    - Non-admin users cannot override another user assignment through the generic update API.
    PRIORITY: 1
    DEPENDS-ON: E2-S2, E5-S1

    STORY: Implement task request routing
    ID: E5-S3
    DESCRIPTION: Add deterministic assignment behavior for `task request` with and without an explicit task id.
    AC:
    - `task request <id>` returns `{"status":"ASSIGNED","task":...}` when assignment succeeds or `{"status":"REJECTED"}` otherwise.
    - `task request` returns current inprogress work first, then oldest assigned open work, then oldest unassigned open work in the active project.
    - `task request` returns `{"status":"NO-WORK"}` when no assignable work exists.
    PRIORITY: 1
    DEPENDS-ON: E5-S2

    STORY: Implement dependency management
    ID: E5-S4
    DESCRIPTION: Support adding and removing task dependencies from the CLI and API.
    AC:
    - `task dependency add <id> <dependency-id[,dependency-id...]>` persists dependencies.
    - `task dependency remove <id> <dependency-id[,dependency-id...]>` removes dependencies.
    - `task get <id>` includes dependency data in its detail view.
    PRIORITY: 2
    DEPENDS-ON: E4-S3

    STORY: Implement epic hierarchy and active epic context
    ID: E5-S5
    DESCRIPTION: Support lightweight parent-child organization for epics, tasks, and bugs.
    AC:
    - Tasks and bugs can be created with a `parent_id` pointing to an epic.
    - CLI stores and uses an active epic context for faster entry where applicable.
    - Hierarchy is retrievable for both CLI detail views and web browsing.
    PRIORITY: 2
    DEPENDS-ON: E4-S1

EPIC: History, Comments, And Auditability
ID: E6
DESCRIPTION: Track meaningful item changes with append-only history and support comments attached to work items.
AC:
- Important task changes generate append-only history events.
- Users can add comments to items.
- CLI exposes `task history <id>` and `task comment add <id>`.
- Item detail views surface history and comments.
PRIORITY: 3
DEPENDS-ON: E4, E5

    STORY: Persist append-only history events
    ID: E6-S1
    DESCRIPTION: Record auditable events for important task changes in the history store.
    AC:
    - History events capture project id, task id, event type, payload, created time, and actor.
    - Task creation, update, status change, assignee change, parent change, and comment add events are stored.
    - History records are append-only.
    PRIORITY: 1
    DEPENDS-ON: E4-S1, E5-S1

    STORY: Add comments to work items
    ID: E6-S2
    DESCRIPTION: Support comment creation and persistence linked to tasks.
    AC:
    - `task comment add <id> "<text>"` creates a comment on the target task.
    - Comment creation also emits a history event.
    - Comments are returned on item detail endpoints.
    PRIORITY: 1
    DEPENDS-ON: E6-S1

    STORY: Expose history in CLI and item detail views
    ID: E6-S3
    DESCRIPTION: Make audit trails visible to users in terminal and web interfaces.
    AC:
    - `task history <id>` returns readable event output.
    - Item detail responses include history and comments.
    - Web item detail pages display both history and comments.
    PRIORITY: 2
    DEPENDS-ON: E6-S2

EPIC: Embedded Web Application
ID: E7
DESCRIPTION: Deliver an embedded single-page web application for collaborative project and task management on top of the same API used by the CLI.
AC:
- The web app is embedded into the Go binary with `go:embed`.
- Users can log in, switch projects, add and edit items, browse hierarchy, manage status on a board, and inspect history and comments.
- Normal use does not require manual page refresh.
- The web app is multi-user aware and reflects shared server state.
PRIORITY: 3
DEPENDS-ON: E1, E2, E3, E4, E5, E6

    STORY: Embed and serve SPA assets from the Go binary
    ID: E7-S1
    DESCRIPTION: Package the web client into the server binary and serve it from the main process.
    AC:
    - Web assets are embedded with `go:embed`.
    - Server serves the SPA entrypoint and static assets correctly.
    - Browser navigation works with SPA routing.
    PRIORITY: 1
    DEPENDS-ON: E1-S3

    STORY: Implement project-aware board and item management UI
    ID: E7-S2
    DESCRIPTION: Build the main work-management UI for project switching, item entry, and status board operations.
    AC:
    - Users can switch between projects in the UI.
    - Users can create and edit tasks, bugs, and epics.
    - Board view groups items by status and supports status changes.
    - Hierarchy is visible in the UI.
    PRIORITY: 1
    DEPENDS-ON: E3-S3, E4-S1, E5-S1, E5-S5

    STORY: Implement item detail, history, comments, and live refresh
    ID: E7-S3
    DESCRIPTION: Provide detailed item inspection with collaborative refresh behavior.
    AC:
    - Item detail pages show core fields, history, and comments.
    - New comments and status changes appear without manual page refresh during normal use.
    - UI reflects updates from other connected users in near real time.
    PRIORITY: 2
    DEPENDS-ON: E6-S3

EPIC: Quality Gates And Release Verification
ID: E8
DESCRIPTION: Establish automated checks and release validation so the product meets the documented build, test, and versioning requirements.
AC:
- Repository provides `make build`, `make test`, `make test-go`, and `make test-playwright`.
- `make build` increments the patch version in `VERSION` before the Go build.
- Changes are not complete until relevant automated checks pass.
PRIORITY: 2
DEPENDS-ON: E1, E2, E3, E4, E5, E6, E7

    STORY: Implement build-time version bump and embedding
    ID: E8-S1
    DESCRIPTION: Wire the build pipeline to increment the patch version and embed it into the binary and served assets.
    AC:
    - `make build` updates `VERSION` by incrementing the patch component.
    - The built binary reports the same embedded version through `task version`.
    - Server startup and status output use the embedded version value.
    PRIORITY: 1
    DEPENDS-ON: E1-S4, E2-S5

    STORY: Add Go automated tests
    ID: E8-S2
    DESCRIPTION: Cover domain, service, repository, and HTTP behavior with automated Go tests.
    AC:
    - `make test-go` runs Go unit and integration tests.
    - Tests cover auth, project management, task workflows, history, and permissions.
    - Key failure cases such as HTTP 403 admin-only rejection are exercised.
    PRIORITY: 1
    DEPENDS-ON: E2-S2, E5-S3, E6-S1

    STORY: Add end-to-end web and CLI verification
    ID: E8-S3
    DESCRIPTION: Validate primary user workflows through integration and browser automation.
    AC:
    - `make test-playwright` covers login, project switch, task creation, board updates, and item detail views.
    - `make test` runs the full automated suite.
    - Release validation confirms the documented success criteria end to end.
    PRIORITY: 2
    DEPENDS-ON: E7-S3, E8-S2
```

If you want, I can also reshape this into a smaller backlog with fewer epics and fewer stories per epic.
