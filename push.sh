#!/usr/bin/env bash
# Initialize git and push the Moonlight apps server to GitHub.
#
# Usage: ./push.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

REMOTE_URL="https://github.com/harys-rifai/moonlight-mon.git"

# Initialize git repo if not already initialized
if [ ! -d ".git" ]; then
    echo "Initializing git repository..."
    git init
fi

# Add all files
echo "Staging all changes..."
git add .

# Commit (create initial commit if nothing to commit)
if git diff --cached --quiet; then
    echo "No changes to commit."
else
    git commit -m "Initial commit: Moonlight apps server with PostgreSQL support"
fi

# Add remote origin if not already added
if ! git remote get-url origin >/dev/null 2>&1; then
    echo "Adding remote origin..."
    git remote add origin "$REMOTE_URL"
else
    echo "Remote origin already exists."
fi

# Rename branch to main
git branch -M main

# Push to origin main
echo "Pushing to origin main..."
git push -u origin main

echo "Done! Repository pushed to $REMOTE_URL"
