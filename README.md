# devkit

Personal dev workspace generator. Composes your identity, constraints, and company context into AI config files for any coding tool.

## Install

```sh
go install github.com/dklinux7/devkit/cmd/devkit@latest
```

## Usage

```sh
devkit init                    # set up ~/.devkit/
devkit generate ~/my-project   # write AI config files
devkit search "query"          # search your notes
```

Run `devkit help` for full details.

## What it does

You maintain markdown files describing how you work. devkit composes them and writes config files that Claude Code, Cursor, Copilot, Windsurf, and OpenCode all understand.

One source of truth → every AI tool gets the same context.

## Not licensed

This project is not licensed for use, modification, or distribution.
Source code is publicly visible for transparency and reference only.
All rights reserved.
