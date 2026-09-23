#!/bin/bash
set -euo pipefail

REPOS_YAML="gh-infra/y-maeda1116/repos.yaml"
TARGETS_TXT="gh-infra/targets.txt"
OWNER="y-maeda1116"

echo "=== Fetching repositories from GitHub ==="
gh_repos=$(gh repo list "$OWNER" --visibility public -L 100 --json name --jq '.[].name' | sort)

echo "=== Parsing ${REPOS_YAML} ==="
yaml_repos=$(grep -E '^\s+- name:' "$REPOS_YAML" | sed 's/.*name: *//' | sort)

# standalone YAML (security-base.yaml など) で個別管理するリポは
# RepositorySet (repos.yaml) に追加しない
STANDALONE_REPOS="security-base"

echo "=== Detecting new repositories ==="
new_repos=$(comm -23 <(echo "$gh_repos") <(printf '%s\n%s\n' "$yaml_repos" "$STANDALONE_REPOS" | sort -u))

if [[ -z "$new_repos" ]]; then
  echo "No new repositories found. All up to date."
  exit 0
fi

echo "New repositories:"
echo "$new_repos"
echo ""

# repos.yaml の repositories セクションの最後にエントリを追加
for repo in $new_repos; do
  echo "Adding ${OWNER}/${repo} ..."

  # repos.yaml の最後の "  - name:" エントリの後に新しいエントリを追加
  # 末尾の改行を確保しつつ追記
  printf '\n  - name: %s\n' "$repo" >> "$REPOS_YAML"

  # targets.txt にも追加
  echo "${OWNER}/${repo}" >> "$TARGETS_TXT"
done

echo ""
echo "=== Sorting targets.txt ==="
sort -u -o "$TARGETS_TXT" "$TARGETS_TXT"

echo ""
echo "=== Validating ==="
if gh infra validate gh-infra/y-maeda1116/; then
  echo ""
  echo "Added $(echo "$new_repos" | wc -l | tr -d ' ') repo(s). Next steps:"
  echo "  gh infra plan y-maeda1116/"
  echo "  gh infra apply y-maeda1116/"
else
  echo "Validation failed. Reverting changes ..."
  git checkout -- "$REPOS_YAML" "$TARGETS_TXT"
  exit 1
fi
