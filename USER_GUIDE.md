# User Guide

`task` is a ticket management tool.

This guide describes a single Go binary that provides a server, a CLI, and an embedded web application backed by SQLite.

## How `task` Works

`task` has three interfaces:

1. The server, which owns persistence, authentication, and collaboration.
2. The CLI, which provides fast and explicit terminal workflows.
3. The web app, which is embedded in the same binary and uses the same API.

All project data follows the server data model and API semantics, whether you are working against a remote server or a local workspace.

Client-side files live under `$TASK_HOME`, which defaults to `~/.config/task`.

- `$TASK_HOME/config.json` stores non-sensitive client defaults such as the current username, server URL, and active project
- `$TASK_HOME/credentials.json` stores the current session token

## Getting Started

Write the local agent instructions template into the current repository:

```bash
task onboard
```

`task onboard` appends the embedded onboarding template into `${CWD}/AGENTS.md`. If the file does not exist yet, it is created.

Initialize a task sqlite database:

```bash
task initdb
```

If `-f` is omitted, `task initdb` creates the SQLite database at `$TASK_HOME/task.db`.

`task initdb` creates:

1. an `admin` account
2. the default project, `Default Project` with project id `1`

Bootstrap resolution works like this:

- admin username: always `admin`
- admin password: `-password` if provided, otherwise a generated random password printed to stdout
- existing database file: overwritten only when `--force` is supplied

Start the server:

```bash
task server
```

If `-f` is omitted, `task server` uses `$TASK_HOME/task.db`.

If `-v` is supplied, `task server` prints verbose request and response logs to stdout.

On startup, `task server` also prints a colored ASCII-art `TASK` banner before the listen message.

Immediately below the banner it prints:

- the embedded version
- the resolved task database path

By default the web app is available at `http://localhost:8080`.

Show the current CLI version:

```bash
task version
```

`task version` prints the semantic version embedded into the binary at build time. Each `make build` increments that semantic version before compiling the binary.

Running `task` with no arguments prints a colored ASCII-art `TASK` banner above the main usage output.

If you are using the CLI against a running server on another host, configure TASK_SERVER first:

```bash
export TASK_SERVER=http://your-server:8080
```

As an admin create users:

```bash
task user create --username XXXX --password YYYY
created user xxxxx
```

As an admin enable/disable users:

```bash
task user enable --username XXXX
task user disable --username XXXX
task user ls|list
task user rm|delete --username XXXX
```

These commands are admin-only. If a logged-in non-admin user runs them, the server returns `403` and the CLI prints `user is not an admin`.

## Accounts And Login

Create an account:

```bash
task register --username name --password '*******'
```

Log in:

```bash
task login -username name -password '*******'
```

For `task register`, you can omit the flags and let the CLI resolve them from `TASK_USERNAME` and `TASK_PASSWORD`. If those are not set, `task register` falls back to `whoami` and `password`.

`task login` resolves values in this order:

1. a valid session already stored in `$TASK_HOME/credentials.json`
2. the `username` already stored in `$TASK_HOME/config.json`
3. `-username` and `-password`
4. `TASK_USERNAME` and `TASK_PASSWORD`
5. interactive prompts for anything still missing

If login fails with `invalid credentials`, the CLI prints that message, prompts for username and password, and retries once.

When prompts are shown, any discovered values are presented as defaults that you can keep or replace.

When `task login` prompts for a password in an interactive terminal, typed characters are masked with `*`.

On successful login:

- the session token is stored in `$TASK_HOME/credentials.json`
- the `username` and `server_url` fields in `$TASK_HOME/config.json` are updated

Registering a user does not log that user in or create local session credentials.

Check the current mode and connection state:

```bash
task status
```

`task status` always prints the current effective configuration first, then performs a mode-appropriate connectivity check.

In REMOTE mode it prints:

- `mode: remote`
- `server: <TASK_SERVER>`
- `username: <configured username or blank>`
- `authenticated: true|false`

Then it calls the remote status endpoint and prints:

- `connection: success` in green if the server responds successfully
- `connection: failure` in red if the server cannot be contacted or returns an error

In LOCAL mode it prints:

- `mode: local`
- `db_path: <resolved database path>`
- `db_exists: true|false`

Then it opens the database if present and verifies the schema is usable. It prints:

- `connection: success` in green if the database can be opened and queried
- `connection: failure` in red if the database is missing, cannot be opened, or the schema is invalid

If the database does not exist in LOCAL mode, it also prints:

- `hint: run task initdb`

If `-nocolor` is set, the same output is printed without ANSI colors.

Show aggregate counts:

```bash
task count
15
task count -project_id 1
11
```

`task count` prints totals for users and work items by type. Without `-project_id` it also prints the total project count.

Log out:

```bash
task logout
```

`task logout` removes `$TASK_HOME/credentials.json`.

The web app uses the same account system. Once logged in, your session is shared across normal browser workflows.

## Typical Workflow

Most teams use `task` in this order:

1. Create or select a project.
2. Capture epics, tasks, and bugs.
3. Review and search what has been collected.
4. Assign, claim, and organize work.
5. Inspect dependencies and revision history.

## Projects

Create a project:

```bash
task project create -description "Portal backlog" -ac "Portal launch criteria" "Customer Portal"
```

The project is now the default project.

List projects:

```bash
task project list
task project ls
```

`task project list` prints the project id, title, and status, and marks the active project as `(current)`.

Select the active project for subsequent commands:

```bash
task project use 2
```

Show the current project:

```bash
task project
```

`task project` shows the current active project, or `no active project` if none is selected.

Get details on a project:

```bash
task project get <id>
task project 2
```

Update a project:

