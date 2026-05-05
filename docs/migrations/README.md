# Template Migrations

このディレクトリは、テンプレリポ (`project_crud_with_auth_tmpl`) で行った重要な改善のうち、**派生プロジェクトでも取り込みたいもの** をハウツー化した記録です。

## 背景

派生プロジェクトはテンプレから生成された後、独立して進化することを前提にしています ([README_TEMPLATE.md](../../README_TEMPLATE.md) の「テンプレートとの関係」参照)。`git merge` でテンプレ更新を取り込む運用は実態と合わなくなったため、**機能単位で AI に移植させる** 運用に切り替えました。

このディレクトリに置かれた各 `yyyy-mm-dd-X.md` は、**Claude Code に渡してそのまま実行できる** 単位のハウツーです。

## 派生プロジェクトでの使い方

派生プロジェクト側で Claude Code を起動し、以下のように指示します:

```
テンプレリポの docs/migrations/2026-05-01-quality-gates.md を読んで、
このプロジェクトにも同じ改善を入れてください。
```

Claude が:
1. テンプレリポのハウツーを参照
2. 派生プロジェクトの構造に合わせて適用
3. ローカル検証 + コミットまで実行

## 一覧

| 日付 | ファイル | 概要 |
|---|---|---|
| 2026-05-01 | [2026-05-01-quality-gates.md](./2026-05-01-quality-gates.md) | lint / CI / dependabot / `make check` 一式の導入 |
| 2026-05-01 | [2026-05-01-echo-to-chi.md](./2026-05-01-echo-to-chi.md) | Echo から chi/v5 への移行 (integration テストも含む) |
| 2026-05-02 | [2026-05-02-maxbody-protection.md](./2026-05-02-maxbody-protection.md) | G120 (DoS) 対策の `MaxBodySize` ミドルウェア導入 |
| 2026-05-05 | [2026-05-05-limits-package.md](./2026-05-05-limits-package.md) | body サイズ上限を `internal/limits` パッケージに集約 |
| 2026-05-05 | [2026-05-05-go-126-upgrade.md](./2026-05-05-go-126-upgrade.md) | Go 1.26 化 (sqlc 最新版の要求への追従) |
| 2026-05-05 | [2026-05-05-datastar-js-sync.md](./2026-05-05-datastar-js-sync.md) | Datastar JS のセルフホスト版を Go SDK と同期更新 |

## 書き方の方針

各ハウツーは以下の構造で書きます:

1. **Why**: なぜ必要だったか（実害があれば実害）
2. **What**: 何を変えたか（変更ファイル一覧）
3. **How**: どう実装したか（コードスニペット、コマンド）
4. **派生プロジェクトへの適用**: Claude Code への指示例
5. **検証**: 動作確認方法

派生プロジェクトの Claude が読んで自走できる程度の粒度。
