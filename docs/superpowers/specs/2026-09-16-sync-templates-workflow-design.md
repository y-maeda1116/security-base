# 同期自動化ワークフロー設計 (sync-templates)

日付: 2026-09-16
ステータス: 承認済み (アプローチA: ワークフローだけで解決、Go コード無変更)

## 背景と目的

`tools/sync` は security-base のファイルをテンプレート3リポジトリ
(python-template-base / ts-template-base / template-go-cross) へ配布する Go ツールだが、
現状はローカルでの手動実行のみ。main へのマージ後に自動で同期PRを作成し、
「信頼の源泉」の配布パイプラインを閉じる。

## 要件

- main ブランチのマージで、同期対象ファイルに変更があった場合のみ発火
- 既存PRがある場合や差分がない場合はスキップ (既存 `tools/sync` の挙動に準拠)
- Go コード (`tools/sync`) は変更しない
- トークンは fine-grained PAT で対象3リポに限定

## 設計

### ワークフロー構成

`.github/workflows/sync-templates.yml` を新規作成する。

| 項目 | 内容 |
|------|------|
| トリガー | `push` (branches: `[main]` + paths フィルタ) / `workflow_dispatch` |
| paths | 同期対象ファイル (reusable-*.yml / ci.yml / dependabot.yml / configs/** / scripts/** / SECURITY.md / .github/CODEOWNERS) + `tools/sync/**` (ツール自体の変更) |
| permissions | `contents: read` のみ |
| concurrency | group: `sync-templates`, cancel-in-progress: false |
| ランナー | ubuntu-latest |

### ステップ

1. **Checkout** — `actions/checkout` (SHA ピン、既存と同じコミット)
2. **Set up Go** — `actions/setup-go` (SHA ピン、`1.26.x` + `check-latest: true`)
3. **git config** — `user.name: security-base-sync` / `user.email: security-base-sync@users.noreply.github.com` を `--global` に設定。CI ランナーには git のコミット設定がなく、これがないと `tools/sync` の `git commit` が失敗するため
4. **gh auth setup-git** — env `GH_TOKEN: secrets.SYNC_TOKEN`。git が gh を credential helper として使い、clone (private 対応) と push の認証をカバー。トークンはリモートURLやログに露出しない
5. **go run . -config config.yaml** — `working-directory: tools/sync`、env `GITHUB_TOKEN: secrets.SYNC_TOKEN` (既存 `GetToken()` が参照) + `GH_TOKEN`

### 認証 (fine-grained PAT)

- デフォルトの `GITHUB_TOKEN` は他リポジトリに push / PR 作成ができないため、fine-grained PAT を使用する
- 対象リポジトリ: python-template-base / ts-template-base / template-go-cross の3リポのみ
- 権限:
  - **Contents: Read and write** — ブランチ push
  - **Pull requests: Read and write** — PR 作成
  - **Workflows: Read and write** — 同期対象に `.github/workflows/*.yml` が含まれるため (workflow ファイルを含む push に必要)
- 保存先: security-base の repository secret `SYNC_TOKEN`

### エラー処理

- `tools/sync` の `main.go` はエラー時に exit 1 するため、失敗すればワークフローが赤くなる
- 失敗通知は GitHub 標準のワークフロー失敗通知 (メール) で拾う
- issue 起票 (security-audit-scheduled.yml のパターン) はスコープ外

### セキュリティ検討

- トリガーは保護された main への push のみ。`pull_request_target` を使わないため、外部からの入力はゼロ (インジェクションリスクなし)
- `SYNC_TOKEN` は fine-grained で対象3リポ・最小権限に限定。security-base 自体への権限を持たない
- 全 `uses:` はコミットSHA ピン (CLAUDE.md の必須ルール)
- Dependabot (`directory: "/"`) が新しいワークフロー内の actions も自動監視

## ユーザー手作業 (実装後)

1. fine-grained PAT を作成 (上記の対象・権限)
2. security-base の Settings → Secrets and variables → Actions に `SYNC_TOKEN` を登録
3. 手動 `workflow_dispatch` で初回動作確認 (対象リポにPRが作成されるか、または no-change skip になるか)

## テスト・検証

- ローカル: `cd tools/sync && go test ./...` (コード無変更のため影響なし、回帰確認のみ)
- YAML 構文検証
- 実機検証: push 後に `workflow_dispatch` で実行し、結果を確認

## スコープ外

- 失敗時の issue 自動起票
- マージ済み同期ブランチの自動削除 (現状 `tools/sync` はブランチを残す)
- Go コード側の変更 (トークン付きURL、GitHub App 対応等)
