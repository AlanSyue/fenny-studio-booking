package service

import (
	"context"
	"fmt"
	"time"

	_ "time/tzdata" // 內嵌時區資料庫，避免主機缺少 /usr/share/zoneinfo 時失效

	"github.com/jackc/pgx/v5"
)

// taipei is the timezone all inquiry numbers are dated by. The production
// server runs on UTC, so using the server date would put every inquiry
// submitted after 20:00 Taipei time on the wrong day.
var taipei = mustLoadTaipei()

func mustLoadTaipei() *time.Location {
	loc, err := time.LoadLocation("Asia/Taipei")
	if err != nil {
		// time/tzdata is embedded above, so this cannot happen in practice.
		panic(fmt.Sprintf("load Asia/Taipei: %v", err))
	}
	return loc
}

// FormatInquiryNo renders an inquiry number such as "FEN-20260805-001".
// Sequences beyond 999 widen rather than truncate.
func FormatInquiryNo(t time.Time, seq int) string {
	return fmt.Sprintf("FEN-%s-%03d", t.In(taipei).Format("20060102"), seq)
}

// InquiryNoQuerier is the subset of pgx used here, satisfied by both
// *pgxpool.Pool and pgx.Tx.
type InquiryNoQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// NextInquiryNo atomically reserves the next sequence number for the Taipei
// date of `now` and returns the formatted inquiry number. The INSERT ...
// ON CONFLICT DO UPDATE is atomic, so concurrent submissions can never
// receive the same sequence.
func NextInquiryNo(ctx context.Context, q InquiryNoQuerier, now time.Time) (string, error) {
	day := now.In(taipei).Format("2006-01-02")

	var seq int
	err := q.QueryRow(ctx,
		`INSERT INTO inquiry_daily_counters (day, last_seq)
		 VALUES ($1, 1)
		 ON CONFLICT (day) DO UPDATE SET last_seq = inquiry_daily_counters.last_seq + 1
		 RETURNING last_seq`,
		day,
	).Scan(&seq)
	if err != nil {
		return "", fmt.Errorf("reserve inquiry sequence: %w", err)
	}

	return FormatInquiryNo(now, seq), nil
}
