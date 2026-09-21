package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sujaykumarsuman/xlearn/internal/review/store/gen"
)

// This file is the mistake-journal / weak-area / notifications half of the review
// store (S07). The open/close/re-open state machine lives in the tx helpers below and
// is driven from HandleProblemSolved (below-clean) and Score (clean revisit / failed
// re-solve) in store.go, so a journal state change and its scheduler state change
// always commit together (ADR-0016).

// Mistake is one journal entry (R-MJ1). Category "" means uncategorised (the learner
// hasn't picked from the seeded 8-value picker yet). RevisitDate is the zero time when
// unset (a below-clean entry with no scheduled ladder).
type Mistake struct {
	ID           string
	AccountID    string
	ProblemID    string
	Pattern      string
	Mistake      string
	RootCause    string
	Insight      string
	Category     string
	Status       string
	RevisitCount int
	RevisitDate  time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// MistakeInput is a manual create (POST /mistakes).
type MistakeInput struct {
	ProblemID string
	Pattern   string
	Mistake   string
	RootCause string
	Insight   string
	Category  string // "" = uncategorised
	Status    string // "" defaults to open
}

// MistakePatch is a partial edit (PATCH /mistakes/{id}); nil fields are unchanged. A
// non-nil Category of "" clears the classification back to uncategorised.
type MistakePatch struct {
	Pattern      *string
	Mistake      *string
	RootCause    *string
	Insight      *string
	Category     *string
	Status       *string
	RevisitCount *int
}

// WeakArea is the current weekly weak-area (R-MJ3): the top category, its count, the
// full per-category map, and the supporting open entries.
type WeakArea struct {
	WeekOf      time.Time
	TopCategory string
	TopCount    int
	Counts      map[string]int
	Entries     []Mistake
}

// Reminder is one in-app notification row (v1 single channel).
type Reminder struct {
	ID          string
	AccountID   string
	Kind        string
	DueAt       time.Time
	DeliveredAt time.Time
	CreatedAt   time.Time
}

// --- state-machine tx helpers (called inside HandleProblemSolved / Score) ---

// lockJournal takes the per-(account, problem) advisory lock that serialises all
// journal mutations across the two entry points (the opener and the incrementer), so a
// concurrent open can't collide with a close→open re-open on the one-open-entry index.
func lockJournal(ctx context.Context, qtx *gen.Queries, accountID, problemID string) error {
	if err := qtx.LockMistakeJournal(ctx, gen.LockMistakeJournalParams{AccountID: accountID, ProblemID: problemID}); err != nil {
		return fmt.Errorf("lock mistake journal: %w", err)
	}
	return nil
}

// openMistakeTx opens a mistake for (account, problem) with a pre-resolved pattern (the
// resolver is a network call and must run OUTSIDE the transaction) and, when a row is
// actually inserted (not deduped by the one-open-entry index), appends a mistake_opened
// outbox row. Callers must hold the journal advisory lock (lockJournal). Returns whether
// a new entry was opened.
func (s *PgStore) openMistakeTx(ctx context.Context, qtx *gen.Queries, aid pgtype.UUID, accountID, problemID, pattern string, revisitDate pgtype.Timestamptz) (bool, error) {
	_, err := qtx.OpenMistake(ctx, gen.OpenMistakeParams{
		AccountID:   aid,
		ProblemID:   problemID,
		Pattern:     pattern,
		Category:    pgtype.Text{}, // uncategorised — the learner picks from the seeded picker
		RevisitDate: revisitDate,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil // an open entry already exists — no duplicate, no re-emit
	}
	if err != nil {
		return false, fmt.Errorf("open mistake: %w", err)
	}
	if eerr := emitMistakeOpened(ctx, qtx, accountID, problemID, ""); eerr != nil {
		return false, eerr
	}
	return true, nil
}

// recordCleanRevisitTx increments the open mistake's clean-revisit count for this
// problem; on the close transition it emits mistake_closed. A no-op when the problem
// has no open mistake (the common case — a clean re-solve of a never-missed problem).
func (s *PgStore) recordCleanRevisitTx(ctx context.Context, qtx *gen.Queries, aid pgtype.UUID, accountID, problemID string) error {
	if err := lockJournal(ctx, qtx, accountID, problemID); err != nil {
		return err
	}
	row, err := qtx.IncrementCleanRevisit(ctx, gen.IncrementCleanRevisitParams{AccountID: aid, ProblemID: problemID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // no open mistake for this problem
	}
	if err != nil {
		return fmt.Errorf("increment clean revisit: %w", err)
	}
	if row.Status == MistakeClosed {
		return emitMistakeClosed(ctx, qtx, accountID, uuidString(row.ID), row.ProblemID)
	}
	return nil
}

// recordFailedResolveTx applies the fail branch of the state machine (R-MJ4): re-open
// a closed entry (+ emit mistake_opened), reset an open entry's streak (no re-emit), or
// open a fresh entry (with the pre-resolved pattern) when none exists. revisitDate is
// the freshly-reset Day-1 due. It holds the journal advisory lock across the read +
// mutation so the closed→open re-open can't race a concurrent open into a unique
// violation that would roll back the whole failed re-solve.
func (s *PgStore) recordFailedResolveTx(ctx context.Context, qtx *gen.Queries, aid pgtype.UUID, accountID, problemID, pattern string, revisitDate pgtype.Timestamptz) error {
	if err := lockJournal(ctx, qtx, accountID, problemID); err != nil {
		return err
	}
	latest, err := qtx.GetLatestMistake(ctx, gen.GetLatestMistakeParams{AccountID: aid, ProblemID: problemID})
	switch {
	case err == nil && latest.Status == MistakeClosed:
		if _, rerr := qtx.ReopenMistake(ctx, gen.ReopenMistakeParams{ID: latest.ID, RevisitDate: revisitDate}); rerr != nil {
			return fmt.Errorf("reopen mistake: %w", rerr)
		}
		return emitMistakeOpened(ctx, qtx, accountID, problemID, textString(latest.Category))
	case err == nil && latest.Status == MistakeOpen:
		// A fail wipes the clean-revisit streak (a fail never counts toward the 2). The
		// entry is already open, so no mistake_opened is re-emitted.
		if _, rerr := qtx.ResetCleanRevisitCount(ctx, gen.ResetCleanRevisitCountParams{
			AccountID: aid, ProblemID: problemID, RevisitDate: revisitDate,
		}); rerr != nil {
			return fmt.Errorf("reset revisit count: %w", rerr)
		}
		return nil
	case errors.Is(err, pgx.ErrNoRows):
		_, oerr := s.openMistakeTx(ctx, qtx, aid, accountID, problemID, pattern, revisitDate)
		return oerr
	default:
		return fmt.Errorf("get latest mistake: %w", err)
	}
}

// --- journal CRUD (gateway API) ---

// ListMistakes returns the account's journal (status "" = all).
func (s *PgStore) ListMistakes(ctx context.Context, accountID, status string) ([]Mistake, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return nil, ErrNotFound
	}
	var rows []gen.ReviewMistakeEntry
	if status == "" {
		rows, err = s.q.ListMistakes(ctx, aid)
	} else {
		rows, err = s.q.ListMistakesByStatus(ctx, gen.ListMistakesByStatusParams{AccountID: aid, Status: status})
	}
	if err != nil {
		return nil, fmt.Errorf("list mistakes: %w", err)
	}
	out := make([]Mistake, 0, len(rows))
	for _, r := range rows {
		out = append(out, toMistake(r))
	}
	return out, nil
}

// GetMistake returns one entry scoped to its owner.
func (s *PgStore) GetMistake(ctx context.Context, accountID, id string) (Mistake, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return Mistake{}, ErrNotFound
	}
	mid, err := parseUUID(id)
	if err != nil {
		return Mistake{}, ErrNotFound
	}
	row, err := s.q.GetMistake(ctx, gen.GetMistakeParams{ID: mid, AccountID: aid})
	if errors.Is(err, pgx.ErrNoRows) {
		return Mistake{}, ErrNotFound
	}
	if err != nil {
		return Mistake{}, fmt.Errorf("get mistake: %w", err)
	}
	return toMistake(row), nil
}

// CreateMistake manually creates a journal entry (POST /mistakes).
func (s *PgStore) CreateMistake(ctx context.Context, accountID string, in MistakeInput) (Mistake, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return Mistake{}, ErrNotFound
	}
	if in.Category != "" && !ValidMistakeCategory(in.Category) {
		return Mistake{}, ErrInvalidCategory
	}
	status := in.Status
	if status == "" {
		status = MistakeOpen
	}
	if status != MistakeOpen && status != MistakeClosed {
		return Mistake{}, ErrInvalidCategory // reuse for a bad status enum → 422
	}
	row, err := s.q.CreateMistake(ctx, gen.CreateMistakeParams{
		AccountID:    aid,
		ProblemID:    in.ProblemID,
		Pattern:      in.Pattern,
		Mistake:      in.Mistake,
		RootCause:    in.RootCause,
		Insight:      in.Insight,
		Category:     textOrNull(in.Category),
		RevisitDate:  pgtype.Timestamptz{},
		Status:       status,
		RevisitCount: 0,
	})
	if isUniqueViolation(err) {
		return Mistake{}, ErrConflict
	}
	if err != nil {
		return Mistake{}, fmt.Errorf("create mistake: %w", err)
	}
	return toMistake(row), nil
}

