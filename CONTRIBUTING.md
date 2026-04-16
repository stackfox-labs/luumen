# Contributing to Luumen

Thank you for contributing to Luumen.

This guide explains how to set up the project, make changes safely, and open a pull request.

## Ways to contribute

- Report bugs
- Improve docs
- Add tests
- Fix issues
- Propose or implement features aligned with the project spec

For larger changes, open an issue first so scope and direction can be agreed before implementation.

## Project layout

- cmd/: CLI entrypoints
- internal/: core implementation (commands, config, tooling orchestration)
- docs/: website and docs frontend (Vite + React)

## Prerequisites

- Go 1.25+
- Git
- Node.js and npm (only needed when working on docs/)

## Local setup

1. Clone the repository.
2. From repository root, verify Go is available:

```bash
go version
```

3. Run tests:

```bash
go test ./...
```

4. Run the CLI locally:

```bash
go run ./cmd/luu --help
```

Optionaly install the development binary for faster iteration:

```bash
go install ./cmd/luu-dev
```

## Working on docs

If your contribution touches docs or website UI:

```bash
cd docs
npm install
npm run dev
```

Before opening a PR for docs changes, run:

```bash
npm run lint
npm run typecheck
npm run build
```

## Development guidelines

- Keep changes focused and small when possible.
- Prefer explicit behavior and clear error messages.
- Avoid unrelated refactors in the same pull request.
- Add or update tests when changing behavior.
- Preserve existing command UX unless a change is intentional and documented.

## Testing expectations

Run all Go tests before opening a PR:

```bash
go test ./...
```

If you changed docs/frontend files under docs/, also run:

```bash
npm run lint
npm run typecheck
npm run build
```

## Pull request process

1. Create a branch from main.
2. Make your changes.
3. Add or update tests and docs as needed.
4. Run the relevant checks locally.
5. Open a pull request with:
   - A clear summary
   - Why the change is needed
   - Test evidence (commands run and outcomes)

## Commit messages

Use clear, descriptive commit messages that explain intent.

Examples:

- cli: improve add command tool resolution errors
- config: validate luumen.toml task entries
- docs: fix installation instructions

## Release notes

Tagged releases are built by GitHub Actions for supported operating systems and architectures.

If your change is user-facing, include a PR description that can be reused in release notes.

## Questions

If anything is unclear, open an issue and ask before spending time on a large implementation.
