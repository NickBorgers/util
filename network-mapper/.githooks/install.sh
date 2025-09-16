#!/bin/bash

# Script to install git hooks for the network-mapper project
# This sets up pre-commit hooks that run the same checks as CI

set -e

echo "🔧 Installing git hooks for network-mapper..."

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT=$(git rev-parse --show-toplevel)

# Set the git hooks directory to our custom .githooks directory
echo "📁 Setting git hooks directory..."
git config core.hooksPath "$SCRIPT_DIR"

echo "✅ Git hooks installed successfully!"
echo ""
echo "The following hooks are now active:"
echo "  • pre-commit: Runs CI checks using devcontainer before each commit"
echo ""
echo "To disable hooks temporarily, you can use:"
echo "  git commit --no-verify"
echo ""
echo "To uninstall hooks:"
echo "  git config --unset core.hooksPath"