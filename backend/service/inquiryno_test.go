package service

import (
	"testing"
	"time"
)

func TestFormatInquiryNo(t *testing.T) {
	utc := time.UTC

	tests := []struct {
		name string
		when time.Time
		seq  int
		want string
	}{
		{
			name: "台北時間白天",
			when: time.Date(2026, 8, 5, 3, 52, 0, 0, utc), // 台北 11:52 同日
			seq:  1,
			want: "FEN-20260805-001",
		},
		{
			name: "UTC 晚間跨日到台北隔天",
			when: time.Date(2026, 8, 5, 16, 30, 0, 0, utc), // 台北 8/6 00:30
			seq:  1,
			want: "FEN-20260806-001",
		},
		{
			name: "台北時間剛好午夜",
			when: time.Date(2026, 8, 5, 16, 0, 0, 0, utc), // 台北 8/6 00:00
			seq:  7,
			want: "FEN-20260806-007",
		},
		{
			name: "流水號補零至三位",
			when: time.Date(2026, 8, 5, 3, 0, 0, 0, utc),
			seq:  42,
			want: "FEN-20260805-042",
		},
		{
			name: "超過三位不截斷",
			when: time.Date(2026, 8, 5, 3, 0, 0, 0, utc),
			seq:  1234,
			want: "FEN-20260805-1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatInquiryNo(tt.when, tt.seq)
			if got != tt.want {
				t.Errorf("FormatInquiryNo(%v, %d) = %q, want %q", tt.when, tt.seq, got, tt.want)
			}
		})
	}
}
