#!/bin/bash
# targets.txt の改行コードを念のため修正
sed -i 's/\r$//' gh-infra/targets.txt

while read -r repo; do
  [[ -z "$repo" ]] && continue

  # $repo は "owner/repo-name" の形式
  # 出力先を "owner/repo-name.yaml" に設定
  filepath="gh-infra/${repo}.yaml"
  
  # ディレクトリ（owner部分）を取り出して作成
  owner_dir=$(dirname "$filepath")
  mkdir -p "$owner_dir"

  echo "Importing $repo -> $filepath ..."
  
  # 実行（標準エラーも混ぜて出力するとデバッグしやすいです）
  gh infra import "$repo" > "$filepath" 2>&1

  if [ -s "$filepath" ]; then
    echo "  ✓ Done."
  else
    echo "  ✗ Failed (Check $filepath for error logs)"
  fi
done < gh-infra/targets.txt
