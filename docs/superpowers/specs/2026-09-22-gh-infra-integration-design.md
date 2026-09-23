# gh-infra 設定管理の security-base 統合設計

日付: 2026-09-22
ステータス: 承認済み (実装完了)

## 背景と目的

y-maeda1116 配下の GitHub リポジトリ設定は、現在 2 つの管理経路に分かれている:

1. **my-github-config** — gh-infra (`babarot/gh-infra`) で 16 リポの設定
   (labels / features / merge strategy / rulesets / actions 設定) を YAML で宣言管理
2. **security-base の apply-security.sh** — 脆弱性アラート・PVR・secret scanning・classic ブランチ保護を即時適用

この分断により以下の問題が発生している (2026-09-22 時点の実測):

- **ドリフト**: my-github-config の `security-base.yaml` は ruleset `main` を宣言しているが、
  security-base の実状態は classic branch protection のみ (rulesets は空)。最終同期は 2026-05-05。
- **未管理リポ**: 公開リポは現在 29 個、targets.txt は 16 個。**13 リポがいずれの管理下にもない**。
- **二重管理**: ブランチ保護が apply-security.sh (classic Branch Protection API) と
  gh-infra (Rulesets API) の両方に存在。GitHub は併存を推奨していない。

security-base を「信頼の源泉」として一本化し、リポ設定の IaC もここに集約する。

## 要件

- gh-infra の資材 (YAML・スクリプト) を security-base に移入する
- ブランチ保護を gh-infra rulesets に一本化し、apply-security.sh から当該ステップを削除する
- 週次で `gh infra plan` を CI 実行し、ドリフトを issue で検知する
- 書き込み (apply) はローカル手動実行のまま — CI に書き込み権限の PAT を置かない
- 新規リポは `sync-repos.sh` で検出・追加する

## 設計

### ディレクトリ構成

```
security-base/
  gh-infra/                    # 新設。tools/sync/config.yaml の同期対象に含めない
    README.md                  # 運用ガイド (my-github-config の README を移入先パスに合わせて改変)
    targets.txt
    sync-repos.sh              # 内部パス (y-maeda1116/ → gh-infra/y-maeda1116/) を修正
    get-repo-list.sh           # 同上
    gh-infra-import.sh         # 同上
    y-maeda1116/
      repos.yaml               # RepositorySet (defaults + 29 リポ)
      security-base.yaml       # security-base 個別 (standalone Repository)
```

- **archive/ は移入しない**: 初回 import 成果物の参照用。アーカイブ済みリポの YAML が
  plan/validate の対象に残ると差分ノイズになる。履歴は my-github-config リポに残る。
- `gh-infra/**` を `tools/sync/config.yaml` の files に追加しないため、
  テンプレート 3 リポへ誤配布されることはない。

### rulesets (ブランチ保護) の一本化

gh-infra v0.13.0 の rulesets スキーマで必要な機能がすべて表現可能:

| 現行 apply-security.sh [4/5] | gh-infra rulesets での表現 |
|---|---|
| `enforce_admins: true` | `bypass_actors` を指定しない (デフォルトで管理者にも適用) |
| `allow_force_pushes: false` | `non_fast_forward: true` |
| `allow_deletions: false` | `deletion: true` |
| `required_status_checks.contexts` | `required_status_checks.contexts[]` |
| `dismiss_stale_reviews` (count 0) | 対応する機能なし → 削除 (review count 0 では効果がないため) |

**security-base.yaml の ruleset は以下に強化する** (現行 classic 適用分 "Python CI" を引き継ぎつつ
go-test を追加):

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

- チェック名は check run 名 (= ジョブの `name:`) を指定する。実機で plan → apply →
  PR マージ検証まで確認する (検証事項)。
- **repos.yaml の defaults には required_status_checks を入れない**: リポ毎に存在する
  チェック名が異なり、存在しないチェックを要求するとそのリポはマージ不能になるため。
  defaults は現状どおり `non_fast_forward` / `deletion` のみ。
- reconcile モードはデフォルト (additive) を維持: YAML に無い ruleset を勝手に削除しない。

### apply-security.sh のスリム化

- [4/5] ブランチ保護ステップと、[5/5] 検証の branch protection 部分を削除
- 5 ステップ → 4 ステップ (脆弱性アラート / PVR / secret scanning+push protection / 検証)
- 役割は「gh-infra が管理しないセキュリティ機能の即時適用」に絞る
  (gh-infra v0.13.0 は vulnerability alerts / PVR / secret scanning を schema 管理対象外)
- usage・README の文言を更新

### 定期ドリフト検出ワークフロー

`.github/workflows/gh-infra-plan.yml` を新規作成:

| 項目 | 内容 |
|---|---|
| トリガー | 週次 cron (既存 audit の `17 6 * * 1` と時間をずらす: `43 6 * * 1`) + `workflow_dispatch` |
| permissions | `contents: read` (ジョブ全体) / `issues: write` (issue 起票ジョブのみ) |
| concurrency | group: `gh-infra-plan`, cancel-in-progress: false |

ジョブ構成 (security-audit-scheduled.yml のパターンを踏襲):

