-- identity.admin_audit (m1-02, M1a expand; ADR-0033 §8): the owner admin CLI's audit
-- log. m1-04's CLI is the writer; v1.6.0 only creates the table.

-- name: InsertAdminAudit :one
INSERT INTO identity.admin_audit (verb, target, detail)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListAdminAudit :many
-- Newest first, capped (the CLI's `audit` read).
SELECT * FROM identity.admin_audit
ORDER BY at DESC, id
LIMIT $1;
