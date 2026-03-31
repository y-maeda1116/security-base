# security-base

GitHubリポジトリのセキュリティ設定を共通管理するためのリポジトリです。
他のGo/TypeScriptリポジトリから呼び出される「信頼の源泉」として機能します。

## 構成

```
security-base/
├── .github/
│   ├── workflows/              # 再利用可能ワークフロー
│   │   ├── reusable-go-security.yml
│   │   ├── reusable-py-security.yml
│   │   ├── reusable-ts-security.yml
│   │   └── reusable-secret-scan.yml
│   └── dependabot.yml          # Dependabot version updates
├── configs/                    # 共通Lint設定
│   ├── .bandit.yml
│   ├── .golangci.yml
│   └── .eslintrc.base.json
├── scripts/                    # 自動化スクリプト
│   └── apply-security.sh
└── README.md
```

## セキュリティ機能

| 機能 | 説明 |
|------|------|
| Reusable Go Security | golangci-lint (gosec, errcheck等) + govulncheck |
| Reusable Python Security | pip-audit (依存パッケージ脆弱性) + bandit (コードセキュリティLint) |
| Reusable TypeScript Security | npm audit + eslint-plugin-security |
| Reusable Secret Scan | Trivy または Gitleaks によるシークレット検出 |
| Dependabot | GitHub Actions の週次バージョンアップ自動更新 |
| apply-security.sh | 脆弱性アラート・ブランチ保護の一括設定 |

## 他リポジトリからの呼び出し方

呼び出し先のリポジトリで以下のように設定してください。

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

`security-base` の設定ファイルをコピーしてプロジェクトルートに配置してください。

```bash
curl -o .golangci.yml https://raw.githubusercontent.com/y-maeda1116/security-base/main/configs/.golangci.yml
```

有効な linter: gosec, errcheck, govet, staticcheck, unused, ineffassign

### Python (bandit)

`bandit` をインストールし、設定ファイルをプロジェクトルートに配置してください。

```bash
pip install bandit
curl -o .bandit.yml https://raw.githubusercontent.com/y-maeda1116/security-base/main/configs/.bandit.yml
```

MEDIUM 以上の深刻度を検出します。テスト用の `assert` (B101) はスキップされます。

### TypeScript (ESLint)

`eslint-plugin-security` をインストールし、ベース設定を extends に追加します。

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
