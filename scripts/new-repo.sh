#!/usr/bin/env bash
set -euo pipefail

OWNER="y-maeda1116"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SECURITY_BASE_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

usage() {
  show_help
  exit 1
}

show_help() {
  cat <<'EOF'
Usage: new-repo.sh <name> --template <ts|py|go> [options]

Create a new repository from a security-base template and apply all
security settings in one go.

Arguments:
  name           Repository name (e.g. my-new-project)

Options:
  --template     Template repository: ts (ts-template-base),
                 py (python-template-base), go (template-go-cross)
  --description  Repository description
  --private      Create as private repo
                 (gh-infra management is skipped: rulesets are not
                 available for private repos on GitHub Free)
  --dry-run      Print planned commands without executing anything
  -h, --help     Show this help

Example:
  ./scripts/new-repo.sh my-new-project --template ts --description "説明"
EOF
}

die() {
  echo "Error: $1" >&2
  exit 1
}

info() {
  echo "==> $1"
}

REPO_NAME=""
TEMPLATE_KIND=""
DESCRIPTION=""
VISIBILITY="public"
DRY_RUN=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --template)
      [[ $# -ge 2 ]] || usage
      TEMPLATE_KIND="$2"
      shift 2
      ;;
    --description)
      [[ $# -ge 2 ]] || usage
      DESCRIPTION="$2"
      shift 2
      ;;
    --private)
      VISIBILITY="private"
      shift
      ;;
    --dry-run)
      DRY_RUN=true
      shift
      ;;
    -h|--help)
      show_help
      exit 0
      ;;
    -*)
      die "Unknown option: $1 (see --help)"
      ;;
    *)
      [[ -z "$REPO_NAME" ]] || die "Multiple repository names given"
      REPO_NAME="$1"
      shift
      ;;
  esac
done

[[ -n "$REPO_NAME" ]] || usage
[[ -n "$TEMPLATE_KIND" ]] || die "--template is required (ts / py / go)"

case "$TEMPLATE_KIND" in
  ts) TEMPLATE_REPO="ts-template-base" ;;
  py) TEMPLATE_REPO="python-template-base" ;;
  go) TEMPLATE_REPO="template-go-cross" ;;
  *) die "Invalid --template '$TEMPLATE_KIND' (ts / py / go)" ;;
esac

if [[ ! "$REPO_NAME" =~ ^[A-Za-z0-9._-]+$ ]]; then
  die "Invalid repository name: $REPO_NAME"
fi

FULL_REPO="${OWNER}/${REPO_NAME}"

# --- Preflight -------------------------------------------------------------

command -v gh >/dev/null 2>&1 || die "GitHub CLI (gh) is not installed."
gh auth status >/dev/null 2>&1 || die "gh is not authenticated (run 'gh auth login')."
gh extension list 2>/dev/null | grep -q "babarot/gh-infra" \
  || die "gh-infra extension is not installed (see gh-infra/README.md)."

if gh repo view "$FULL_REPO" >/dev/null 2>&1; then
  die "Repository ${FULL_REPO} already exists."
fi

# gh-infra 登録は security-base のローカルリポジトリを書き換えるため、
# 現在 main にいて gh-infra/ に未コミット変更がないことを要求する
if [[ "$VISIBILITY" == "public" ]]; then
  current_branch="$(git -C "$SECURITY_BASE_ROOT" branch --show-current)"
  [[ "$current_branch" == "main" ]] \
    || die "security-base local repo must be on 'main' (current: ${current_branch})."
  [[ -z "$(git -C "$SECURITY_BASE_ROOT" status --porcelain -- gh-infra/)" ]] \
    || die "security-base has uncommitted changes under gh-infra/. Commit or revert them first."
fi

run() {
  if $DRY_RUN; then
    echo "[dry-run] $*"
  else
    "$@"
  fi
}

# リポジトリ作成後に失敗した場合、無防護な状態で残ることを警告する
REPO_CREATED=false
warn_partial_failure() {
  if [[ "$REPO_CREATED" == "true" ]]; then
    {
      echo ""
      echo "WARNING: script failed after creating ${FULL_REPO}."
      echo "  The repository may be missing security settings (rulesets, Dependabot, secret scanning)."
      echo "  Apply them manually:"
      echo "    gh infra apply ${SECURITY_BASE_ROOT}/gh-infra/${OWNER}/ -r ${FULL_REPO} --auto-approve"
      echo "    ${SCRIPT_DIR}/apply-security.sh ${FULL_REPO}"
    } >&2
  fi
}
trap warn_partial_failure ERR

