-- name: InsertTouchResult :exec
-- The durable auto-score record for one re-solve (R-SR2). auto_pass is computed by
-- the service (pattern < 2 min AND solved in-timer AND complexity stated); mock_mode
-- marks the stricter Day 21 / Day 45 touches (R-SR4).
INSERT INTO review.touch_result (
    revision_item_id, named_pattern_secs, solved_in_timer, stated_complexity, auto_pass, mock_mode
) VALUES ($1, $2, $3, $4, $5, $6);
