# Git Hooks for Network Mapper

This directory contains git hooks that ensure code quality by running the same checks as the CI pipeline locally before commits.

## Setup

To install the git hooks, run:

```bash
cd network-mapper
./.githooks/install.sh
```

## What the Pre-commit Hook Does

The pre-commit hook runs the following checks using the devcontainer:

1. **Dependency Management**
   - `go mod download` - Downloads required modules
   - `go mod verify` - Verifies dependencies

2. **Code Quality**
   - `go test -v ./...` - Runs all tests
   - `go vet ./...` - Checks for common Go mistakes
   - `staticcheck ./...` - Advanced static analysis

3. **Code Formatting**
   - `gofmt -s -l .` - Ensures code is properly formatted

## Benefits

- **Catch Issues Early**: Problems are detected before they reach the repository
- **Consistent Environment**: Uses the same devcontainer as CI for consistency
- **Time Saving**: Reduces failed CI builds and back-and-forth fixes
- **Code Quality**: Maintains high code quality standards automatically

## Bypassing Hooks (Not Recommended)

In emergency situations, you can bypass the hooks with:

```bash
git commit --no-verify
```

However, this is not recommended as it may lead to CI failures.

## Uninstalling

To remove the git hooks:

```bash
git config --unset core.hooksPath
```

## Requirements

- [devcontainer CLI](https://github.com/devcontainers/cli): `npm install -g @devcontainers/cli`
- Docker (for running the devcontainer)

## Troubleshooting

If you encounter issues:

1. **Devcontainer not found**: Install the devcontainer CLI
2. **Permission denied**: Ensure hook files are executable (`chmod +x .githooks/*`)
3. **Docker issues**: Ensure Docker is running and accessible

The hooks are designed to be fast and reliable, but if you need to disable them temporarily during development, use `git commit --no-verify`.