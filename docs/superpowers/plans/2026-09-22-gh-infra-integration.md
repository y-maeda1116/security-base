# gh-infra Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** my-github-config の gh-infra 資材を security-base に移入し、ブランチ保護を rulesets に一本化、週次ドリフト検出 CI を追加する。

**Architecture:** `gh-infra/` ディレクトリに YAML とスクリプトを移入 (テンプレート同期対象外)。ブランチ保護は gh-infra rulesets (`security-base.yaml` の `required_status_checks` 強化を含む) に寄せ、apply-security.sh は脆弱性系 4 ステップにスリム化。週次 `gh infra plan --ci` でドリフトを検出し issue 起票 (既存 audit ワークフローと同型)。

**Tech Stack:** gh-infra (babarot/gh-infra) v0.13.0, GitHub Actions (ubuntu-latest), gh CLI, bash

**Spec:** `docs/superpowers/specs/2026-09-22-gh-infra-integration-design.md`

## Global Constraints

- `uses:` はコミットSHAピン留め必須 (checkout: `actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1`)
- `gh extension install` は `--tag v0.13.0` 固定 (`@latest` 禁止の規約に準拠)
- コミットメッセージは Conventional Commits (`feat:`, `ci:`, `docs:`, `chore:`, `refactor:`)
- `gh-infra/**` を `tools/sync/config.yaml` の files に追加しない (テンプレート誤配布防止)
- secret は env 経由で渡し、`${{ }}` 展開を `run:` の本文に埋め込まない
- `archive/` ディレクトリは移入しない
- 移入するスクリプトの中身 (パス修正を除く) は変更しない — my-github-config で実績のあるものをそのまま持ってくる

---

### Task 1: gh-infra 資材の移入

**Files:**
- Create: `gh-infra/README.md`
- Create: `gh-infra/targets.txt`
- Create: `gh-infra/sync-repos.sh`
- Create: `gh-infra/get-repo-list.sh`
- Create: `gh-infra/gh-infra-import.sh`
- Create: `gh-infra/y-maeda1116/repos.yaml`
- Create: `gh-infra/y-maeda1116/security-base.yaml`

**Interfaces:**
- Consumes: my-github-config (github.com/y-maeda1116/my-github-config) の資材
- Produces: `gh-infra/y-maeda1116/` 以下の YAML 群 (以降のタスクとワークフローが参照するパス)

- [ ] **Step 1: my-github-config を一時クローンして資材をコピー**

```bash
TMP=$(mktemp -d)
git clone --depth 1 https://github.com/y-maeda1116/my-github-config.git "$TMP/my-github-config"
mkdir -p gh-infra
cp -R "$TMP/my-github-config/y-maeda1116" gh-infra/y-maeda1116
rm -rf gh-infra/y-maeda1116/archive
cp "$TMP/my-github-config/targets.txt" \
   "$TMP/my-github-config/sync-repos.sh" \
   "$TMP/my-github-config/get-repo-list.sh" \
   "$TMP/my-github-config/gh-infra-import.sh" gh-infra/
rm -rf "$TMP"
```

注意: `archive/` は移入しない。`gh-infra/y-maeda1116/` には `repos.yaml` と `security-base.yaml` のみが残る。

- [ ] **Step 2: スクリプト3本のパスを修正**

3 スクリプトとも、リポジトリルートから実行する前提でパスに `gh-infra/` プレフィックスを付ける。

`gh-infra/get-repo-list.sh` を以下の全体にする:

```bash
#!/bin/bash
gh repo list y-maeda1116 --visibility public -L 100 --json nameWithOwner --jq '.[].nameWithOwner' > gh-infra/targets.txt
```

`gh-infra/sync-repos.sh` — 冒頭のパス変数と validate 呼び出しを修正:

```bash
REPOS_YAML="gh-infra/y-maeda1116/repos.yaml"
TARGETS_TXT="gh-infra/targets.txt"
OWNER="y-maeda1116"
```

(中略なし — 上記3行の置き換えのみ。最後の validate は)

```bash
if gh infra validate gh-infra/y-maeda1116/; then
```

(それ以外の行はクローンしたまま変更しない)

`gh-infra/gh-infra-import.sh` — 読み込み元と出力先のパスを修正:

```bash
sed -i 's/\r$//' gh-infra/targets.txt
```

```bash
  filepath="gh-infra/${repo}.yaml"
```

(それ以外の行は変更しない)

- [ ] **Step 3: 実行権限を維持**