// UpdateMistake overlays a patch onto an entry (POST edit / PATCH). It reads the
// current row, applies the provided fields, and writes the full set.
func (s *PgStore) UpdateMistake(ctx context.Context, accountID, id string, in MistakePatch) (Mistake, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return Mistake{}, ErrNotFound
	}
	mid, err := parseUUID(id)
	if err != nil {
		return Mistake{}, ErrNotFound
	}
	cur, err := s.q.GetMistake(ctx, gen.GetMistakeParams{ID: mid, AccountID: aid})
	if errors.Is(err, pgx.ErrNoRows) {
		return Mistake{}, ErrNotFound
	}
	if err != nil {
		return Mistake{}, fmt.Errorf("get mistake: %w", err)
	}

	pattern, mistake, root, insight := cur.Pattern, cur.Mistake, cur.RootCause, cur.Insight
	category := cur.Category
	status := cur.Status
	revisitCount := cur.RevisitCount
	if in.Pattern != nil {
		pattern = *in.Pattern
	}
	if in.Mistake != nil {
		mistake = *in.Mistake
	}
	if in.RootCause != nil {
		root = *in.RootCause
	}
	if in.Insight != nil {
		insight = *in.Insight
	}
	if in.Category != nil {
		if *in.Category != "" && !ValidMistakeCategory(*in.Category) {
			return Mistake{}, ErrInvalidCategory
		}
		category = textOrNull(*in.Category)
	}
	if in.Status != nil {
		if *in.Status != MistakeOpen && *in.Status != MistakeClosed {
			return Mistake{}, ErrInvalidCategory
		}
		status = *in.Status
	}
	if in.RevisitCount != nil {
		if *in.RevisitCount < 0 {
			return Mistake{}, ErrInvalidCategory
		}
		revisitCount = int32(*in.RevisitCount)
	}

	row, err := s.q.UpdateMistake(ctx, gen.UpdateMistakeParams{
		ID:           mid,
		AccountID:    aid,
		Pattern:      pattern,
		Mistake:      mistake,
		RootCause:    root,
		Insight:      insight,
		Category:     category,
		Status:       status,
		RevisitCount: revisitCount,
		RevisitDate:  cur.RevisitDate,
	})
	if isUniqueViolation(err) {
		return Mistake{}, ErrConflict // re-opening while another open entry exists
	}
	if err != nil {
		return Mistake{}, fmt.Errorf("update mistake: %w", err)
	}
	return toMistake(row), nil
}

