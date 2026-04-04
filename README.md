# Luumen

Luumen is a unified CLI for Roblox developers.

It provides one command surface for common project workflows while using the existing Roblox tooling ecosystem under the hood.

## Why Luumen

Roblox projects built outside Studio often rely on several separate tools for setup, dependency management, serving, builds, formatting, and linting.

Luumen exists to make those workflows easier to run and easier to standardize across repos.

Instead of learning a different set of setup steps and commands for every project, a repo can expose one consistent interface through `luu`.

## What Luumen uses

Luumen works with the existing ecosystem:

* **Rokit** for tool installation and management
* **Wally** for package dependencies
* **Rojo**, **Selene**, **StyLua**, **Lune**, and other tools for project workflows

It does not replace those tools.

## Commands

```bash
luu create
luu init
luu install
luu add
luu dev
luu build
luu lint
luu format
luu test
luu run <task>
luu doctor
```

## Examples

Create a project:

```bash
luu create my-game
cd my-game
luu dev
```

Create from a specific template:

```bash
luu create --template rojo-wally my-game
luu create --template rojo-only my-game
luu create --template lune-http my-script
luu create --template lute-guessing my-script
```

Set up an existing repo:

```bash
cd existing-repo
luu init
luu install
luu dev
```

Add a tool or package:

```bash
luu add rojo
luu add stylua
luu add sleitnick/knit
luu add tool:rojo-rbx/rojo
luu add pkg:sleitnick/knit
```

## Configuration

Luumen uses a repo config file named:

```text
luumen.toml
```

This file is used for workflow and task configuration.
It does not replace tool-specific config files such as:

* `rokit.toml`
* `wally.toml`
* Rojo project files

## Status

Luumen is under development.

## Goal

A Luumen repo should be easy to understand and easy to run.

A developer should be able to clone a repo and use:

```bash
luu install
luu dev
```

without having to learn a different command setup for every project.

## License

MIT License. See [LICENSE](LICENSE).