Run: `chmod +x gh-infra/sync-repos.sh gh-infra/get-repo-list.sh gh-infra/gh-infra-import.sh && ls -l gh-infra/`
Expected: 3 スクリプトに x が付いている

- [ ] **Step 4: gh-infra/README.md を作成**

新規作成 (my-github-config の README を security-base 向けに改変):

```markdown
# リポジトリ設定の宣言管理 (gh-infra)

[gh-infra](https://github.com/babarot/gh-infra) を利用して、y-maeda1116 配下の
GitHub リポジトリ設定を YAML で宣言的に管理する。security-base が
「信頼の源泉」としてリポジトリ設定も管理する。

## ディレクトリ構成

```
gh-infra/
  sync-repos.sh             GitHub上の新規リポジトリを自動検出・追加
  get-repo-list.sh          リポジトリ一覧を targets.txt に出力
  gh-infra-import.sh        targets.txt のリポジトリを一括 import
  targets.txt               管理対象リポジトリ一覧
  y-maeda1116/
    repos.yaml              RepositorySet (全リポの共通設定)
    security-base.yaml      security-base 専用設定 (rulesets 強化)
```

## 前提条件

- [GitHub CLI](https://cli.github.com/) インストール済み
- gh-infra 拡張 v0.13.0:

```bash
gh extension install babarot/gh-infra --tag v0.13.0
```

## 運用コマンド (リポジトリルートから)

```bash
# 構文チェック
gh infra validate gh-infra/y-maeda1116/

# 差分確認 (GitHub 実状態との比較)
gh infra plan gh-infra/y-maeda1116/

# GitHub へ適用 (ローカル実行のみ。CI からは実行しない)
gh infra apply gh-infra/y-maeda1116/

# 特定リポジトリのみ操作
gh infra plan gh-infra/y-maeda1116/ -r y-maeda1116/til
```

## 新規リポジトリ追加

```bash
./gh-infra/sync-repos.sh                              # 自動検出して repos.yaml / targets.txt に追加
gh infra plan gh-infra/y-maeda1116/                   # 差分確認
gh infra apply gh-infra/y-maeda1116/                  # 共通設定を適用
./scripts/apply-security.sh y-maeda1116/<new-repo>    # 脆弱性系を適用
git add gh-infra && git commit && git push            # 管理状態を記録
```

## ブランチ保護について

classic branch protection ではなく **rulesets** で管理する
(`repos.yaml` defaults は `non_fast_forward` / `deletion`、
`security-base.yaml` は `required_status_checks` を含む強化版)。
`scripts/apply-security.sh` はブランチ保護を扱わない。

## ドリフト検出

`.github/workflows/gh-infra-plan.yml` が週次で `gh infra plan --ci` を実行し、
宣言状態と GitHub 実状態のドリフトを issue で検知する。
認証は読み取り専用 fine-grained PAT (repository secret `GH_INFRA_TOKEN`)。

注意: private リポジトリは GitHub Free で rulesets が使えない
(gh-infra の制約)。管理対象は public リポジトリのみ。

## 移行元

my-github-config (2026-09-22 移行)。
```

- [ ] **Step 5: 構文・スキーマ検証**

Run: `gh infra validate gh-infra/y-maeda1116/`
Expected: エラーなし (validate 通過)

- [ ] **Step 6: Commit**

```bash
git add gh-infra/
git commit -m "feat: import gh-infra repository settings from my-github-config"
```

### Task 2: repos.yaml / targets.txt を現行の公開リポに最新化

**Files:**
- Modify: `gh-infra/y-maeda1116/repos.yaml`
- Modify: `gh-infra/targets.txt`

**Interfaces:**
- Consumes: Task 1 の `gh-infra/sync-repos.sh` (パス修正済み)
- Produces: 現行の全公開リポを含む repos.yaml (29 前後。2026-09-22 時点の測定は 29)

- [ ] **Step 1: sync-repos.sh を実行**

Run: `./gh-infra/sync-repos.sh`
Expected: `New repositories:` に未管理リポが列挙され、`Added N repo(s).` と出て validate が通る。
(差分がなければ `No new repositories found.` で終了 — そのまま Step 2 へ)

- [ ] **Step 2: 追加結果を確認**