1. **plan** — checkout → `gh extension install babarot/gh-infra --tag v0.13.0` →
   `gh infra plan gh-infra/y-maeda1116/ --ci` (ドリフトあれば exit 1)
2. **notify-on-failure** — plan ジョブ失敗時 (ドリフト or 実行エラー) に issue 起票。
   `gh-infra-drift` ラベル、起票済みならコメント追記、run URL を本文に記載
3. **close-on-success** — green で当該 issue を自動クローズ

- `--ci` フラグにより「差分あれば非ゼロ終了」が保証されるため、出力のパースは不要。
  差分の詳細は run ログで確認する運用。
- 失敗要因の区別 (ドリフトか、トークン期限切れ等の実行エラーか) は issue では分類せず、
  どちらも「plan が green でない = 対応が必要」として同じフローで拾う。

### トークン設計

- **`GH_INFRA_TOKEN`** (新規 repository secret): 読み取り専用 fine-grained PAT
  - 対象: y-maeda1116 の全公開リポ (All repositories でも可、ただし限定推奨)
  - 権限: **Administration: read** + Metadata: read (自動付与)。
    rulesets / labels / actions permissions 等の読み取りをカバーする想定。
    不足があれば初回実行時のエラーで判別して追加する (検証事項)。
  - 有効期限を設定 (90 日推奨)。期限切れは plan ジョブの失敗 = issue で検知され、
    更新運用のトリガーになる。
- **apply (書き込み) はローカル実行**: ワークフローには書き込み PAT を置かない。
  ローカルでは `gh auth` のユーザートークンを使用 (管理者権限で apply する)。
- 既存 `SYNC_TOKEN` とは独立 (対象リポ・権限・用途が異なるため)。

### 移行手順 (ブランチ上で実施)

1. `gh-infra/` に資材移入 (スクリプトのパス修正を含む)
2. `sync-repos.sh` を実行し repos.yaml / targets.txt を 29 リポに最新化
3. `security-base.yaml` に required_status_checks を追加
4. ローカルで `gh infra validate` → `gh infra plan` → 差分が意図通りか確認
5. `gh infra apply` を security-base のみ (`-r`) に実行し、rulesets 適用
6. **classic branch protection の削除** (security-base のみ、API で削除):
   rulesets との併存解消。適用は plan 確認後に実施
7. apply-security.sh スリム化・ワークフロー追加・README 更新をコミット

他 28 リポへの rulesets 適用と classic protection 削除は、マージ後にユーザーが
`gh infra apply` (全体) で実行する (ユーザー手作業)。

### 運用フロー (移行後)

新規リポ作成時:

```bash
./gh-infra/sync-repos.sh                              # 検出して repos.yaml / targets.txt に追加
gh infra plan gh-infra/y-maeda1116/                   # 差分確認
gh infra apply gh-infra/y-maeda1116/                  # 共通設定を適用
./scripts/apply-security.sh y-maeda1116/<new-repo>    # 脆弱性系を適用
git add gh-infra && git commit && git push            # 管理状態を security-base に記録
```

定期 (毎週月曜): plan CI がドリフトを検知 → issue → YAML を直すか GitHub 側の変更を
import して反映 → 解消で issue 自動クローズ。

## セキュリティ検討

- 読み取り専用・期限付き・最小権限の fine-grained PAT。CI に置くトークンはこの 1 本のみ追加
- ワークフローの `uses:` はコミット SHA ピン (checkout のみ)
- `gh extension install` は `--tag v0.13.0` でバージョン固定。Dependabot は拡張を監視
  できないため、更新は plan ジョブのログ/手動確認で気付く運用 (README に記載)
- ドリフト検出は public リポのみ。private リポは GitHub Free で rulesets が使えない
  (gh-infra の制約) ため、将来 private リポを追加する場合は classic branch_protection
  (gh-infra の別機能) での検討が必要

## テスト・検証

- `gh infra validate gh-infra/y-maeda1116/` (ローカル)
- `gh infra plan` の差分が意図 (security-base の rulesets 追加 + 未管理 13 リポの追加) どおりか
- apply 後: security-base の PR で "Python CI" / "Go Test (tools/sync)" が必須チェックとして
 機能すること (実 PR で確認)
- ワークフロー YAML の構文検証
- `GH_INFRA_TOKEN` 登録後に `workflow_dispatch` で初回 green 確認 (ユーザー手作業)

## ユーザー手作業 (マージ後)

1. 読み取り専用 fine-grained PAT を作成し、repository secret `GH_INFRA_TOKEN` に登録
2. `gh infra apply gh-infra/y-maeda1116/` を全体に実行 (29 リポへ rulesets 適用)
3. 各リポの classic branch protection を削除 (必要なリポのみ。API または UI)
4. `gh-infra-plan` を `workflow_dispatch` で初回実行し green 確認
5. my-github-config リポに移転通知を push してアーカイブ (任意)

## スコープ外

- 全リポへの required_status_checks 展開 (チェック名がリポ毎に異なるため個別対応。将来)
- rulesets の reconcile: authoritative 化 (YAML に無い ruleset の削除管理)
- apply の CI 化・ドリフトの自動修正
- 新規 13 リポの個別設定チューニング (defaults の共通設定が適用されるのみ)
- private リポの扱い
