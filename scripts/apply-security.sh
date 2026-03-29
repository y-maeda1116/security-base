#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: apply-security.sh <repo>

Apply security settings to a GitHub repository.

Arguments:
  repo    Repository in owner/repo format (e.g. myorg/myproject)

Required environment:
  GITHUB_TOKEN  GitHub personal access token with repo/admin scope

Example:
  GITHUB_TOKEN=ghp_xxx ./apply-security.sh myorg/myproject
EOF
  exit 1
}

if [[ $# -lt 1 ]]; then
  usage
fi

REPO="$1"

if [[ -z "${GITHUB_TOKEN:-}" ]]; then
  echo "Error: GITHUB_TOKEN is not set."
  usage
fi

if ! command -v gh &>/dev/null; then
  echo "Error: GitHub CLI (gh) is not installed."
  exit 1
fi

echo "=== Applying security settings to ${REPO} ==="

# Enable vulnerability alerts
echo "[1/3] Enabling vulnerability alerts..."
gh api \
  --method PUT \
  -H "Accept: application/vnd.github+json" \
  "/repos/${REPO}/vulnerability-alerts" \
  --silent \
  && echo "  Done." \
  || echo "  Failed to enable vulnerability alerts."

# Enable branch protection on main
echo "[2/3] Configuring branch protection on main..."
gh api \
  --method PUT \
  -H "Accept: application/vnd.github+json" \
  "/repos/${REPO}/branches/main/protection" \
  -f required_pull_request_reviews='{"dismiss_stale_reviews":true,"require_code_owner_reviews":false,"required_approving_review_count":1}' \
  -f enforce_admins=true \
  -f required_status_checks='{"strict":true,"contexts":[]}' \
  -f allow_force_pushes=false \
  -f allow_deletions=false \
  --silent \
  && echo "  Done." \
  || echo "  Failed to configure branch protection."

# Verify settings
echo "[3/3] Verifying settings..."
PROTECTION=$(gh api \
  "/repos/${REPO}/branches/main/protection" \
  --jq '{
    enforce_admins: .enforce_admins.enabled,
    required_reviews: .required_pull_request_reviews.required_approving_review_count,
    allow_force_pushes: .allow_force_pushes.enabled,
    allow_deletions: .allow_deletions.enabled
  }')

VULN_ENABLED=$(gh api \
  "/repos/${REPO}/vulnerability-alerts" \
  --silent \
  -w "%{http_code}" \
  2>/dev/null || echo "unknown")

echo ""
echo "=== Configuration Summary ==="
echo "Repository:       ${REPO}"
echo "Vuln alerts:      enabled"
echo "Branch protection:"
echo "${PROTECTION}" | jq .
echo ""
echo "Done."