Run: `grep -c '  - name:' gh-infra/y-maeda1116/repos.yaml && sort -u gh-infra/targets.txt | wc -l`
Expected: 両方とも概ね同じ件数 (labels の `- name:` 行も grep に含まれるため、repos.yaml 側は「リポ数 + ラベル数」になる点に注意。正確には `git diff gh-infra/y-maeda1116/repos.yaml` の追加行を目視確認)

- [ ] **Step 3: Commit**

```bash
git add gh-infra/y-maeda1116/repos.yaml gh-infra/targets.txt
git commit -m "chore: sync gh-infra repo list to current public repositories"
```

### Task 3: security-base.yaml の rulesets を強化

**Files:**
- Modify: `gh-infra/y-maeda1116/security-base.yaml` (rulesets セクション)

**Interfaces:**
- Consumes: Task 1 で移入した `security-base.yaml` (ruleset `main`)
- Produces: `required_status_checks` を含む ruleset — Task 7 の適用対象、gh-infra-plan.yml の宣言状態

- [ ] **Step 1: rulesets セクションを修正**

`security-base.yaml` の `rulesets:` を以下に置き換える (`spec.rulesets` 配下。他のセクションは変更しない):

```yaml
  rulesets:
    - name: main
      target: branch
      enforcement: active
      conditions:
        ref_name:
          include:
            - refs/heads/main
      rules:
        non_fast_forward: true
        deletion: true
        required_status_checks:
          strict_required_status_checks_policy: true
          contexts:
            - context: "Python CI"
            - context: "Go Test (tools/sync)"
```

(従来の `rules:` に `required_status_checks` を追加する形。`non_fast_forward` / `deletion` は維持)

- [ ] **Step 2: 検証**

Run: `gh infra validate gh-infra/y-maeda1116/`
Expected: エラーなし

Run: `gh infra plan gh-infra/y-maeda1116/ -r y-maeda1116/security-base`
Expected: security-base に rulesets 適用の差分が表示される (読み取りのみ。適用はしない)

- [ ] **Step 3: Commit**

```bash
git add gh-infra/y-maeda1116/security-base.yaml
git commit -m "feat: require CI checks on main via rulesets in gh-infra config"
```

### Task 4: ドリフト検出ワークフローを作成

**Files:**
- Create: `.github/workflows/gh-infra-plan.yml`

**Interfaces:**
- Consumes: `gh-infra/y-maeda1116/` (Task 1-3 の YAML)、repository secret `GH_INFRA_TOKEN` (読み取り専用 fine-grained PAT)
- Produces: ワークフロー `GH Infra Plan` (失敗時 issue 起票、成功時自動クローズ)

- [ ] **Step 1: ワークフローファイルを作成**

`.github/workflows/gh-infra-plan.yml` に以下をそのまま書く:

