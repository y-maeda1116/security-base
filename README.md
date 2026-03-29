# security-base

GitHubリポジトリのセキュリティ設定を共通管理するためのリポジトリです。
他のGo/TypeScriptリポジトリから呼び出される「信頼の源泉」として機能します。

## 構成

```
security-base/
├── .github/workflows/     # 再利用可能ワークフロー
│   ├── reusable-go-security.yml
│   ├── reusable-ts-security.yml
│   └── reusable-secret-scan.yml
├── configs/                # 共通Lint設定
│   ├── .golangci.yml
│   └── .eslintrc.base.json
├── scripts/                # 自動化スクリプト
│   └── apply-security.sh
└── README.md
```

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
    uses: your-org/security-base/.github/workflows/reusable-go-security.yml@main
    with:
      go-version: "1.23"
      golangci-lint-version: "v1.62"
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
    uses: your-org/security-base/.github/workflows/reusable-ts-security.yml@main
    with:
      node-version: "20"
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
    uses: your-org/security-base/.github/workflows/reusable-secret-scan.yml@main
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
    uses: your-org/security-base/.github/workflows/reusable-secret-scan.yml@main
    with:
      scan-tool: "trivy"

  gitleaks-scan:
    uses: your-org/security-base/.github/workflows/reusable-secret-scan.yml@main
    with:
      scan-tool: "gitleaks"
```

## 共通設定ファイルの使い方

### Go (golangci-lint)

以下をプロジェクトルートに配置してください。

```yaml
# プロジェクトルートの .golangci.yml に import を追加
run:
  timeout: 5m

linters:
  enable:
    - gosec
    - errcheck
    - govet
    - staticcheck
    - unused
    - gosimple
    - ineffassign
    - typecheck
    - misspell
    - gofmt
    - goimports

linters-settings:
  gosec:
    severity: medium
    confidence: medium
  errcheck:
    check-type-assertions: true
    check-blank: true
```

または `security-base` リポジトリのファイルをそのままコピーして使用します。

```bash
curl -o .golangci.yml https://raw.githubusercontent.com/your-org/security-base/main/configs/.golangci.yml
```

### TypeScript (ESLint)

`eslint-plugin-security` をインストールし、ベース設定を extends に追加します。

```bash
npm install --save-dev eslint eslint-plugin-security
```

```jsonc
// .eslintrc.json
{
  "extends": [
    "./node_modules/your-org-security-base/configs/.eslintrc.base.json"
  ]
}
```

## リポジトリ設定の自動適用

```bash
# 環境変数を設定
export GITHUB_TOKEN="ghp_your_token"

# 対象リポジトリにセキュリティ設定を適用
./scripts/apply-security.sh your-org/your-repo
```

適用される設定:
- 脆弱性アラート (Dependabot alerts) の有効化
- `main` ブランチの保護設定:
  - プルリクエストレビュー必須 (1人以上)
  - 管理者にもルール適用
  - ステータスチェック合格必須
