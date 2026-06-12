# db/ — スキーマ管理ガイド

スキーマを変更するとき（テーブル・カラムの追加/変更/削除）は、**必ずこのファイルの
手順に従うこと**。このディレクトリには役割の異なる2系統のファイルがあり、
両方を手で同期させる必要がある（自動検証はない。片方だけの変更は事故になる）。

## 2系統の役割

| ファイル | 読む人 | 役割 |
|---|---|---|
| `migrations/*.sql` | goose | **実際に DB へ適用される正**。アプリ起動時に `goose.Up` が自動実行（`cmd/server/main.go`）。手動の `goose up` は不要 |
| `schema.sql` / `schema_business.sql` | sqlc | クエリの型チェックとコード生成のための**完成形スキーマの参照資料**。DB には一切適用されない |
| `query.sql` / `query_business.sql` | sqlc | クエリ定義。`_business` は業務側（派生プロジェクトで書き換える側） |

片方だけ変更したときの壊れ方:

- **migration のみ** → sqlc が新カラムを知らず、クエリが生成できない（コンパイル時に気づける）
- **schema のみ** → コンパイルは通るが、実 DB にカラムがなく**実行時に no such column で死ぬ**（こちらが危険）

## テーブル・カラムを追加する手順

1. `make migrate-new NAME=add_xxx` で `migrations/` に雛形を作成
2. 生成されたファイルに `-- +goose Up` / `-- +goose Down` の SQL を書く
3. `schema_business.sql`（コア側なら `schema.sql`）を**同じ最終形**に更新する
   - sqlc ソース（schema/query）には日本語コメントを書かない（生成バグの原因）
4. 必要なら `query_business.sql` にクエリを追加
5. `make generate` で sqlc を再生成
6. アプリ起動（`make dev`）で migration が自動適用されることを確認
7. `make check`

migration ファイルは `db/fs.go` で embed されるため、ファイルを置くだけでビルドに含まれる。

## カラムの変更・削除（テーブル再作成の儀式）

SQLite の `ALTER TABLE` は列の型・制約の変更ができないため、テーブルの作り直しが
必要になる。**順序が重要**で、正しい順は「新テーブルを別名で作る → コピー →
旧を DROP → 新をリネーム」:

```sql
-- +goose NO TRANSACTION
-- PRAGMA foreign_keys はトランザクション内では変更できないため NO TRANSACTION にし、
-- 自前で BEGIN/COMMIT する。
-- +goose Up
PRAGMA foreign_keys = OFF;
BEGIN;
CREATE TABLE projects_new (
    -- 新しい列構成
);
INSERT INTO projects_new (id, name, created_at)
    SELECT id, name, created_at FROM projects;
DROP TABLE projects;
ALTER TABLE projects_new RENAME TO projects;
-- インデックス・トリガーがあればここで再作成
PRAGMA foreign_key_check;  -- 結果が空であること
COMMIT;
PRAGMA foreign_keys = ON;
```

### 罠: 旧テーブルを先にリネームしない（issue #38）

「`ALTER TABLE projects RENAME TO projects_old` → 新規作成 → `DROP projects_old`」
の順は**使わないこと**。SQLite はリネーム時に、参照元テーブルの FK 定義を
リネーム後の名前（`projects_old`）に自動で書き換えるため、その後 DROP すると
参照元の FK が存在しないテーブルを指したまま残り、以後の INSERT が FK エラーになる。

上記の「新を別名で作って最後にリネーム」の順なら、参照元の FK 定義は元の名前の
ままなので壊れない。すでに参照が壊れた場合や参照元も列変更したい場合は、
参照元テーブルも同じ儀式で作り直す（詳細は issue #38）。

## 関連

- `docs/migrations/` は別物（テンプレート改善を派生プロジェクトへ移植するためのガイド）
- スキーマ変更を含む改善をテンプレートから取り込む場合も、この同期ルールに従うこと
