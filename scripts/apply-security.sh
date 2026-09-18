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
  GITHUB_TOKEN="$(gh auth token 2>/dev/null || true)"
fi
if [[ -z "${GITHUB_TOKEN:-}" ]]; then
  echo "Error: GITHUB_TOKEN is not set and gh auth token failed."
  usage
fi

if ! command -v gh &>/dev/null; then
  echo "Error: GitHub CLI (gh) is not installed."
  exit 1
fi

echo "=== Applying security settings to ${REPO} ==="

# Enable vulnerability alerts
echo "[1/5] Enabling vulnerability alerts..."
gh api \
  --method PUT \
  -H "Accept: application/vnd.github+json" \
  "/repos/${REPO}/vulnerability-alerts" \
  --silent \
  && echo "  Done." \
  || echo "  Failed to enable vulnerability alerts."

# Enable private vulnerability reporting (SECURITY.md の報告窓口)
echo "[2/5] Enabling private vulnerability reporting..."
gh api \
  --method PUT \
  -H "Accept: application/vnd.github+json" \
  "/repos/${REPO}/private-vulnerability-reporting" \
  --silent \
  && echo "  Done." \
  || echo "  Failed to enable private vulnerability reporting."

# Enable secret scanning and push protection
# (public リポは無料 / private + GHAS なしの場合は失敗するため警告で続行)
echo "[3/5] Enabling secret scanning and push protection..."
gh api \
  --method PATCH \
  -H "Accept: application/vnd.github+json" \
  "/repos/${REPO}" \
  --input - <<'PAYLOAD' \
  && echo "  Done." \
  || echo "  Failed to enable secret scanning (requires GHAS on private repos)."
{
  "security_and_analysis": {
    "secret_scanning": {
      "status": "enabled"
    },
    "secret_scanning_push_protection": {
      "status": "enabled"
    }
  }
}
PAYLOAD

# Enable branch protection on main
echo "[4/5] Configuring branch protection on main..."
gh api \
  --method PUT \
  -H "Accept: application/vnd.github+json" \
  "/repos/${REPO}/branches/main/protection" \
  --input - <<'PAYLOAD' \
  && echo "  Done." \
  || echo "  Failed to configure branch protection."
{
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": true,
    "require_code_owner_reviews": false,
    "required_approving_review_count": 0
  },
  "enforce_admins": true,
  "required_status_checks": {
    "strict": true,
    "contexts": []
  },
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false
}
PAYLOAD

# Verify settings
echo "[5/5] Verifying settings..."
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
  2>/dev/null || echo "000")

# 204 = 有効, 404 = 無効
PVR_ENABLED=$(gh api \
  "/repos/${REPO}/private-vulnerability-reporting" \
  --silent \
  -w "%{http_code}" \
  2>/dev/null || echo "000")

SECRET_SCANNING=$(gh api \
  "/repos/${REPO}" \
  --jq '{
    secret_scanning: (.security_and_analysis.secret_scanning.status // "unavailable"),
    push_protection: (.security_and_analysis.secret_scanning_push_protection.status // "unavailable")
  }')

echo ""
echo "=== Configuration Summary ==="
echo "Repository:       ${REPO}"
if [[ "$VULN_ENABLED" == "204" ]]; then
  echo "Vuln alerts:      enabled"
else
  echo "Vuln alerts:      unknown (HTTP ${VULN_ENABLED})"
fi
if [[ "$PVR_ENABLED" == "204" ]]; then
  echo "PV reporting:     enabled"
else
  echo "PV reporting:     unknown (HTTP ${PVR_ENABLED})"
fi
echo "Secret scanning:"
echo "${SECRET_SCANNING}" | jq .
echo "Branch protection:"
echo "${PROTECTION}" | jq .
echo ""
echo "Done."
