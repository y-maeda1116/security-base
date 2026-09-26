# security-base

[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/y-maeda1116/security-base/badge)](https://securityscorecards.dev/viewer/?uri=github.com/y-maeda1116/security-base)

GitHubリポジトリのセキュリティ設定を共通管理するためのリポジトリです。
他のGo/TypeScript/Pythonリポジトリから呼び出される「信頼の源泉」として機能します。

脆弱性の報告は [SECURITY.md](SECURITY.md) を参照してください。

## 構成

```
security-base/
├── .github/
│   ├── workflows/              # GitHub Actions ワークフロー
│   │   ├── ci.yml                        # Python CI (uv + ruff + mypy + pytest + pip-audit)
│   │   ├── reusable-go-security.yml
│   │   ├── reusable-py-security.yml
│   │   ├── reusable-ts-security.yml
│   │   ├── reusable-secret-scan.yml
│   │   ├── scorecard.yml                 # OpenSSF Scorecard 週次スコア化
│   │   ├── security-audit-scheduled.yml  # 週次セルフ監査
│   │   ├── gh-infra-plan.yml             # 週次ドリフト検出 (gh infra plan --ci)
│   │   └── sync-templates.yml            # テンプレートリポジトリへの自動同期PR
│   ├── CODEOWNERS              # コードオーナー定義
│   └── dependabot.yml          # Dependabot version updates
├── gh-infra/                   # リポジトリ設定の宣言管理 (gh-infra)
│   ├── y-maeda1116/            # repos.yaml (共通) / security-base.yaml (個別)
│   └── targets.txt             # 管理対象リポジトリ一覧
├── configs/                    # 共通Lint設定
│   ├── .golangci.yml
│   └── .eslintrc.base.json
├── scripts/                    # 自動化スクリプト
│   └── apply-security.sh
├── src/                        # Python package
├── tests/                      # Python tests
├── tools/
│   └── sync/                   # テンプレート同期ツール (Go)
├── pyproject.toml              # Python project configuration
├── SECURITY.md                 # セキュリティポリシー
└── README.md
```

## セキュリティ機能

| 機能 | 言語 | 説明 |
|------|------|------|
| Python CI | Python | uv + ruff (Sルール) + mypy (strict) + pytest + pip-audit |
| Reusable Go Security | Go | golangci-lint (gosec, errcheck等) + govulncheck |
| Reusable Python Security | Python | pip-audit + bandit (外部ファイル不要) |
| Reusable TypeScript Security | TypeScript | npm audit + eslint-plugin-security |
| Reusable Secret Scan | 共通 | Trivy または Gitleaks によるシークレット検出 |
| Dependabot | 共通 | GitHub Actions の週次バージョンアップ自動更新 |
| OpenSSF Scorecard | 共通 | セキュリティ姿勢を週次でスコア化し、公開API・バッジで確認 |
| Sync Templates | 共通 | main マージ後にテンプレートリポジトリへ自動同期PR |
| apply-security.sh | 共通 | 脆弱性アラート・脆弱性報告・シークレットスキャン+プッシュ保護の設定 |
| GH Infra | 共通 | リポジトリ設定をYAMLで宣言管理し、週次でドリフト検出 |

## Python プロジェクトテンプレート

このリポジトリは Python プロジェクトのセキュリティ重視テンプレートとしても機能します。

### 特徴

- **uv** による高速パッケージ管理 (`pyproject.toml` に集約)
- **Ruff** で Lint/Format + `S` (Security) ルール有効
- **mypy** strict モードで厳格な型チェック
- **bandit** 設定を `[tool.bandit]` セクションに集約 (外部ファイル不要)
- **pip-audit** で依存パッケージの脆弱性可視化
- **pytest** + カバレッジ80%必須

### CI パイプライン

```
uv sync → ruff check → ruff format → mypy → pytest → pip-audit
```

### 新規プロジェクトでの使い方

```bash
# 1. リポジトリをクローン
git clone https://github.com/y-maeda1116/security-base.git my-project
cd my-project

# 2. uv で依存関係をインストール
uv sync --group dev

# 3. 開発
uv run pytest
uv run ruff check .
uv run mypy src
```

## 他リポジトリからの呼び出し方 (Reusable Workflows)

reusable workflow は各リポジトリに配布したローカルコピー (`./.github/workflows/reusable-*.yml`) を参照する。
テンプレートリポジトリ (python-template-base / ts-template-base / template-go-cross) には tools/sync が自動配布する。
`y-maeda1116/security-base/...@main` のようなブランチ直接参照はピン留めされず、Scorecard の
Pinned-Dependencies チェックで減点されるため使わない。

### Goプロジェクトのセキュリティチェック

```yaml
# .github/workflows/go-security.yml
name: Go Security
on:
  pull_request:
  push:
    branches: [main]

jobs:
  go-security:
    uses: ./.github/workflows/reusable-go-security.yml
    with:
      go-version: "1.26"
      golangci-lint-version: "v2.11.4"
```

### Pythonプロジェクトのセキュリティチェック

```yaml
# .github/workflows/py-security.yml
name: Python Security
on:
  pull_request:
  push:
    branches: [main]

jobs:
  py-security:
    uses: ./.github/workflows/reusable-py-security.yml
    with:
      python-version: "3.13"
```

### TypeScriptプロジェクトのセキュリティチェック

```yaml
# .github/workflows/ts-security.yml
name: TypeScript Security
on:
  pull_request:
  push:
    branches: [main]

jobs:
  ts-security:
    uses: ./.github/workflows/reusable-ts-security.yml
    with:
      node-version: "24"
      package-manager: "npm"
```

### シークレットスキャン

```yaml
# .github/workflows/secret-scan.yml
name: Secret Scan
on:
  pull_request:
  push:
    branches: [main]

jobs:
  secret-scan:
    uses: ./.github/workflows/reusable-secret-scan.yml
    with:
      scan-tool: "trivy"
```

### TrivyとGitleaksを両方使う場合

```yaml
# .github/workflows/secret-scan.yml
name: Secret Scan
on:
  pull_request:
  push:
    branches: [main]

jobs:
  trivy-scan:
    uses: ./.github/workflows/reusable-secret-scan.yml
    with:
      scan-tool: "trivy"

  gitleaks-scan:
    uses: ./.github/workflows/reusable-secret-scan.yml
    with:
      scan-tool: "gitleaks"
```

## 共通設定ファイルの使い方

### Go (golangci-lint v2)

```bash
curl -o .golangci.yml https://raw.githubusercontent.com/y-maeda1116/security-base/main/configs/.golangci.yml
```

有効な linter: gosec, errcheck, govet, staticcheck, unused, ineffassign

### TypeScript (ESLint)

```bash
npm install --save-dev eslint eslint-plugin-security
```

```jsonc
// .eslintrc.json
{
  "extends": [
    "./node_modules/y-maeda1116-security-base/configs/.eslintrc.base.json"
  ]
}
```

### Python (bandit)

外部設定ファイルは不要です。`pyproject.toml` の `[tool.bandit]` セクションをコピーして使用してください。

## テンプレートリポジトリへの自動同期

main ブランチにマージされた変更のうち同期対象ファイル
(reusable workflows / `configs/` / `scripts/` / `SECURITY.md` / `CODEOWNERS` 等) に
該当するものがある場合、`Sync Templates` ワークフローが自動でテンプレートリポジトリ
(python-template-base / ts-template-base / template-go-cross) へ同期PRを作成します。

- ツール本体: `tools/sync` (Go)
- 設定: `tools/sync/config.yaml` (同期対象ファイルと配布先ターゲット)
- 認証: repository secret `SYNC_TOKEN` — fine-grained PAT
  (対象3リポのみ、Contents / Pull requests / Workflows の読み書き権限)
- 差分がない場合はスキップ (同一内容のPRが既に開いていても、新規ブランチの差分が
  空にならない限りPRが新規作成される点に注意)
- `dependabot.yml` は同期対象外。security-base 自身のものは gomod
  (`/tools/sync`) を含み他リポジトリでは不正な設定になるため、各リポジトリで
  言語エコシステム (pip / npm / gomod) に合わせて自己管理する

注: `SYNC_TOKEN` を登録するまで、マージ直後の初回実行は失敗します (想定動作)

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

## リポジトリ設定の自動適用

```bash
# gh 認証済みの場合は GITHUB_TOKEN は省略可能
./scripts/apply-security.sh y-maeda1116/your-repo
```

適用される設定:
- 脆弱性アラート (Dependabot alerts) の有効化
- プライベート脆弱性報告 (Private Vulnerability Reporting) の有効化
- シークレットスキャン + プッシュ保護 (public リポは無料 / private は GHAS 必須)
