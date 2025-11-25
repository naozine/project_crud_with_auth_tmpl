-- name: ListProjects :many
SELECT * FROM projects ORDER BY created_at DESC;

-- name: CreateProject :one
INSERT INTO projects (name)
VALUES (?)
RETURNING *;

-- name: GetProject :one
SELECT * FROM projects WHERE id = ? LIMIT 1;

-- name: UpdateProject :one
UPDATE projects
SET name = ?
WHERE id = ?
RETURNING *;

-- name: DeleteProject :exec

DELETE FROM projects

WHERE id = ?;



-- name: GetUserByEmail :one

SELECT * FROM users WHERE email = ? LIMIT 1;



-- name: CreateUser :one

INSERT INTO users (email, name, role, is_active)

VALUES (?, ?, ?, ?)

RETURNING *;



-- name: UpdateUser :one

UPDATE users

SET name = ?, role = ?, is_active = ?, updated_at = CURRENT_TIMESTAMP

WHERE id = ?

RETURNING *;



-- name: ListUsers :many

SELECT * FROM users ORDER BY created_at DESC;



-- name: DeleteUser :exec

DELETE FROM users WHERE id = ?;