echo "=== Creating ${FULL_REPO} from ${TEMPLATE_REPO} (${VISIBILITY}) ==="

# --- 1. Create repository from template ------------------------------------

info "[1/5] Creating repository from template..."
create_args=(repo create "$FULL_REPO" --template "${OWNER}/${TEMPLATE_REPO}" --"$VISIBILITY")
[[ -n "$DESCRIPTION" ]] && create_args+=(--description "$DESCRIPTION")
run gh "${create_args[@]}"
$DRY_RUN || REPO_CREATED=true

# --- 2. Register in gh-infra ------------------------------------------------

if [[ "$VISIBILITY" == "public" ]]; then
  info "[2/5] Registering in gh-infra (targets.txt / repos.yaml)..."
  run "${SCRIPT_DIR}/../gh-infra/sync-repos.sh"

  info "[3/5] Planning and applying gh-infra settings..."
  run gh infra plan "${SECURITY_BASE_ROOT}/gh-infra/${OWNER}/" -r "$FULL_REPO"
  run gh infra apply "${SECURITY_BASE_ROOT}/gh-infra/${OWNER}/" -r "$FULL_REPO" --auto-approve
else
  info "[2/5] Skipping gh-infra registration (private repo)."
  info "[3/5] Skipping gh-infra apply."
fi

# --- 4. Apply vulnerability / scanning settings -----------------------------

info "[4/5] Applying security settings (apply-security.sh)..."
run "${SCRIPT_DIR}/apply-security.sh" "$FULL_REPO"
# apply-security.sh は ステップ3 の GitHub 側設定を即時有効化するため
# Dependabot の自動セキュリティ修正もここで有効化する
run gh api --method PUT "repos/${FULL_REPO}/automated-security-fixes" --silent

# --- 5. Commit gh-infra registration ----------------------------------------

if [[ "$VISIBILITY" == "public" ]]; then
  info "[5/5] Committing gh-infra registration and creating PR..."
  if $DRY_RUN; then
    echo "[dry-run] git add gh-infra && git commit && git push -u origin <branch> && gh pr create"
  else
    if [[ -z "$(git -C "$SECURITY_BASE_ROOT" status --porcelain -- gh-infra/)" ]]; then
      echo "  No gh-infra changes to commit (already registered?)."
    else
      branch="chore/register-${REPO_NAME}"
      git -C "$SECURITY_BASE_ROOT" checkout -b "$branch"
      git -C "$SECURITY_BASE_ROOT" add gh-infra/
      git -C "$SECURITY_BASE_ROOT" commit \
        -m "chore: register ${REPO_NAME} in gh-infra management

Created from ${TEMPLATE_REPO} via scripts/new-repo.sh.
protect-main ruleset / SHA pinning 等の共通設定を適用済み。
apply-security.sh で脆弱性系設定を有効化済み。"
      git -C "$SECURITY_BASE_ROOT" push -u origin "$branch"
      # スクリプトは cd しないため、CWD ではなく明示的にリポジトリと head を指定する
      gh pr create --repo "${OWNER}/security-base" --head "${OWNER}:${branch}" \
        --title "chore: register ${REPO_NAME} in gh-infra management" \
        --body "## 概要
新規リポジトリ \`${FULL_REPO}\` を gh-infra の宣言管理に登録する。
\`scripts/new-repo.sh --template ${TEMPLATE_KIND}\` による自動作成。

## 適用済み設定
- ruleset \`protect-main\` (non_fast_forward / deletion)
- merge_strategy.auto_delete_head_branches
- Actions SHA pinning required
- vulnerability alerts / PVR / secret scanning & push protection
- automated security fixes (Dependabot)

## テストプラン
- [x] \`gh infra validate\` / \`plan\` / \`apply\` を実行済み
- [x] \`apply-security.sh\` 実行済み"
    fi
  fi
else
  info "[5/5] Skipping PR creation (private repo)."
fi

echo ""
echo "=== Summary ==="
echo "Repository:       ${FULL_REPO}"
echo "Template:         ${TEMPLATE_REPO}"
echo "Visibility:       ${VISIBILITY}"
if [[ "$VISIBILITY" == "public" ]]; then
  echo "gh-infra PR:      created in security-base (merge it to record management state)"
fi
echo ""
echo "Next: clone it and start development."
echo "  git clone git@github.com:${FULL_REPO}.git"
echo "Done."
