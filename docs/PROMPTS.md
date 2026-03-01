0. <freeform converted to DESIGN.md>

1. Write a USER_GUIDE.md at the top based on a hypothetical implementation of this using the docs/DESIGN.md.    

Do not include how to run it, only from the perspective of a user in the terminal.

2. Using the DESIGN and USER_GUIDE write an example breakdown of implementation requirements as REQUIREMENTS.md in the format:

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

The intent is to take this output and model it in beads, or jira, or an issue tracker.  The scope is:
- ALL examples in the user guides
- ALL of the backend and frontend functionality as per the design

Note the DEPENDS-ON is a method of describing blocking features.

Ensure the acceptance critera contains
    - use red/green testing 
    - use make to verify all tests pass
    - work in a branch that contains the EPIC and TASK name for example feature/<epic>-<task>

3. Write/rewrite a parser go program that translates a requirements.md into `task` commands (but do not call `task`). It should just be a single go file runnable as "parser -f REQUIREMENTS.md" which writes to stdout all the `task` commands with double- newlines between them.   It should read the whole requirements, validate they are correct and have referntial integrity where they refer to other EPICS or STORIES, call out the error-line if there is one, exit 1 if there is a problem, or just print the commands and exit 0.

Each entry acceptance criteria should include a reference to look at RULES.md, DESIGN.md, USER_GUIDE.md as additional context.

Put this in tools/parser.go and update e the Makefile to have a `make tools` which builds a parser binary in the root

4. Work on the REQUIREMENTS in order.

-----

`-json` in client calls will pretty-print JSON as the response.

`task create|new|add I am a new task` should create a new task called "I am a new task"
Note: new,create,add are the same
Note: list,ls are the same
Note: rm,delete,del are the same

-title TITLE is the same as not specifying the title
-ac ACCEPTANCE_CRITERIA

If -t[ype] is unspecified, default to a task
If -p[riority] is unspecified, default to 1
If -a[ssignee] is unspecified, leave blank
If -d[escription] is unspecified, leave blank
If -parent is unspecifed, leave blank
If -project is unspecifed, use the current project

`task get <id>` should pretty print the entity by major headings.

An entity is deemed `orphaned` if it does not have a parent_id.  Orphans can be found with

`task orphans`

If a task is created, print to stdout the task id
If any client command fails, exit 1
If any client command succeeds, exit 0

`task count` should print the total number of everything by type
    users
    tasks 123 (50 completed, 75 in progress, 110)
    epics 10 (5 completed)
    projects (5)

`task count -project_id` should print the total number of everything by type for a given project
    users
    tasks 123 (50 completed, 75 in progress, 110)
    epics 10 (5 completed)

`task status` should ping the server and indicate the server version and the client version.
    It should warn if versions differ.

`task assign <id> <name>` is an admin only command that assigns a task ID to a user
`task unassign <id> <name>` is an admin only command that un-assigns a task ID to a user

`task claim <id>` assigns the caller to the task.  If another user is assigned, fail.  A user cannot override an assignment.
`task unclaim <id>` un-assigns the caller to the task.  If the user is not assigned, fail.   A user cannot override an asssigment.

`task ls,list -u[ser] <name>` lists all tasks assigned to the user

`task server` : below the "rainbow" task in the USAGE print the VERSION
`task server` : below the VERSION print the taskdb location.


task list 
    should be much nicer - print in a table perhaps?
    should incldue the assignee
    -n should limit number of responses on the server side (default 0 meaning all)



A task is worked on by one worker (the assignee)
A task can be in 3 stages [design, develop, test]
    - design [idle,inprogress,review,complete]
    - develop [idle,inprogress,review,complete]
    - test [idle,inprogress,review,complete]
OR
A task can be in 3 states: idle,inprogress,complete
A task can have two outcomes: success, failure
A task can be closed/archived/deleted to remove it from visibility

If a task has children, it cannot be complete unlesss all children are complete.

task state change commands
    task open 1             - moves state to open
    task close 1            - moves status to closed
    task ready 1            - moves ready state to true
    task unready 1          - moves ready state to false
    task fail 1             - moves state to failed
    task success 1          - moves state to success
    task active 1           - moves status to active
    task idle 1              - moves state to idle
    task inprogress 1        - moves state to inprogress
    task complete 1          - moves state to complete

`task onboard` should append an `${CWD}/AGENTS.md` file which is embeddedin the go code under cmd/task/AGENTS.md

group the CLI usage by admin commands and client commands
order the CLI commands alphabetically in their section

Ensure the CLI usage is up to date.
Update the code, DESIGN and USER_GUIDE for the above.  



`task get N` should return in format
ID           :
ParentID     :
ProjectID    :
Type         : task
Description  :
Title        :
Assignee     :
Order        :
DependsOn    : [1,2,3]
Status       :
Priority     :    
Created      :
LastModified :
Closed       :
Acceptance Criteria :

`task history N` should print the history.


Create the `add-dependency` `remove-dependency` commands.

If a task 4 depends on 3 other tasks (1, 2, 3) completing 

task dependency add 4 1,2,3

Now 4 depends-on 1,2,3.

LEt's task 4 does not depend on task 2

task dependency remove 4 2

note, the comma-separated ability for the tasks.

Ensure the CLI usage is up to date.
Update the code, DESIGN and USER_GUIDE for the above.  



remove slug from projects everywhere, cli, model, database.   

to add acceptance criteria
task project N update -ac "the acceptance criteria"

to update title or description
task project N update -title "the new title"
task project N update -description "the new description"

to add acceptance criteria
task project N update -ac "the acceptance criteria"

also make it an option when creating projects

## project status
task project N enable
task project N disable




## New instruction req

`task req -f file1,file2,file3 -o requirements.md` should read all files mentioned in -f and write to the -o filename the results of the prompt to an agent.  The agent should be prompted via a process invocation that receives the entire prompt.  

The agent should default to codex however can be overridden using `-agent` in which case e.g. a call to copilot coudl occurr using `copilot -p PROMPT`

PROMPT:
-------

Write an example breakdown of implementation requirements as OUTPUT_FILE in the format:

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
----