```bash
task project 2 update -title "New project title"
task project 2 update -description "The new description"
task project 2 update -ac "The acceptance criteria"
```

Enable or disable a project:

```bash
task project 2 enable
task project 2 disable
```

The active project is remembered by the CLI so you do not need to pass a project ID for every command.

## Capture Work

Capture is intentionally lightweight. You can add project work as soon as it appears, then organize it later.

Add a task (type defaults to task)

```bash
task add "Customers can reset their password."
```

These are equivalent:

```bash
task add "I am a new task"
task create "I am a new task"
task new "I am a new task"
task add -title "I am a new task"
```

Add a bug:

```bash
task bug "This is a bug"
```

Add an epic:

```bash
task epic "This is an Epic"
```

```bash
task create -t task -p 1 -a alice -d "This is a Task" -ac "Has a title and description" -estimate_effort 5 -estimate_complete 2026-04-30T17:00:00Z "This is a Task"
```

Creation defaults:

- `-t` / `-type`: defaults to `task`
- `-p` / `-priority`: defaults to `1`
- `-a` / `-assignee`: defaults to blank
- `-d` / `-description`: defaults to blank
- `-ac`: defaults to blank
- `-estimate_effort`: defaults to `0`
- `-estimate_complete`: defaults to blank and should use RFC3339 when set
- `-parent`: defaults to blank
- `-project`: defaults to the current project

Command aliases:

- `task add`, `task create`, and `task new` are the same command
- `task list` and `task ls` are the same command
- `task list -n <limit>` applies a server-side limit, where `0` means all results

Each captured item records:

- its project
- its author
- its creation time
- its current status
- its revision history

In the web app, use the capture panel at the top of the project page to create the same item types. Newly created items appear immediately for other connected users.

## Review And Search

List all items in the active project:

```bash
task list
task ls
task list -n 20
```

Filter by item kind:

```bash
task list --type task
task list --type bug
task list --type epic
```

Filter by status:

```bash
task list --status open
task list --status notready
task list --status inprogress
task list --status complete
task list --status fail
```

Filter by assignee:

```bash
task list -u alice
task ls -u alice
```

`task list` prints a table with the task id, type, status, assignee, priority, and title.

Search within the active project:

```bash
task search "password reset"
task search password reset -status open -owner alice
```

Search across all projects:

```bash
task search password reset -allprojects
```

Show a single item:

```bash
task get 42
task get -json 42
```

`task get` prints the task fields directly, including `DependsOn`, the acceptance criteria, `EstimateEffort`, `EstimateComplete`, `CloneOf` when the task is a clone, and comments ordered most recent first.

Show orphaned items with no parent:

```bash
task orphans
```

Assignment commands:

```bash
task assign 42 alice
task unassign 42 alice
task dependency add 4 1,2,3
task dependency remove 4 2
task claim 42
task unclaim 42
task request
task request 42
task set-parent 17 9
task unset-parent 17
```

`task assign` and `task unassign` are admin-only.

They also fail if the named user does not exist or is disabled.

`task claim` fails if another user is already assigned. `task unclaim` fails unless you are the current assignee.

`task request` asks the server to assign work to the current user. If the user already has an `inprogress` task, that task is returned. Otherwise, if the user has assigned `open` work, the oldest assigned `open` task is returned. If no work can be assigned, the JSON response status is `NO-WORK`. If a specific task is requested and cannot be assigned, the JSON response status is `REJECTED`.

Status commands:

```bash
task open 42
task ready 42
task inprogress 42
task complete 42
task fail 42
```

`task ready` is an alias for `task open`.

Most client-facing commands also support `-json` to pretty-print the JSON response.

Show the history of any item:

```bash
task history 17
```

`task history` prints the stored history events for that item.

In the web app, the item detail pane shows:

1. the current item
2. dependencies
3. comments
4. revision history

## Web Interface

The embedded web app is the easiest way to work visually across many related items.

Use it for:

1. capturing work during discovery and delivery
2. reviewing related items side by side
3. browsing task details and dependencies without switching commands

Because the CLI and web app use the same server API, edits made in one interface appear in the other without any import or sync step.

## Command Reference

```bash
task initdb
task server -v
task version

task register --username <name> --password <password>
task login --username <name> --password <password>
task status
task logout

task user create --username <name> --password <password>
task user ls
task user delete --username <name>
task user enable --username <name>
task user disable --username <name>

task project create "..."
task project list
task project ls
task project use <id>
task project
task project get <id>
task project <id>
task project <id> update -title "..."
task project <id> update -description "..."
task project <id> update -ac "..."
task project <id> enable
task project <id> disable

task add "..."
task bug "..."
task epic "..."

task list
task ls
task list --type task
task list --status open
task list -u <name>
task search "..."
task search "..." -allprojects
task get <id>
task history <id>
task comment add <id> "..."
task orphans

task dependency add <id> <id[,id...]>
task dependency remove <id> <id[,id...]>
task assign <id> <name>
task unassign <id> <name>
task claim <id>
task unclaim <id>
task request [<id>]
task set-parent <id> <parent-id>
task unset-parent <id>
task open <id>
task ready <id>
task inprogress <id>
task complete <id>
task fail <id>
task update <id> -status <status>
task count
task count -project_id <id>

task update <id> -status open
task update <id> -title "new title"
task update <id> -description "new description"
task update <id> -ac "new acceptance criteria"
task update <id> -priority 4
task update <id> -order 7
task update <id> -parent_id 12
task update <id> -estimate_effort 5
task update <id> -estimate_complete 2026-04-30T17:00:00Z
task update <id> -status open -priority 2 -title "new title"
task update <id> -status closed

```