```yaml
name: GH Infra Plan

# 週次で gh-infra の宣言状態 (gh-infra/y-maeda1116/) と GitHub 実状態の
# ドリフトを検出する。失敗時はissueを起票し、次回成功時に自動で閉じる。
# トークンは読み取り専用 fine-grained PAT (GH_INFRA_TOKEN)。

on:
  schedule:
    - cron: '43 6 * * 1' # 毎週月曜 06:43 UTC (scheduled-security-audit と時間をずらす)
  workflow_dispatch:

permissions:
  contents: read

concurrency:
  group: gh-infra-plan
  cancel-in-progress: false

jobs:
  plan:
    name: Drift check (gh infra plan --ci)
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1

      # 未登録なら以降のステップが不透明なエラーで失敗するため、最初に fail-fast。
      # secret は env 経由で渡し、${{ }} 展開を run: の本文に埋め込まない。
      - name: Verify GH_INFRA_TOKEN is set
        env:
          GH_INFRA_TOKEN: ${{ secrets.GH_INFRA_TOKEN }}
        run: |
          if [ -z "$GH_INFRA_TOKEN" ]; then
            echo "Error: GH_INFRA_TOKEN secret is not configured."
            echo "Set it in Settings -> Secrets and variables -> Actions."
            exit 1
          fi

      - name: Install gh-infra extension (pinned)
        env:
          GH_TOKEN: ${{ github.token }}
        run: gh extension install babarot/gh-infra --tag v0.13.0

      # 差分があれば exit 1 (--ci)。ドリフトと実行エラー (トークン期限切れ等) は
      # 区別せず、どちらも「対応が必要」として同じフローで拾う。
      - name: Run drift check
        env:
          GH_TOKEN: ${{ secrets.GH_INFRA_TOKEN }}
        run: gh infra plan gh-infra/y-maeda1116/ --ci

  notify-on-failure:
    name: Open issue on failure
    needs: [plan]
    if: ${{ always() && contains(needs.*.result, 'failure') }}
    runs-on: ubuntu-latest
    permissions:
      issues: write
    steps:
      - name: Create or update tracking issue
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GH_REPO: ${{ github.repository }}
          PLAN_RESULT: ${{ needs.plan.result }}
          RUN_URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}
        run: |
          set -euo pipefail
          LABEL="gh-infra-drift"
          gh label create "$LABEL" --description "gh-infra drift check failed" --color D73A4A 2>/dev/null || true
          EXISTING=$(gh issue list --label "$LABEL" --state open --json number --jq '.[0].number // empty')
          BODY="gh infra plan reported differences (or failed).

          Results:
          - plan: ${PLAN_RESULT}

          Check the run log for the diff: ${RUN_URL}
          Either update gh-infra/y-maeda1116/ or import the GitHub-side change.
          The next green scheduled run will close this issue."
          if [ -z "$EXISTING" ]; then
            gh issue create --title "Repository settings drift detected (gh-infra)" --label "$LABEL" --body "$BODY"
          else
            gh issue comment "$EXISTING" --body "Still failing.

          $BODY"
          fi

  close-on-success:
    name: Close resolved issue on success
    needs: [plan]
    if: ${{ always() && !contains(needs.*.result, 'failure') }}
    runs-on: ubuntu-latest
    permissions:
      issues: write
    steps:
      - name: Close tracking issue if open
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GH_REPO: ${{ github.repository }}
        run: |
          set -euo pipefail
          LABEL="gh-infra-drift"
          for num in $(gh issue list --label "$LABEL" --state open --json number --jq '.[].number'); do
            gh issue comment "$num" --body "gh infra plan is green. Closing."
            gh issue close "$num"
          done
```

- [ ] **Step 2: YAML 構文を検証**

Run: `uv run python -c "import yaml; yaml.safe_load(open('.github/workflows/gh-infra-plan.yml')); print('YAML OK')"`
Expected: `YAML OK`

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/gh-infra-plan.yml
git commit -m "ci: add weekly gh-infra drift detection workflow"
```

### Task 5: apply-security.sh からブランチ保護を削除

**Files:**
- Modify: `scripts/apply-security.sh`

**Interfaces:**
- Consumes: なし (単独)
- Produces: 4 ステップ版のスクリプト (ブランチ保護は gh-infra rulesets 側で管理)

- [ ] **Step 1: スクリプトを修正**

変更点は3つ。(a) usage にブランチ保護の移管先を明記、(b) `[4/5]` ブランチ保護ステップを削除、(c) ステップ番号を 5→4 に振り直し、検証から branch protection を削除。

`usage()` の heredoc を以下に置き換え:

```bash
usage() {
  cat <<'EOF'
Usage: apply-security.sh <repo>

Apply security settings to a GitHub repository.
Branch protection is managed by gh-infra rulesets (gh-infra/y-maeda1116/).

Arguments:
  repo    Repository in owner/repo format (e.g. myorg/myproject)

Required environment:
  GITHUB_TOKEN  GitHub personal access token with repo/admin scope

Example:
  GITHUB_TOKEN=ghp_xxx ./apply-security.sh myorg/myproject
EOF
  exit 1
}
```

`[1/5]` → `[1/4]`、`[2/5]` → `[2/4]`、`[3/5]` → `[3/4]` に文言変更。

`# Enable branch protection on main` のブロック (`echo "[4/5] ..."` から `PAYLOAD` まで) を丸ごと削除。

`# Verify settings` 以降を以下に置き換え (PROTECTION 取得・表示の削除と `[5/5]`→`[4/4]`):

```bash
# Verify settings
echo "[4/4] Verifying settings..."
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
echo ""
echo "Done."
```

- [ ] **Step 2: 構文検証**

Run: `bash -n scripts/apply-security.sh && echo "syntax OK"`
Expected: `syntax OK`

- [ ] **Step 3: Commit**

```bash
git add scripts/apply-security.sh
git commit -m "refactor: drop branch protection from apply-security.sh in favor of rulesets"
```

### Task 6: README を更新

**Files:**
- Modify: `README.md`

**Interfaces:**
- Consumes: Task 1-5 の成果物 (`gh-infra/`、`gh-infra-plan.yml`、スリム化した apply-security.sh)
- Produces: なし (ドキュメント)

