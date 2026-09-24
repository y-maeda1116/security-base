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
gh extension install babarot/gh-infra --pin v0.13.0
```

Dependabot は gh-infra 拡張を監視できないため、更新は plan ジョブのログや
手動確認で気付く運用 (バージョン固定のため、更新時はこの README とワークフローのタグを更新する)。

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
