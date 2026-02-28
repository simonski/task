# User Guide

`task` is a ticket management tool.

This guide describes the product as implemented in the design in [docs/DESIGN.md](/Users/simon/code/task/docs/DESIGN.md): a single Go binary that provides a server, a CLI, and an embedded web application backed by SQLite.

## How `task` Works

`task` has three interfaces:

1. The server, which owns persistence, authentication, and collaboration.
2. The CLI, which provides fast and explicit terminal workflows.
3. The web app, which is embedded in the same binary and uses the same API.

All project data lives on the server. Neither the CLI nor the web interface writes directly to the database.

## Getting Started

Initialize a task sqlite database

```bash
task init (-f filename.db -username admin -password password)
```

Note the `-username` and `-password` settings will set an administrator username and password which is used to create users that can then store tasks.

Start the server:

```bash
task server (-f filename.db)
```

By default the web app is available at `http://localhost:8000`.

If you are using the CLI against a running server on another host, configure TASK_URL first:

```bash
export TASK_URL=http://your-server:8000
```

As an admin create users:

```bash
task user create (-username XXXX -password YYYY)
username: xxxxx
password: xxxxx
```

As an admin enable/disable users:

```bash
task user enable (-username XXXX)
task user disable (-username XXXX)
task user ls|list
```


## Accounts And Login

Create an account:

```bash
task register
username: name
password: *******
```

Log in:

(or use TASK_USERNAME / TASK_PASSWORD)

```bash
task register
username: name
password: *******
```

Show the current authenticated user:

```bash
task whoami
```

Check the status of the user and connection

```bash
task status
```

Log out:

```bash
task logout
```

The web app uses the same account system. Once logged in, your session is shared across normal browser workflows.

## Typical Workflow

Most teams use `task` in this order:

1. Create or select a project.
2. Capture raw requirements, notes, and questions.
3. Review and search what has been collected.
4. Curate source material into structured requirements.
5. Record decisions and resolve open questions.
6. Inspect traceability and revision history.
7. Generate and export a specification.

## Projects

Create a project:

```bash
task project create "Customer Portal".
```

The project is now the default project.  

List projects:

```bash
task project list
```

Select the active project for subsequent commands:

```bash
task project use customer-portal
```

Show the current project commands

```bash
task project
```

# get details on a project
```bash
task project get <project-name or id>
```

The active project is remembered by the CLI so you do not need to pass a project ID for every command.  

## Capture Source Material

Capture is intentionally lightweight. You can add raw material as soon as it appears, then organize it later.

Add a task (type defaults to task)

```bash
task add "Customers can reset their password."
```

Add a note:

```bash
task note "Need audit history for password changes."
```

Add a question:

```bash
task question "Should invited but inactive users be able to reset passwords?"
```

Add a bug:

```bash
task bug "This is a bug"
```

Add an epic

```bash
task epic "This is an Epic"
```

the current epic will now be thihs epic id.

```bash
task create -type task "This is a Task"
```

This task is now created and attached to the most recent epic.

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
```

Filter by item kind:

```bash
task list --type raw_requirement
task list --type note
task list --type question
task list --type requirement
task list --type decision
```

Filter by status:

```bash
task list --status proposed
task list --status accepted
```

Search across titles and bodies:

```bash
task search "password reset"
```

Show a single item:

```bash
task show|get 42
```

The item detail view includes content, metadata, trace links, and recent history events.

## Curate Requirements

Curation turns source material into structured requirements while preserving the original records.

Curate one source item:

```bash
task curate 42
```

Curate several related items together:

```bash
task curate 42 43 44
```

When a curation is created, `task`:

1. creates a new `requirement` item
2. stores links back to the source items
3. records a history event for the curation

In the web app, you can select multiple source items and curate them together from the review view.

## Review And Decision Workflow

Curated requirements can be reviewed before they become part of the accepted project baseline.

Show items waiting for review:

```bash
task review
```

Show only proposed requirements:

```bash
task review --status proposed
```

Accept a requirement:

```bash
task accept requirement 17
```

Reject a requirement:

```bash
task reject requirement 18
```

Revise an existing requirement:

```bash
task revise requirement 17
```

Revision creates a new history event and preserves the previous state for auditability.

## Questions And Decisions

Questions and decisions are first-class artifacts in `task`.

Create a decision:

```bash
task decision add "Reset links expire after 15 minutes."
```

List decisions in the current project:

```bash
task decision list
```

Link a decision to the question it resolves:

```bash
task trace link --from decision:24 --to question:11 --rel answers
```

This makes it possible to see which questions remain open and which requirements are backed by explicit decisions.

## Traceability

Traceability is a core feature of `task`. Every important artifact can be linked to the material that led to it.

Show upstream and downstream links for a requirement:

```bash
task trace requirement 17
```

Show the history of any item:

```bash
task history 17
```

Show a discussion or review conversation:

```bash
task conversation show 9
```

Common trace relationships include:

- `derived_from`
- `supports`
- `answers`
- `included_in`

In the web app, the item detail pane shows:

1. source inputs that support the item
2. related questions and decisions
3. specification sections that include it
4. revision history

## Specification Generation

Once a project has enough accepted requirements and decisions, generate a specification:

```bash
task spec generate
```

Show the generated specification:

```bash
task spec show
```

Show a single section:

```bash
task spec show --section goals
```

Trace a specification section back to its sources:

```bash
task spec trace 3
```

Export the specification as Markdown:

```bash
task spec export markdown
```

The exported document preserves section structure while allowing you to inspect the linked requirements inside `task`.

## Web Interface

The embedded web app is the easiest way to work visually across many related items.

Use it for:

1. capturing notes during discovery sessions
2. reviewing related items side by side
3. browsing trace links without switching commands
4. reading the generated specification as a structured document

Because the CLI and web app use the same server API, edits made in one interface appear in the other without any import or sync step.

## Command Reference

```bash
task init
task server

task auth register --username <name> --password <password>
task auth login --username <name> --password <password>
task auth whoami
task auth logout

task project create "..."
task project list
task project use ...
task project current
task project open ...

task add "..."
task note "..."
task question "..."

task list
task list --type requirement
task list --status proposed
task show <id>
task search "..."

task curate <id>
task curate <id> <id>
task curate --from-search "..."
task review
task accept requirement <id>
task reject requirement <id>
task revise requirement <id>

task decision add "..."
task decision list

task trace requirement <id>
task trace link --from decision:<id> --to question:<id> --rel answers
task history <id>
task conversation show <id>

task spec generate
task spec show
task spec show --section goals
task spec trace <id>
task spec export markdown
```
