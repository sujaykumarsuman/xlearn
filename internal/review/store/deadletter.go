package store

import (
	"context"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/review/store/gen"
)

var _ events.DeadLetterSink = (*PgStore)(nil)

// RecordDeadLetter writes the ids-only dead-letter row for an event whose handler
// failed its last delivery (mi-05, ADR-0035 §1.2). Idempotent per (durable, event_id).
func (s *PgStore) RecordDeadLetter(ctx context.Context, dl events.DeadLetter) error {
	if err := s.q.InsertDeadLetter(ctx, gen.InsertDeadLetterParams{
		EventID:   dl.EventID,
		Subject:   dl.Subject,
		Durable:   dl.Durable,
		ErrClass:  dl.ErrClass,
		StreamSeq: streamSeqParam(dl.StreamSeq),
		At:        tsz(dl.At),
	}); err != nil {
		return fmt.Errorf("insert dead letter: %w", err)
	}
	return nil
}

// ListDeadLetters is the on-demand read of review.event_dead_letter, newest first
// (D34: dead letters are rows, read when the owner looks).
func (s *PgStore) ListDeadLetters(ctx context.Context, limit int32) ([]events.DeadLetter, error) {
	rows, err := s.q.ListDeadLetters(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list dead letters: %w", err)
	}
	out := make([]events.DeadLetter, 0, len(rows))
	for _, r := range rows {
		dl := events.DeadLetter{EventID: r.EventID, Subject: r.Subject, Durable: r.Durable, ErrClass: r.ErrClass, At: r.At.Time}
		if r.StreamSeq.Valid && r.StreamSeq.Int64 > 0 {
			dl.StreamSeq = uint64(r.StreamSeq.Int64)
		}
		out = append(out, dl)
	}
	return out, nil
}

// streamSeqParam maps the stream sequence to a nullable bigint (0 = unknown → NULL).
func streamSeqParam(seq uint64) pgtype.Int8 {
	if seq == 0 || seq > math.MaxInt64 {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: int64(seq), Valid: true}
}
