#!/bin/bash
# Install git hooks for argazer project

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
HOOKS_DIR="$PROJECT_ROOT/.git/hooks"

echo "==> Installing git hooks for argazer..."

# Create pre-push hook
cat > "$HOOKS_DIR/pre-push" << 'EOF'
#!/bin/sh
# Pre-push hook to run linter and tests before pushing

echo "==> Running pre-push checks..."

# Run golangci-lint
echo "==> Running golangci-lint..."
if command -v golangci-lint >/dev/null 2>&1; then
    golangci-lint run --timeout=10m
    if [ $? -ne 0 ]; then
        echo "[ERROR] Linter found issues. Fix them before pushing."
        echo "        You can skip this check with: git push --no-verify"
        exit 1
    fi
    echo "[OK] Linter passed"
else
    echo "[WARN] golangci-lint not found, skipping lint check"
    echo "       Install it: https://golangci-lint.run/usage/install/"
fi

# Run tests
echo "==> Running tests..."
go test ./...
if [ $? -ne 0 ]; then
    echo "[ERROR] Tests failed. Fix them before pushing."
    echo "        You can skip this check with: git push --no-verify"
    exit 1
fi
echo "[OK] Tests passed"

echo "[OK] All pre-push checks passed!"
exit 0
EOF

chmod +x "$HOOKS_DIR/pre-push"

echo "[OK] Git hooks installed successfully!"
echo ""
echo "The following hooks are now active:"
echo "  - pre-push: Runs linter and tests before pushing"
echo ""
echo "To bypass the hook (not recommended), use: git push --no-verify"

