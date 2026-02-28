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
    - all tests pass
    - use make to verify tests
    - work in a branch that contains the EPIC and FEATURE name



3. Work on the REQUIREMENTS.md in order.  