# Design

## Product Summary

`task` is a lightweight ticket and project management system delivered as a single Go binary.

It is designed for small teams that want low-friction task tracking without separate infrastructure for the API, database, and web UI. The product combines a server, a terminal-first CLI, and an embedded web application around one shared data model.

The system has three interfaces:

1. A server that owns persistence, authentication, and collaboration.
2. A CLI for fast, explicit terminal workflows.
3. An embedded web application for browsing, editing, and status management.

## Product Principles

1. The server is the single system of record.
2. The CLI and web app use the same API semantics and data model.
3. Common operations should be fast and predictable from the terminal.
4. Projects should support lightweight hierarchy through epics and child tasks.
5. Every meaningful change should be traceable through history and comments.

## Primary Users And Workflows

The primary user is a small software team managing projects, epics, tasks, bugs, and notes.

The first release must support these workflows end to end:

1. Initialize a local SQLite-backed workspace.
1. use argon2id for encrpytion in sqlite
2. Start the server and embedded web app from the same binary.
3. Create and manage users.
4. Authenticate from the CLI and the web app.
5. Create and select projects.
6. Add work items such as tasks, bugs, notes, questions, and epics.
7. List, filter, search, and inspect items.
8. Organize work beneath an active epic.
9. Review item history and comments.
10. Manage work visually in the web app, including status-based board views.

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
- `slug`
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
- `note`
- `question`

Model notes:

- `parent_id` is nullable and supports hierarchical work
- tasks created while an active epic is selected can default to that epic as parent
- notes and questions are lightweight task records rather than a separate subsystem

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

### Comment

Comments provide discussion on work items.

- `id`
- `item_id`
- `user_id`
- `comment`
- `created_at`

## Functional Scope

### Workspace Initialization

The product must support local initialization of a SQLite database from the CLI.

The first run should allow the operator to create an initial administrator account.

Representative flow:

```bash
task init -f filename.db -username admin -password password
```

### Server

The server is the system of record.

Responsibilities:

- manage SQLite persistence
- expose the HTTP API for CLI and web use
- enforce authentication and authorization
- serve the embedded web application
- support multi-user access
- provide near-real-time refresh for connected clients

The default local server should listen on `http://localhost:8000`.

### Authentication And User Management

The first release must support:

1. administrator bootstrap during initialization
2. user creation by administrators
3. enable and disable user accounts
4. login and logout from CLI and web
5. user identity inspection from the CLI

Representative commands:

```bash
task user create -username alice -password secret
task user enable -username alice
task user disable -username alice
task user list

task register
task login
task whoami
task status
task logout
```

The CLI may also read credentials from environment variables such as `TASK_USERNAME` and `TASK_PASSWORD`.

### Project Management

Users must be able to:

1. create projects
2. list projects
3. inspect project details
4. select an active project for CLI defaults

Representative commands:

```bash
task project create "Customer Portal"
task project list
task project use customer-portal
task project get customer-portal
task project
```

The selected project should be remembered locally by the CLI.

### Work Item Capture

Creating work should be low-friction.

Users must be able to create:

- tasks
- bugs
- notes
- questions
- epics

Representative commands:

```bash
task add "Customers can reset their password."
task note "Need audit history for password changes."
task question "Should invited but inactive users be able to reset passwords?"
task bug "Reset token fails after first use."
task epic "Authentication"
task create -type task "Add password reset audit event"
```

Behavior notes:

- `task add` defaults to a normal task
- creating an epic can make that epic the active parent context for subsequent child tasks
- each item records project, creator, timestamps, status, and revision history

### Review And Search

Users must be able to:

1. list all items in the active project
2. filter by type
3. filter by status
4. search across titles and descriptions
5. inspect full item detail

Representative commands:

```bash
task list
task list --type bug
task list --status open
task search "password reset"
task get 42
```

### Workflow And Status Management

The system should support basic review and task progression through status changes.

The exact status set may be configured later, but the first release should support a small default set suitable for list and board views, for example:

- `open`
- `in_progress`
- `blocked`
- `done`

The CLI and web app must both make status changes easy.

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
3. item detail pages in the CLI and web app that surface both

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
task project use customer-portal

task epic "Authentication"
task add "Customers can reset their password."
task bug "Reset token expires immediately."
task list
task get 42
task search "password reset"
task history 42
```

The CLI should support long and short aliases where they improve usability, but the canonical commands should stay consistent.

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
- inspect history and discussion

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

Changes are not complete until the relevant automated checks pass.

## Success Criteria

The product is successful if a user can:

1. initialize a local workspace and start the server
2. create users and authenticate successfully
3. create and switch projects quickly
4. add tasks, bugs, notes, questions, and epics with minimal friction
5. inspect work through list, search, detail, history, and comments
6. manage work visually through the web interface