- [ ] **Step 1: 構成ツリーを更新**

`.github/workflows/` ブロック内の `security-audit-scheduled.yml` 行の後に追加:

```
│   │   ├── gh-infra-plan.yml              # 週次ドリフト検出 (gh infra plan --ci)
```

`├── configs/` ブロックの前に挿入:

```
├── gh-infra/                   # リポジトリ設定の宣言管理 (gh-infra)
│   ├── y-maeda1116/            # repos.yaml (共通) / security-base.yaml (個別)
│   └── targets.txt             # 管理対象リポジトリ一覧
```

- [ ] **Step 2: 機能表を更新**

`apply-security.sh` 行を以下に置き換え、その下に `GH Infra` 行を追加:

```markdown
| apply-security.sh | 共通 | 脆弱性アラート・脆弱性報告・シークレットスキャン+プッシュ保護の設定 |
| GH Infra | 共通 | リポジトリ設定をYAMLで宣言管理し、週次でドリフト検出 |
```

- [ ] **Step 3: gh-infra セクションを追加**

`## リポジトリ設定の自動適用` の見出しの直前に以下を挿入:

```markdown
## リポジトリ設定の宣言管理 (gh-infra)

y-maeda1116 配下のリポジトリ設定 (labels / features / merge strategy / rulesets /
actions 設定) を `gh-infra/y-maeda1116/` の YAML で宣言的に管理します。
ブランチ保護は classic branch protection ではなく rulesets で管理します
(security-base のみ必須CIチェック付き)。

```bash
# 新規リポジトリを管理対象に追加
./gh-infra/sync-repos.sh
gh infra plan gh-infra/y-maeda1116/    # 差分確認
gh infra apply gh-infra/y-maeda1116/   # 適用 (ローカル実行のみ)
```

- ドリフト検出: `GH Infra Plan` ワークフローが週次で `gh infra plan --ci` を実行し、
  差分があれば issue を起票 (解消で自動クローズ)
- 認証: repository secret `GH_INFRA_TOKEN` — 読み取り専用 fine-grained PAT
  (Administration: read)
- 適用 (`gh infra apply`) はローカル実行のみ。CI に書き込み権限のトークンは置かない
- 詳細: `gh-infra/README.md`
```

- [ ] **Step 4: 自動適用セクションからブランチ保護を削除**

`## リポジトリ設定の自動適用` の「適用される設定」リストから以下を削除:

```markdown
- `main` ブランチの保護設定:
  - 管理者にもルール適用 (`enforce_admins`)
  - ステータスチェック合格必須
  - フォースpush・ブランチ削除を禁止
```

- [ ] **Step 5: Commit**

```bash
git add README.md
git commit -m "docs: describe gh-infra integration in README"
```

### Task 7: security-base への適用と classic 保護の削除 (移行実行)

**⚠️ このタスクは GitHub の実状態を変更する (worktree の外の副作用)。実装者は実行しない — コントローラーが plan の差分をユーザーに提示し、明示的な承認を得てから実行する。**

**Files:**
- 変更なし (GitHub 側の設定変更のみ。コミットなし)

**Interfaces:**
- Consumes: Task 1-3 の YAML (`gh-infra/y-maeda1116/`)
- Produces: security-base に rulesets 適用済み・classic branch protection 削除済みの状態

- [ ] **Step 1: 全体の plan を実行して差分を確認**

Run: `gh infra plan gh-infra/y-maeda1116/`
Expected: 未適用リポへの rulesets 追加等の差分が表示される

- [ ] **Step 2: ユーザー承認 (HARD STOP)**

差分の要旨をユーザーに提示し、適用の承認を得る。

- [ ] **Step 3: security-base のみ適用**

Run: `gh infra apply gh-infra/y-maeda1116/ -r y-maeda1116/security-base`
Expected: rulesets `main` が適用される

- [ ] **Step 4: classic branch protection を削除**

Run: `gh api --method DELETE /repos/y-maeda1116/security-base/branches/main/protection`
Expected: 204。rulesets との併存解消。

- [ ] **Step 5: 適用結果を検証**

Run: `gh api repos/y-maeda1116/security-base/rulesets --jq '.[] | {name, enforcement, rules: [.rules[].type]}'`
Expected: ruleset `main` (active) に `non_fast_forward`, `deletion`, `required_status_checks` が含まれる

他 28 リポへの適用はマージ後のユーザー作業 (spec「ユーザー手作業」参照)。
