# Contributing to Auto Ingress

Thank you for your interest in contributing! This document explains how to participate.

## Getting Started

### Prerequisites

- Go 1.22+
- Docker 20.10+
- Git

### Setup

```bash
git clone https://github.com/nodecoda/auto-ingress.git
cd auto-ingress
make build
make test
```

## Development Workflow

### 1. Fork and Branch

```bash
git checkout -b feat/your-feature
```

Branch naming:
- `feat/...` — new features
- `fix/...` — bug fixes
- `docs/...` — documentation
- `refactor/...` — refactoring
- `test/...` — test additions

### 2. Write Code

- Follow existing code style and patterns
- Add tests for new functionality
- Keep commits focused and well-described
- Run `go vet ./...` before committing

### 3. Test

```bash
# Run all tests
make test

# Lint
make lint

# Build
make build
```

### 4. Submit Pull Request

- Use a clear, descriptive title
- Reference related issues (e.g. `Fixes #42`)
- Describe what changed and why
- Include test results

## Reporting Issues

### Bug Reports

Include:
- Steps to reproduce
- Expected behavior
- Actual behavior
- Environment (OS, Docker version, Go version)
- Relevant logs or error messages

### Feature Requests

Include:
- Problem description
- Proposed solution
- Alternatives considered

## Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Use `gofmt` (automatic with most editors)
- Keep functions focused and small
- Write descriptive variable and function names
- Add comments for exported functions

## Commit Messages

Format:

```
<type>: <short summary>

<body (optional)>
```

Types:
- `feat` — new feature
- `fix` — bug fix
- `docs` — documentation
- `refactor` — code refactoring
- `test` — test changes
- `chore` — maintenance

Example:
```
feat: add wildcard domain support

Support *.example.com patterns in site configuration
for catch-all subdomain routing.
```

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