// --- weekly weak-area ---

// AccountsWithMistakes lists every account with at least one mistake entry.
func (s *PgStore) AccountsWithMistakes(ctx context.Context) ([]string, error) {
	ids, err := s.q.ListAccountsWithMistakes(ctx)
	if err != nil {
		return nil, fmt.Errorf("list accounts with mistakes: %w", err)
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, uuidString(id))
	}
	return out, nil
}

// CountOpenMistakesByCategory counts open entries opened within [start, end).
func (s *PgStore) CountOpenMistakesByCategory(ctx context.Context, accountID string, start, end time.Time) (map[string]int, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return nil, ErrNotFound
	}
	rows, err := s.q.CountOpenMistakesByCategoryInRange(ctx, gen.CountOpenMistakesByCategoryInRangeParams{
		AccountID:   aid,
		CreatedAt:   tsz(start),
		CreatedAt_2: tsz(end),
	})
	if err != nil {
		return nil, fmt.Errorf("count open mistakes by category: %w", err)
	}
	out := make(map[string]int, len(rows))
	for _, r := range rows {
		out[textString(r.Category)] = int(r.N)
	}
	return out, nil
}

// SaveWeakAreaSnapshot upserts one account's weekly snapshot (idempotent per week_of).
func (s *PgStore) SaveWeakAreaSnapshot(ctx context.Context, accountID string, weekOf time.Time, topCategory string, counts map[string]int) error {
	aid, err := parseUUID(accountID)
	if err != nil {
		return ErrNotFound
	}
	if counts == nil {
		counts = map[string]int{}
	}
	countsJSON, err := json.Marshal(counts)
	if err != nil {
		return fmt.Errorf("marshal counts: %w", err)
	}
	if _, err := s.q.UpsertWeakAreaSnapshot(ctx, gen.UpsertWeakAreaSnapshotParams{
		AccountID:   aid,
		WeekOf:      dateOf(weekOf),
		TopCategory: textOrNull(topCategory),
		CountsJson:  countsJSON,
	}); err != nil {
		return fmt.Errorf("upsert weak-area snapshot: %w", err)
	}
	return nil
}

