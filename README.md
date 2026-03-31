# security-base

GitHubリポジトリのセキュリティ設定を共通管理するためのリポジトリです。
他のGo/TypeScript/Pythonリポジトリから呼び出される「信頼の源泉」として機能します。

## 構成

```
security-base/
├── .github/
│   ├── workflows/              # 再利用可能ワークフロー
│   │   ├── ci.yml              # Python CI (uv + ruff + mypy + pytest + pip-audit)
│   │   ├── reusable-go-security.yml
│   │   ├── reusable-py-security.yml
│   │   ├── reusable-ts-security.yml
│   │   └── reusable-secret-scan.yml
│   └── dependabot.yml          # Dependabot version updates
├── configs/                    # 共通Lint設定
│   ├── .golangci.yml
│   └── .eslintrc.base.json
├── scripts/                    # 自動化スクリプト
│   └── apply-security.sh
├── src/                        # Python package
├── tests/                      # Python tests
├── pyproject.toml              # Python project configuration
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
| apply-security.sh | 共通 | 脆弱性アラート・ブランチ保護の一括設定 |

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
    uses: y-maeda1116/security-base/.github/workflows/reusable-go-security.yml@main
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
    uses: y-maeda1116/security-base/.github/workflows/reusable-py-security.yml@main
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
    uses: y-maeda1116/security-base/.github/workflows/reusable-ts-security.yml@main
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
    uses: y-maeda1116/security-base/.github/workflows/reusable-secret-scan.yml@main
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
    uses: y-maeda1116/security-base/.github/workflows/reusable-secret-scan.yml@main
    with:
      scan-tool: "trivy"

  gitleaks-scan:
    uses: y-maeda1116/security-base/.github/workflows/reusable-secret-scan.yml@main
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

## リポジトリ設定の自動適用

```bash
# gh 認証済みの場合は GITHUB_TOKEN は省略可能
./scripts/apply-security.sh y-maeda1116/your-repo
```

適用される設定:
- 脆弱性アラート (Dependabot alerts) の有効化
- `main` ブランチの保護設定:
  - 管理者にもルール適用 (`enforce_admins`)
  - ステータスチェック合格必須
  - フォースpush・ブランチ削除を禁止
