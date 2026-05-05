# security-base — プロジェクトルール

GitHubリポジトリのセキュリティ設定を共通管理するリポジトリ。
他のGo/TypeScript/Pythonリポジトリから参照される「信頼の源泉」として機能する。

## 必須ルール

### GitHub Actions は SHA ピン留め必須

サプライチェーン攻撃を防ぐため、`uses:` にはタグ (`@v4`) ではなく **コミットSHA** (`@abc123...`) を使用する。

```yaml
# OK
uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd

# NG
uses: actions/checkout@v4
```

Dependabot が週次でSHA付きの更新PRを作成するため、タグ版を使う理由はない。

`go install` 等の `@latest` も再現性を損なうため避け、バージョンを明示的に指定する。

### Python ツールチェイン

- パッケージマネージャ: **uv** (`uv sync`, `uv run`)
- Lint/Format: **Ruff** (Sルール = flake8-bandit 有効)
- 型チェック: **mypy** strict モード
- セキュリティLint: **bandit** (設定は `pyproject.toml` の `[tool.bandit]`)
- 依存関係監査: **pip-audit**
- テスト: **pytest** + カバレッジ80%以上必須

### Go ツールチェイン (tools/sync)

- Lint: **golangci-lint v2** (gosec, errcheck, govet, staticcheck 等)
- 脆弱性チェック: **govulncheck** (バージョンを明示指定、`@latest` は禁止)

### TypeScript セキュリティ

- 監査: **npm audit --audit-level=high**
- Lint: **eslint-plugin-security** (設定は `configs/.eslintrc.base.json`)

## コミット規約

Conventional Commits を使用:

```
feat: <description>
fix: <description>
refactor: <description>
test: <description>
docs: <description>
chore: <description>
ci: <description>
build(deps): <description>   # Dependabot PR用
```

## 変更時の確認事項

- ワークフロー変更時: SHA ピン留めが正しいか確認
- `pyproject.toml` 変更時: `uv lock` を実行し `uv.lock` を更新
- `configs/` 変更時: `tools/sync/config.yaml` の同期対象に含まれているか確認
- 新しい reusable workflow 追加時: `dependabot.yml` と `tools/sync/config.yaml` に追加

## テスト

```bash
# Python
uv run pytest

# Go (tools/sync)
cd tools/sync && go test ./...
```

## ファイル構成

```
.github/workflows/    # 再利用可能ワークフロー (SHA ピン留め)
.github/dependabot.yml
configs/              # golangci-lint / ESLint 共通設定
scripts/apply-security.sh
src/                  # Python package
tests/                # Python tests
tools/sync/           # Go製 同期ツール
pyproject.toml        # Python 設定集約
```
