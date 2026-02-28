# Introduction

`task` is a requirements gathering tool.   It is intended to gather and refine requirements into a specification which can be given to a software factory.

```bash
make build
```

> Note: if you run `make reset` and you use `VSCode` I advise you to restart VSCode as the tasks daemon is a bit flaky.

```bash
task count
```

## Create requirements

This will generate tasks instructions from the requirements document.

```bash
task count
```

## see what ralph would do

```bash
wiggum check -name ralph check
```

## Open VSCode and install a tasks plugin

Observe the kanban

## simulate a tasks loop

```bash
# name is ralph and ralph works fast
# max is 0 (loop till done)
# dryrun means dont really do the work but simulate it
export PATH=./bin:$PATH
wiggum loop -name ralph -max 1 -dryrun 5 -sleep 1

wiggum loop -name ralph -max 1 -dryrun 5 -sleep 1
```



Refresh the kanban

## Building

```bash
go install github.com/simonski/task@latest
```

## Running

```bash
task init
task server
```

## Usage

Either via the website `http://localhost:8000` or via the terminal using `task`



