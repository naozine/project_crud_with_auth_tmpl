-- NOTE: Do not use Japanese in sqlc source files (causes code generation bugs)
-- NOTE: This file is read by sqlc only and is NEVER applied to the database.
--       When changing the schema, follow db/README.md: write a migration in
--       db/migrations/ AND keep this file in sync with the resulting schema.

CREATE TABLE IF NOT EXISTS projects (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
