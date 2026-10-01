-- name: TakeDailyMessage :one
-- L18's durable daily cap (sprint m1-07 task 3, migration 00008): at most daily_cap coach
-- messages per account per UTC day. `day` is ALWAYS the UTC date computed in Go and passed
-- in (every query in this file), never current_date, so the cap does not depend on the session
-- TimeZone.
--
-- Claim one message of the account's day atomically. The first message of a day inserts
-- n = 1; later ones increment n only while it is below the cap. At the cap the DO UPDATE's
-- WHERE is false, so nothing is written and NO ROW comes back (pgx.ErrNoRows) — that is
-- the "cap reached" signal. Concurrent claims serialize on the row lock and re-check the
-- WHERE against the committed n, so they cannot overshoot (the store race test).
INSERT INTO coach.message_quota_day AS q (account_id, day, n)
VALUES (sqlc.arg(account_id), sqlc.arg(day), 1)
ON CONFLICT (account_id, day) DO UPDATE
SET n = q.n + 1
WHERE q.n < sqlc.arg(daily_cap)::int
RETURNING q.n;

-- name: RefundDailyMessage :exec
-- Give back one claimed message when the turn never reached the provider (a store or key
-- error after the claim), so a request that spent nothing counts nothing. Never below 0.
UPDATE coach.message_quota_day
SET n = n - 1
WHERE account_id = sqlc.arg(account_id) AND day = sqlc.arg(day) AND n > 0;

-- name: GetDailyMessages :one
-- Read-only: the account's messages so far on a UTC day, for GET /coach/admission (which
-- must consume nothing). No row means none yet.
SELECT n
FROM coach.message_quota_day
WHERE account_id = sqlc.arg(account_id) AND day = sqlc.arg(day);
