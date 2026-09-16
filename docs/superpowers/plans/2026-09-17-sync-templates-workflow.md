# Sync Templates Workflow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** main マージ時に同期対象ファイルの変更をテンプレート3リポへ自動PRする `sync-templates` ワークフローを作成する。

**Architecture:** ワークフロー1ファイルの追加のみで、Go ツール (`tools/sync`) は無変更。認証は fine-grained PAT (`secrets.SYNC_TOKEN`) を `gh auth setup-git` (git push) と env `GITHUB_TOKEN` (PR 作成、既存 `GetToken()`) の両方に供給する。

**Tech Stack:** GitHub Actions (ubuntu-latest), Go 1.26.x, gh CLI (ランナー標準搭載)

**Spec:** `docs/superpowers/specs/2026-09-16-sync-templates-workflow-design.md`

## Global Constraints

- `uses:` はコミットSHAピン留め必須 (CLAUDE.md の必須ルール。タグ参照は禁止)
- コミットメッセージは Conventional Commits (`ci:`, `docs:` 等)
- paths フィルタの対象はスペックの表と完全一致 (同期対象ファイル + `tools/sync/**`)
- Go コード (`tools/sync/**/*.go`) は変更しない
- ワークフロー内で `github.event.*` 等の信頼できない入力を `run:` に埋め込まない

---

### Task 1: sync-templates ワークフローを作成

**Files:**
- Create: `.github/workflows/sync-templates.yml`

**Interfaces:**
- Consumes: `tools/sync` の CLI (`go run . -config config.yaml`、env `GITHUB_TOKEN` を参照する `GetToken()`)
- Produces: ワークフロー `Sync Templates` (secret `SYNC_TOKEN` を要求。未登録の場合 Run sync ステップが失敗する)

- [ ] **Step 1: ワークフローファイルを作成**

`.github/workflows/sync-templates.yml` に以下をそのまま書く:

```yaml
name: Sync Templates

# main マージ後に同期対象ファイルへ変更があれば、テンプレート3リポへ自動PR。
# tools/sync 自体の変更時 (config.yaml の同期対象・配布先変更を含む) も発火。

on:
  push:
    branches: [main]
    paths:
      - '.github/workflows/reusable-go-security.yml'
      - '.github/workflows/reusable-py-security.yml'
      - '.github/workflows/reusable-ts-security.yml'
      - '.github/workflows/reusable-secret-scan.yml'
      - '.github/workflows/ci.yml'
      - '.github/dependabot.yml'
      - 'configs/**'
      - 'scripts/**'
      - 'SECURITY.md'
      - '.github/CODEOWNERS'
      - 'tools/sync/**'
  workflow_dispatch:

permissions:
  contents: read

concurrency:
  group: sync-templates
  cancel-in-progress: false

jobs:
  sync:
    name: Sync to template repositories
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1

      - name: Set up Go
        uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e
        with:
          go-version: 1.26.x
          # 1.26.x のようなパッチ範囲指定で、ランナーのツールキャッシュに古い
          # パッチがあればそれを使ってしまうのを防ぐ (reusable-go-security と同様)。
          check-latest: true

      # CI ランナーには git のコミット設定がなく、tools/sync の git commit が
      # 失敗するため必須。
      - name: Configure git identity
        run: |
          git config --global user.name "security-base-sync"
          git config --global user.email "security-base-sync@users.noreply.github.com"

      # gh を git の credential helper にする。clone と push の認証をカバーし、
      # トークンがリモートURLやログに露出しない。SYNC_TOKEN は fine-grained PAT
      # (対象3リポのみ、Contents / Pull requests / Workflows の RW 権限)。
      - name: Set up git credential helper
        env:
          GH_TOKEN: ${{ secrets.SYNC_TOKEN }}
        run: gh auth setup-git

      # GITHUB_TOKEN は既存 GetToken() が参照 (PR 作成用)。
      # GH_TOKEN は git push 時に credential helper から呼ばれる gh が参照。
      - name: Run sync
        env:
          GITHUB_TOKEN: ${{ secrets.SYNC_TOKEN }}
          GH_TOKEN: ${{ secrets.SYNC_TOKEN }}
        working-directory: tools/sync
        run: go run . -config config.yaml
```

- [ ] **Step 2: YAML 構文を検証**

Run: `uv run python -c "import yaml; yaml.safe_load(open('.github/workflows/sync-templates.yml')); print('YAML OK')"`
Expected: `YAML OK`

- [ ] **Step 3: 既存テストの回帰確認 (コード無変更のため影響なしの確認)**

Run: `cd tools/sync && go test ./...`
Expected: `ok github.com/y-maeda1116/security-base/tools/sync`

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/sync-templates.yml
git commit -m "ci: add sync-templates workflow to auto-sync template repos"
```

### Task 2: README に自動同期の説明を追記

**Files:**
- Modify: `README.md` (「リポジトリ設定の自動適用」セクションの直前に挿入)

**Interfaces:**
- Consumes: Task 1 のワークフロー名 (`Sync Templates`) と secret 名 (`SYNC_TOKEN`)
- Produces: なし (ドキュメント)

- [ ] **Step 1: README にセクションを追記**

`## リポジトリ設定の自動適用` の見出しの直前に以下を挿入:

```markdown
## テンプレートリポジトリへの自動同期

main ブランチにマージされた変更のうち同期対象ファイル
(reusable workflows / `configs/` / `scripts/` / `SECURITY.md` / `CODEOWNERS` 等) に
該当するものがある場合、`Sync Templates` ワークフローが自動でテンプレートリポジトリ
(python-template-base / ts-template-base / template-go-cross) へ同期PRを作成します。

- ツール本体: `tools/sync` (Go)
- 設定: `tools/sync/config.yaml` (同期対象ファイルと配布先ターゲット)
- 認証: repository secret `SYNC_TOKEN` — fine-grained PAT
  (対象3リポのみ、Contents / Pull requests / Workflows の読み書き権限)
- 差分がない場合、または同名PRが既に開いている場合はスキップ
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: describe sync-templates workflow in README"
```