// WeakAreaCurrent reads the account's latest snapshot + the supporting open entries in
// the top category. found is false when no snapshot exists yet.
func (s *PgStore) WeakAreaCurrent(ctx context.Context, accountID string) (WeakArea, bool, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return WeakArea{}, false, ErrNotFound
	}
	snap, err := s.q.GetLatestWeakArea(ctx, aid)
	if errors.Is(err, pgx.ErrNoRows) {
		return WeakArea{}, false, nil
	}
	if err != nil {
		return WeakArea{}, false, fmt.Errorf("get latest weak area: %w", err)
	}
	wa := WeakArea{
		WeekOf:      snap.WeekOf.Time,
		TopCategory: textString(snap.TopCategory),
		Counts:      map[string]int{},
	}
	if len(snap.CountsJson) > 0 {
		_ = json.Unmarshal(snap.CountsJson, &wa.Counts)
	}
	if wa.TopCategory != "" {
		wa.TopCount = wa.Counts[wa.TopCategory]
		ents, err := s.q.ListOpenMistakesByCategory(ctx, gen.ListOpenMistakesByCategoryParams{
			AccountID: aid,
			Category:  textOrNull(wa.TopCategory),
		})
		if err != nil {
			return WeakArea{}, false, fmt.Errorf("list supporting entries: %w", err)
		}
		wa.Entries = make([]Mistake, 0, len(ents))
		for _, e := range ents {
			wa.Entries = append(wa.Entries, toMistake(e))
		}
	}
	return wa, true, nil
}

// --- notifications ---

// HandleRevisionDue dedupes on eventID and writes one reminder, in one transaction.
func (s *PgStore) HandleRevisionDue(ctx context.Context, eventID, accountID, kind string, dueAt time.Time) (bool, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return false, fmt.Errorf("parse account id: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	fresh, err := claimInbox(ctx, qtx, eventID)
	if err != nil {
		return false, err
	}
	if !fresh {
		return false, nil // duplicate delivery
	}
	if _, err := qtx.InsertReminder(ctx, gen.InsertReminderParams{
		AccountID: aid,
		Kind:      kind,
		DueAt:     tsz(dueAt),
	}); err != nil {
		return false, fmt.Errorf("insert reminder: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit tx: %w", err)
	}
	return true, nil
}

// ListDueReminders returns the account's undelivered, now-due reminders.
func (s *PgStore) ListDueReminders(ctx context.Context, accountID string, limit int) ([]Reminder, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return nil, ErrNotFound
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.q.ListDueReminders(ctx, gen.ListDueRemindersParams{AccountID: aid, Limit: int32(limit)})
	if err != nil {
		return nil, fmt.Errorf("list due reminders: %w", err)
	}
	out := make([]Reminder, 0, len(rows))
	for _, r := range rows {
		out = append(out, Reminder{
			ID:          uuidString(r.ID),
			AccountID:   uuidString(r.AccountID),
			Kind:        r.Kind,
			DueAt:       r.DueAt.Time,
			DeliveredAt: r.DeliveredAt.Time,
			CreatedAt:   r.CreatedAt.Time,
		})
	}
	return out, nil
}

// --- event emitters ---

// emitMistakeOpened appends a xlearn.review.mistake_opened event (data: problem_id,
// category) to the outbox. category is "" for an auto-opened, not-yet-classified entry.
func emitMistakeOpened(ctx context.Context, qtx *gen.Queries, accountID, problemID, category string) error {
	return insertEvent(ctx, qtx, SubjectMistakeOpened, accountID, map[string]any{
		"problem_id": problemID,
		"category":   category,
	})
}

// emitMistakeClosed appends a xlearn.review.mistake_closed event (data: mistake_id,
// problem_id) to the outbox.
func emitMistakeClosed(ctx context.Context, qtx *gen.Queries, accountID, mistakeID, problemID string) error {
	return insertEvent(ctx, qtx, SubjectMistakeClosed, accountID, map[string]any{
		"mistake_id": mistakeID,
		"problem_id": problemID,
	})
}

// --- conversions + helpers ---

func toMistake(r gen.ReviewMistakeEntry) Mistake {
	return Mistake{
		ID:           uuidString(r.ID),
		AccountID:    uuidString(r.AccountID),
		ProblemID:    r.ProblemID,
		Pattern:      r.Pattern,
		Mistake:      r.Mistake,
		RootCause:    r.RootCause,
		Insight:      r.Insight,
		Category:     textString(r.Category),
		Status:       r.Status,
		RevisitCount: int(r.RevisitCount),
		RevisitDate:  r.RevisitDate.Time, // zero time when NULL (Valid=false)
		CreatedAt:    r.CreatedAt.Time,
		UpdatedAt:    r.UpdatedAt.Time,
	}
}

func textOrNull(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }

func textString(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}

func dateOf(t time.Time) pgtype.Date { return pgtype.Date{Time: t, Valid: true} }

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
