# 詢問編號與 LINE 導流 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 顧客送出團體手作課程詢問後，取得一組每日流水詢問編號（`FEN-20260805-001`），成功頁引導其加入 LINE 官方帳號並一鍵複製完整詢問摘要；店家可在後台以編號查到該筆詢問。

**Architecture:** 編號由 Postgres 的 `inquiry_daily_counters` 表以 `INSERT ... ON CONFLICT DO UPDATE ... RETURNING` 原子取號，在既有的建立詢問 transaction 內完成，因此並行送出不可能撞號。前端成功頁改為接住 API 回傳的完整 inquiry 物件，摘要組字與剪貼簿存取各自抽成獨立純函式模組。

**Tech Stack:** Go 1.25.5（chi + pgx/v5）、PostgreSQL、React 19 + Vite 8 + Tailwind 4、Vitest（本計畫新增）

## Global Constraints

- 編號格式：`FEN-` + `YYYYMMDD` + `-` + 流水號補零至 3 位（例：`FEN-20260805-001`）；超過 999 自然變 4 位，不截斷
- 編號日期一律以 **Asia/Taipei** 計算，絕不使用伺服器的 UTC 日期
- LINE 加好友連結預設值：`https://line.me/R/ti/p/@165nacbf`，可由環境變數 `VITE_LINE_URL` 覆蓋
- Telegram 通知維持非同步、失敗只寫 log，永不影響顧客送出結果
- migration 檔案為純 SQL，依檔名順序由 `deploy.sh` 自動套用，**已套用過的 migration 絕不可修改內容**
- 不實作 LINE push、LINE webhook、LINE 帳號綁定（見 spec 的「不做的事」）
- 所有面向顧客的文案使用繁體中文

## File Structure

**後端**
- `backend/db/migrations/005_inquiry_no.sql`（新增）— 加欄位、建計數表、回填既有資料、設 NOT NULL 與 UNIQUE
- `backend/service/inquiryno.go`（新增）— 編號格式化與取號，單一職責
- `backend/service/inquiryno_test.go`（新增）— 格式化的單元測試
- `backend/model/inquiry.go`（修改）— 加 `InquiryNo` 欄位
- `backend/handler/inquiry.go`（修改）— Create 取號；三處 SQL 加欄位
- `backend/service/telegram.go`（修改）— 訊息標題帶編號

**前端**
- `frontend/src/config.ts`（新增）— LINE 連結設定
- `frontend/src/lib/inquirySummary.ts`（新增）— 摘要組字純函式
- `frontend/src/lib/inquirySummary.test.ts`（新增）— 摘要組字測試
- `frontend/src/lib/clipboard.ts`（新增）— 剪貼簿寫入與降級
- `frontend/src/types/index.ts`（修改）— `Inquiry` 加 `inquiry_no`
- `frontend/src/pages/InquiryForm.tsx`（修改）— 成功頁改版
- `frontend/src/pages/InquiryList.tsx`（修改）— 編號欄與搜尋框
- `frontend/package.json`（修改）— 加 Vitest

摘要組字與剪貼簿拆成 `lib/` 下的獨立模組，是因為 `InquiryForm.tsx` 已達 328 行，且這兩段是與 UI 無關的純邏輯，獨立後才能單獨測試。

---

### Task 1: 資料庫 migration —— 編號欄位與每日計數表

**Files:**
- Create: `backend/db/migrations/005_inquiry_no.sql`

**Interfaces:**
- Consumes: 既有 `inquiries` 表（`id`、`created_at timestamptz`）
- Produces: `inquiries.inquiry_no VARCHAR(20) NOT NULL UNIQUE`；`inquiry_daily_counters(day DATE PRIMARY KEY, last_seq INTEGER NOT NULL)`

- [ ] **Step 1: 建立 migration 檔案**

建立 `backend/db/migrations/005_inquiry_no.sql`：

```sql
-- 1) 先加可為 NULL 的欄位，回填後再收緊約束
ALTER TABLE inquiries ADD COLUMN inquiry_no VARCHAR(20);

-- 2) 每日流水號計數表
CREATE TABLE inquiry_daily_counters (
    day      DATE PRIMARY KEY,
    last_seq INTEGER NOT NULL
);

-- 3) 回填既有詢問：依台北時區分日、同日內按建立時間排序
WITH numbered AS (
    SELECT id,
           (created_at AT TIME ZONE 'Asia/Taipei')::date AS tpe_day,
           ROW_NUMBER() OVER (
               PARTITION BY (created_at AT TIME ZONE 'Asia/Taipei')::date
               ORDER BY created_at, id
           ) AS seq
    FROM inquiries
)
UPDATE inquiries i
SET inquiry_no = 'FEN-' || to_char(n.tpe_day, 'YYYYMMDD') || '-' || lpad(n.seq::text, 3, '0')
FROM numbered n
WHERE i.id = n.id;

-- 4) 把各日已用掉的最大流水號寫進計數表，
--    避免回填後新產生的編號與回填編號碰撞
INSERT INTO inquiry_daily_counters (day, last_seq)
SELECT (created_at AT TIME ZONE 'Asia/Taipei')::date, count(*)
FROM inquiries
GROUP BY 1;

-- 5) 回填完成，收緊約束
ALTER TABLE inquiries ALTER COLUMN inquiry_no SET NOT NULL;
ALTER TABLE inquiries ADD CONSTRAINT inquiries_inquiry_no_key UNIQUE (inquiry_no);
```

- [ ] **Step 2: 在本機資料庫套用並驗證回填結果**

本機資料庫用 `docker-compose.yml` 啟動（`docker compose up -d`）。套用：

```bash
psql "postgres://fenny:fenny@localhost:5432/fenny_booking?sslmode=disable" \
  -f backend/db/migrations/005_inquiry_no.sql
```

預期：全部指令成功，無 ERROR。

- [ ] **Step 3: 驗證回填正確性**

```bash
psql "postgres://fenny:fenny@localhost:5432/fenny_booking?sslmode=disable" -c \
  "SELECT id, inquiry_no, created_at FROM inquiries ORDER BY id;"
psql "postgres://fenny:fenny@localhost:5432/fenny_booking?sslmode=disable" -c \
  "SELECT * FROM inquiry_daily_counters ORDER BY day;"
```

預期：
- 每筆 `inquiry_no` 皆非 NULL，格式為 `FEN-YYYYMMDD-NNN`
- 同一天的多筆詢問流水號從 001 開始連續遞增
- `inquiry_daily_counters` 每一天的 `last_seq` 等於該天的詢問筆數

若本機資料庫是空的（沒有既有詢問），回填部分不會有任何輸出，這是正常的 —— 但仍必須確認五個指令都沒有 ERROR。

- [ ] **Step 4: Commit**

```bash
git add backend/db/migrations/005_inquiry_no.sql
git commit -m "feat(db): add inquiry_no column and daily counter table"
```

---

### Task 2: 編號產生邏輯（Go）

**Files:**
- Create: `backend/service/inquiryno.go`
- Create: `backend/service/inquiryno_test.go`

**Interfaces:**
- Consumes: Task 1 的 `inquiry_daily_counters` 表
- Produces:
  - `service.FormatInquiryNo(t time.Time, seq int) string`
  - `service.NextInquiryNo(ctx context.Context, q InquiryNoQuerier, now time.Time) (string, error)`
  - `type InquiryNoQuerier interface { QueryRow(ctx context.Context, sql string, args ...any) pgx.Row }`

- [ ] **Step 1: 先寫失敗的測試**

建立 `backend/service/inquiryno_test.go`：

```go
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
```

「UTC 晚間跨日」這個案例是本任務的重點 —— 正式站時間是 UTC，若直接用 UTC 日期，台灣時間晚上 8 點後送出的詢問會被編成前一天的號。

- [ ] **Step 2: 執行測試確認失敗**

Run: `cd backend && go test ./service/ -run TestFormatInquiryNo -v`
Expected: 編譯失敗，`undefined: FormatInquiryNo`

- [ ] **Step 3: 實作**

建立 `backend/service/inquiryno.go`：

```go
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
```

- [ ] **Step 4: 執行測試確認通過**

Run: `cd backend && go test ./service/ -run TestFormatInquiryNo -v`
Expected: PASS，五個子測試全綠

- [ ] **Step 5: 確認整個後端仍可編譯**

Run: `cd backend && go build ./...`
Expected: 無輸出（成功）

- [ ] **Step 6: Commit**

```bash
git add backend/service/inquiryno.go backend/service/inquiryno_test.go
git commit -m "feat(backend): add inquiry number generation with Taipei timezone"
```

---

### Task 3: 建立詢問時寫入編號並回傳

**Files:**
- Modify: `backend/model/inquiry.go`
- Modify: `backend/handler/inquiry.go`（`Create`、`List`、`ToggleProcessed` 三處 SQL）

**Interfaces:**
- Consumes: `service.NextInquiryNo`（Task 2）
- Produces: API 回應的 JSON 多一個 `inquiry_no` 欄位（`string`，永不為 null）

- [ ] **Step 1: model 加欄位**

修改 `backend/model/inquiry.go`，在 `ID` 之後插入一行：

```go
type Inquiry struct {
	ID                int       `json:"id"`
	InquiryNo         string    `json:"inquiry_no"`
	Name              *string   `json:"name"`
	// ...其餘欄位維持不變
}
```

用 `string` 而非 `*string`，因為 Task 1 的 migration 已把欄位設為 `NOT NULL`。

- [ ] **Step 2: Create 取號並寫入**

修改 `backend/handler/inquiry.go` 的 `Create`。在 `defer tx.Rollback(ctx)` 之後、`INSERT INTO inquiries` 之前插入取號：

```go
	inquiryNo, err := service.NextInquiryNo(ctx, tx, time.Now())
	if err != nil {
		log.Printf("reserve inquiry number failed: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to create inquiry")
		return
	}
```

接著把 INSERT 改為帶入該欄位（注意 `RETURNING` 也要加，且 Scan 的順序必須一致）：

```go
	var inquiry model.Inquiry
	err = tx.QueryRow(ctx,
		`INSERT INTO inquiries (inquiry_no, name, email, course_id, headcount, city, district, address, budget, additional_contact)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id, inquiry_no, name, email, course_id, headcount, city, district, address, budget, additional_contact, is_processed, created_at`,
		inquiryNo, req.Name, req.Email, req.CourseID, req.Headcount, req.City, req.District, req.Address, req.Budget, req.AdditionalContact,
	).Scan(&inquiry.ID, &inquiry.InquiryNo, &inquiry.Name, &inquiry.Email, &inquiry.CourseID, &inquiry.Headcount,
		&inquiry.City, &inquiry.District, &inquiry.Address, &inquiry.Budget,
		&inquiry.AdditionalContact, &inquiry.IsProcessed, &inquiry.CreatedAt)
```

取號失敗時整筆 rollback（`defer tx.Rollback` 已處理），寧可讓顧客重送，也不要產生沒有編號的詢問單。

檔案頂端的 import 需要加入 `"time"`（`service` 已在 import 清單中）。

- [ ] **Step 3: List 與 ToggleProcessed 帶上編號**

`List` 的 SELECT 改為：

```go
		`SELECT i.id, i.inquiry_no, i.name, i.email, i.course_id, i.headcount, i.city, i.district, i.address, i.budget,
		        i.additional_contact, i.is_processed, i.created_at
		 FROM inquiries i
		 ORDER BY i.created_at DESC`,
```

對應的 Scan 加入 `&inq.InquiryNo`（緊接在 `&inq.ID` 之後）：

```go
		if err := rows.Scan(&inq.ID, &inq.InquiryNo, &inq.Name, &inq.Email, &inq.CourseID, &inq.Headcount,
			&inq.City, &inq.District, &inq.Address, &inq.Budget,
			&inq.AdditionalContact, &inq.IsProcessed, &inq.CreatedAt); err != nil {
```

`ToggleProcessed` 的 `RETURNING` 同樣加入 `inquiry_no`，Scan 加入 `&inquiry.InquiryNo`：

```go
		`UPDATE inquiries SET is_processed = NOT is_processed WHERE id = $1
		 RETURNING id, inquiry_no, name, email, course_id, headcount, city, district, address, budget, additional_contact, is_processed, created_at`,
```

```go
	).Scan(&inquiry.ID, &inquiry.InquiryNo, &inquiry.Name, &inquiry.Email, &inquiry.CourseID, &inquiry.Headcount,
		&inquiry.City, &inquiry.District, &inquiry.Address, &inquiry.Budget,
		&inquiry.AdditionalContact, &inquiry.IsProcessed, &inquiry.CreatedAt)
```

- [ ] **Step 4: 編譯**

Run: `cd backend && go build ./... && go vet ./...`
Expected: 無輸出

- [ ] **Step 5: 實際打 API 驗證（本機）**

啟動本機服務（需先 `docker compose up -d` 並套用 Task 1 的 migration）：

```bash
cd backend && go run . &
sleep 2
curl -s -X POST localhost:8080/api/inquiries -H 'Content-Type: application/json' \
  -d '{"name":"測試甲","email":"a@example.com","city":"台北市","district":"大安區","headcount":"10"}'
echo
curl -s -X POST localhost:8080/api/inquiries -H 'Content-Type: application/json' \
  -d '{"name":"測試乙","email":"b@example.com","city":"台中市","district":"西屯區","headcount":"20"}'
```

Expected: 兩筆皆回 HTTP 201，`inquiry_no` 分別為當日的連號（例如 `FEN-20260805-001`、`FEN-20260805-002`），日期為**台北日期**。

驗證完把測試資料清掉：

```bash
psql "postgres://fenny:fenny@localhost:5432/fenny_booking?sslmode=disable" -c \
  "DELETE FROM inquiries WHERE email IN ('a@example.com','b@example.com');"
```

- [ ] **Step 6: Commit**

```bash
git add backend/model/inquiry.go backend/handler/inquiry.go
git commit -m "feat(backend): assign and return inquiry number on creation"
```

---

### Task 4: Telegram 通知帶上編號

**Files:**
- Modify: `backend/service/telegram.go:86`（`buildInquiryMessage` 的標題行）
- Create: `backend/service/telegram_test.go`

**Interfaces:**
- Consumes: `model.Inquiry.InquiryNo`（Task 3）
- Produces: 通知訊息首行為 `📩 <b>收到新詢問</b> FEN-20260805-001`

- [ ] **Step 1: 先寫失敗的測試**

建立 `backend/service/telegram_test.go`：

```go
package service

import (
	"strings"
	"testing"

	"github.com/AlanSyue/fenny-studio-booking/backend/model"
)

func TestBuildInquiryMessageIncludesInquiryNo(t *testing.T) {
	name := "王小明"
	inquiry := model.Inquiry{
		InquiryNo: "FEN-20260805-001",
		Name:      &name,
		Email:     "a@example.com",
	}

	got := buildInquiryMessage(inquiry)

	firstLine := strings.SplitN(got, "\n", 2)[0]
	if !strings.Contains(firstLine, "FEN-20260805-001") {
		t.Errorf("首行應包含詢問編號，實際得到 %q", firstLine)
	}
	if !strings.Contains(firstLine, "收到新詢問") {
		t.Errorf("首行應保留原標題，實際得到 %q", firstLine)
	}
}

func TestBuildInquiryMessageWithoutInquiryNo(t *testing.T) {
	inquiry := model.Inquiry{Email: "a@example.com"}

	got := buildInquiryMessage(inquiry)

	firstLine := strings.SplitN(got, "\n", 2)[0]
	if strings.HasSuffix(firstLine, " ") {
		t.Errorf("編號為空時首行不應留下多餘空白，實際得到 %q", firstLine)
	}
}
```

- [ ] **Step 2: 執行測試確認失敗**

Run: `cd backend && go test ./service/ -run TestBuildInquiryMessage -v`
Expected: FAIL —— 首行不含 `FEN-20260805-001`

- [ ] **Step 3: 實作**

修改 `backend/service/telegram.go` 中 `buildInquiryMessage` 的標題行，把原本的：

```go
	b.WriteString("📩 <b>收到新詢問</b>\n")
```

改為：

```go
	b.WriteString("📩 <b>收到新詢問</b>")
	if no := strings.TrimSpace(inquiry.InquiryNo); no != "" {
		b.WriteString(" ")
		b.WriteString(html.EscapeString(no))
	}
	b.WriteString("\n")
```

`strings` 與 `html` 都已在 import 清單中。

- [ ] **Step 4: 執行測試確認通過**

Run: `cd backend && go test ./service/ -v`
Expected: PASS，本檔與 Task 2 的編號測試全綠

- [ ] **Step 5: Commit**

```bash
git add backend/service/telegram.go backend/service/telegram_test.go
git commit -m "feat(backend): include inquiry number in telegram notification"
```

---

### Task 5: 前端摘要組字與剪貼簿模組

**Files:**
- Modify: `frontend/package.json`（加入 Vitest）
- Create: `frontend/src/lib/inquirySummary.ts`
- Create: `frontend/src/lib/inquirySummary.test.ts`
- Create: `frontend/src/lib/clipboard.ts`
- Create: `frontend/src/config.ts`
- Modify: `frontend/src/types/index.ts`

**Interfaces:**
- Consumes: API 的 `inquiry_no`（Task 3）
- Produces:
  - `buildInquirySummary(inquiry: Inquiry): string`
  - `copyText(text: string): Promise<boolean>`
  - `LINE_URL: string`

- [ ] **Step 1: 安裝 Vitest**

```bash
cd frontend && npm install -D vitest
```

並在 `frontend/package.json` 的 `scripts` 中加入：

```json
    "test": "vitest run",
```

若安裝出現 peer dependency 衝突（Vite 8 較新），**不要使用 `--force` 或 `--legacy-peer-deps` 硬裝** —— 停下來回報，改以 Task 8 的 agent-browser 端對端驗證涵蓋這段邏輯。

- [ ] **Step 2: 型別加欄位**

修改 `frontend/src/types/index.ts` 的 `Inquiry` interface，在 `id` 之後加入：

```ts
export interface Inquiry {
  id: number
  inquiry_no: string
  name: string | null
  // ...其餘維持不變
}
```

- [ ] **Step 3: 先寫失敗的測試**

建立 `frontend/src/lib/inquirySummary.test.ts`：

```ts
import { describe, it, expect } from 'vitest'
import { buildInquirySummary } from './inquirySummary'
import type { Inquiry } from '../types'

const base: Inquiry = {
  id: 1,
  inquiry_no: 'FEN-20260805-001',
  name: '王小明',
  email: 'a@example.com',
  courses: [
    { id: 1, name: '鬱金香花圈' },
    { id: 2, name: '乾燥花束' },
  ],
  headcount: '20',
  city: '台北市',
  district: '大安區',
  address: '復興南路一段 100 號',
  budget: '每人 800',
  additional_contact: 'LINE: abc123',
  is_processed: false,
  created_at: '2026-08-05T03:52:36Z',
}

describe('buildInquirySummary', () => {
  it('第一行是詢問編號', () => {
    const lines = buildInquirySummary(base).split('\n')
    expect(lines[0]).toBe('【詢問編號 FEN-20260805-001】')
  })

  it('包含所有已填欄位', () => {
    const summary = buildInquirySummary(base)
    expect(summary).toContain('稱呼：王小明')
    expect(summary).toContain('Email：a@example.com')
    expect(summary).toContain('人數：20')
    expect(summary).toContain('預算：每人 800')
    expect(summary).toContain('其他聯絡方式：LINE: abc123')
  })

  it('多個課程以頓號串接', () => {
    expect(buildInquirySummary(base)).toContain('課程：鬱金香花圈、乾燥花束')
  })

  it('地點合併縣市、區與地址', () => {
    expect(buildInquirySummary(base)).toContain(
      '地點：台北市大安區復興南路一段 100 號'
    )
  })

  it('略過空欄位，不留下空標籤', () => {
    const sparse: Inquiry = {
      ...base,
      name: null,
      courses: [],
      headcount: null,
      address: null,
      budget: null,
      additional_contact: null,
    }
    const summary = buildInquirySummary(sparse)
    expect(summary).not.toContain('稱呼')
    expect(summary).not.toContain('課程')
    expect(summary).not.toContain('人數')
    expect(summary).not.toContain('預算')
    expect(summary).not.toContain('其他聯絡方式')
    expect(summary).toContain('地點：台北市大安區')
    expect(summary).toContain('Email：a@example.com')
  })

  it('略過只有空白字元的欄位', () => {
    const summary = buildInquirySummary({ ...base, headcount: '   ' })
    expect(summary).not.toContain('人數')
  })
})
```

- [ ] **Step 4: 執行測試確認失敗**

Run: `cd frontend && npm run test`
Expected: FAIL —— 找不到模組 `./inquirySummary`

- [ ] **Step 5: 實作摘要組字**

建立 `frontend/src/lib/inquirySummary.ts`：

```ts
import type { Inquiry } from '../types'

/**
 * 把一筆詢問整理成可貼進 LINE 的條列式摘要。
 * 空值欄位直接略過，不留下空標籤。
 */
export function buildInquirySummary(inquiry: Inquiry): string {
  const lines: string[] = [`【詢問編號 ${inquiry.inquiry_no}】`]

  const push = (label: string, value: string | null | undefined) => {
    const trimmed = (value ?? '').trim()
    if (trimmed) lines.push(`${label}：${trimmed}`)
  }

  push('稱呼', inquiry.name)
  push('Email', inquiry.email)

  const courseNames = (inquiry.courses ?? [])
    .map((c) => c.name)
    .filter((n) => n && n.trim())
  if (courseNames.length > 0) {
    lines.push(`課程：${courseNames.join('、')}`)
  }

  push('人數', inquiry.headcount)

  const location = [inquiry.city, inquiry.district, inquiry.address]
    .map((part) => (part ?? '').trim())
    .filter(Boolean)
    .join('')
  push('地點', location)

  push('預算', inquiry.budget)
  push('其他聯絡方式', inquiry.additional_contact)

  return lines.join('\n')
}
```

- [ ] **Step 6: 執行測試確認通過**

Run: `cd frontend && npm run test`
Expected: PASS，六個測試全綠

- [ ] **Step 7: 實作剪貼簿模組**

建立 `frontend/src/lib/clipboard.ts`：

```ts
/**
 * 將文字寫入剪貼簿，回傳是否成功。
 *
 * 顧客常從 LINE 內建瀏覽器開啟頁面，該環境不保證有 Clipboard API
 * （非安全內容或舊版 WebView 皆可能缺席），因此保留 execCommand 降級。
 */
export async function copyText(text: string): Promise<boolean> {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      // 落到下面的降級路徑
    }
  }

  try {
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.setAttribute('readonly', '')
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(textarea)
    return ok
  } catch {
    return false
  }
}
```

- [ ] **Step 8: 建立 LINE 連結設定**

建立 `frontend/src/config.ts`：

```ts
/**
 * LINE 官方帳號加好友連結。
 *
 * 內建預設值，確保部署時忘記設定環境變數也不會產生壞連結；
 * 要換帳號時設 VITE_LINE_URL 覆蓋即可。設為空字串則隱藏按鈕。
 */
export const LINE_URL =
  import.meta.env.VITE_LINE_URL ?? 'https://line.me/R/ti/p/@165nacbf'
```

- [ ] **Step 9: 確認型別檢查與建置通過**

Run: `cd frontend && npm run build`
Expected: 建置成功，無 TypeScript 錯誤

- [ ] **Step 10: Commit**

```bash
git add frontend/package.json frontend/package-lock.json frontend/src/config.ts \
        frontend/src/lib/ frontend/src/types/index.ts
git commit -m "feat(frontend): add inquiry summary builder, clipboard helper and LINE config"
```

---

### Task 6: 成功頁改版

**Files:**
- Modify: `frontend/src/pages/InquiryForm.tsx:10`（state）、`:74-93`（送出）、`:99-118`（成功頁）

**Interfaces:**
- Consumes: `buildInquirySummary`、`copyText`、`LINE_URL`（Task 5）；API 回傳的 `Inquiry`（Task 3）
- Produces: 顧客可見的成功頁

- [ ] **Step 1: 改用 result state 取代 submitted boolean**

修改 `frontend/src/pages/InquiryForm.tsx`。import 區加入：

```ts
import type { Course, Inquiry } from '../types'
import { buildInquirySummary } from '../lib/inquirySummary'
import { copyText } from '../lib/clipboard'
import { LINE_URL } from '../config'
```

把 `const [submitted, setSubmitted] = useState(false)` 換成：

```ts
  const [result, setResult] = useState<Inquiry | null>(null)
  const [copied, setCopied] = useState(false)
  const [copyFailed, setCopyFailed] = useState(false)
```

- [ ] **Step 2: 送出時接住回傳值**

把 `handleSubmit` 中的 `await apiFetch('/api/inquiries', {...})` 與其後的 `setSubmitted(true)` 改為：

```ts
      const created = await apiFetch<Inquiry>('/api/inquiries', {
        method: 'POST',
        body: JSON.stringify({
          name: form.name,
          email: form.email,
          course_ids: form.course_ids,
          headcount: form.headcount || null,
          city: form.city,
          district: form.district,
          address: form.address || null,
          budget: form.budget || null,
          additional_contact: form.additional_contact || null,
        }),
      })
      setResult(created)
```

- [ ] **Step 3: 改寫成功頁**

把 `if (submitted) { ... }` 整段換成：

```tsx
  const handleCopy = async () => {
    if (!result) return
    const ok = await copyText(buildInquirySummary(result))
    if (ok) {
      setCopied(true)
      setCopyFailed(false)
      setTimeout(() => setCopied(false), 2000)
    } else {
      setCopyFailed(true)
    }
  }

  if (result) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-amber-50 to-orange-50 flex items-center justify-center p-4">
        <div className="bg-white rounded-2xl shadow-lg p-8 max-w-md w-full text-center">
          <div className="text-5xl mb-4">✅</div>
          <h2 className="text-2xl font-bold text-gray-900 mb-4">已收到您的詢問！</h2>

          <div className="rounded-xl bg-amber-50 border border-amber-200 px-4 py-3 mb-5">
            <p className="text-xs text-amber-700 mb-1">詢問編號</p>
            <p className="text-xl font-bold font-mono tracking-wider text-amber-900">
              {result.inquiry_no}
            </p>
          </div>

          <p className="text-gray-600 mb-1">我們通常會於 1 個工作天內回覆。</p>
          <p className="text-gray-600 mb-5">
            為了加快討論，請加入 LINE 官方帳號，並把詢問編號傳給我們。
          </p>

          {LINE_URL && (
            <a
              href={LINE_URL}
              target="_blank"
              rel="noopener noreferrer"
              className="block w-full rounded-lg bg-[#06C755] px-4 py-3 font-bold text-white hover:bg-[#05b34c] transition"
            >
              加入 LINE 官方帳號
            </a>
          )}

          <button
            onClick={handleCopy}
            className="mt-3 w-full rounded-lg border border-gray-300 px-4 py-2.5 font-medium text-gray-700 hover:bg-gray-50 transition cursor-pointer"
          >
            {copied ? '已複製 ✓' : '複製詢問內容給店家'}
          </button>

          {copyFailed && (
            <div className="mt-3 text-left">
              <p className="text-xs text-gray-500 mb-1">
                無法自動複製，請長按下方文字複製：
              </p>
              <pre className="whitespace-pre-wrap break-words rounded-lg bg-gray-50 border border-gray-200 p-3 text-xs text-gray-700 select-all">
                {buildInquirySummary(result)}
              </pre>
            </div>
          )}

          <button
            onClick={() => {
              setResult(null)
              setCopied(false)
              setCopyFailed(false)
              resetForm()
            }}
            className="mt-6 text-amber-600 hover:text-amber-700 font-medium cursor-pointer"
          >
            填寫新的詢問
          </button>
        </div>
      </div>
    )
  }
```

`#06C755` 是 LINE 官方品牌綠，讓按鈕一眼可辨識。

- [ ] **Step 4: 建置確認**

Run: `cd frontend && npm run build`
Expected: 建置成功，無 TypeScript 錯誤（特別確認沒有殘留的 `submitted` 參照）

- [ ] **Step 5: Commit**

```bash
git add frontend/src/pages/InquiryForm.tsx
git commit -m "feat(frontend): show inquiry number, LINE button and copy summary on success page"
```

---

### Task 7: 後台編號欄位與搜尋

**Files:**
- Modify: `frontend/src/pages/InquiryList.tsx`

**Interfaces:**
- Consumes: API `List` 回傳的 `inquiry_no`（Task 3）
- Produces: 店家可用編號查到詢問（整個需求的閉環）

- [ ] **Step 1: 加入搜尋 state 與過濾邏輯**

修改 `frontend/src/pages/InquiryList.tsx`。在 `filter` state 之後加入：

```ts
  const [search, setSearch] = useState('')
```

把 `filtered` 的計算改為同時套用狀態篩選與關鍵字搜尋：

```ts
  const keyword = search.trim().toLowerCase()
  const filtered = inquiries.filter((inq) => {
    if (filter === 'pending' && inq.is_processed) return false
    if (filter === 'processed' && !inq.is_processed) return false
    if (!keyword) return true
    return [inq.inquiry_no, inq.name ?? '', inq.email]
      .some((field) => field.toLowerCase().includes(keyword))
  })
```

- [ ] **Step 2: 加入搜尋框**

在標題列的 `<div className="flex gap-2">` 之前插入搜尋框，並把外層改為可換行以免窄螢幕擠壓：

把 `<div className="flex justify-between items-center mb-6">` 改為
`<div className="flex flex-wrap justify-between items-center gap-3 mb-6">`，
然後在 `<h1>` 之後插入：

```tsx
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="搜尋編號 / 稱呼 / Email"
          className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm text-gray-900 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200 outline-none transition w-56"
        />
```

- [ ] **Step 3: 表格加入編號欄**

在 `<thead>` 的「狀態」`<th>` 之前插入：

```tsx
                  <th className="text-left px-4 py-3 text-xs font-medium text-gray-500 uppercase">編號</th>
```

在 `<tbody>` 每列的狀態 `<td>` 之前插入：

```tsx
                    <td className="px-4 py-3 text-sm font-mono text-gray-700 whitespace-nowrap">
                      {inq.inquiry_no}
                    </td>
```

- [ ] **Step 4: 補上查無結果的提示**

把 `filtered.length === 0` 的空狀態訊息改為能區分兩種情況：

```tsx
        <div className="text-center py-12 text-gray-500">
          <p>{keyword ? `找不到符合「${search}」的詢問` : '沒有詢問記錄'}</p>
        </div>
```

- [ ] **Step 5: 建置確認**

Run: `cd frontend && npm run build`
Expected: 建置成功

- [ ] **Step 6: Commit**

```bash
git add frontend/src/pages/InquiryList.tsx
git commit -m "feat(admin): show inquiry number and add keyword search"
```

---

### Task 8: 端對端驗證與上線

**Files:**
- 不修改程式碼；驗證與部署

**Interfaces:**
- Consumes: Task 1–7 的全部成果

- [ ] **Step 1: 執行完整測試**

```bash
cd backend && go test ./... && go vet ./...
cd ../frontend && npm run test && npm run build
```

Expected: 全部通過

- [ ] **Step 2: 本機端對端驗證**

啟動本機前後端（`cd backend && go run .`、`cd frontend && npm run dev`），使用 **agent-browser skill**（本專案慣例，不使用 Playwright）驗證：

1. 開啟表單頁，填寫稱呼、Email、選 2 個課程、人數、縣市、區、預算後送出
2. 成功頁顯示 `FEN-` 開頭的詢問編號
3. 「加入 LINE 官方帳號」按鈕的 `href` 為 `https://line.me/R/ti/p/@165nacbf`
4. 點「複製詢問內容給店家」→ 按鈕文字變「已複製 ✓」
5. 讀取剪貼簿內容，確認第一行為 `【詢問編號 FEN-...】`，且包含課程、人數、地點
6. 登入後台詢問列表，把該編號貼進搜尋框，確認只剩該筆詢問
7. 清除搜尋，確認列表恢復完整

- [ ] **Step 3: 清除本機測試資料**

```bash
psql "postgres://fenny:fenny@localhost:5432/fenny_booking?sslmode=disable" -c \
  "DELETE FROM inquiries WHERE email LIKE '%@example.com';"
```

- [ ] **Step 4: 合併回 main 並推送**

```bash
git checkout main
git merge --no-ff feat/inquiry-no-line-guide
git push origin main
```

- [ ] **Step 5: 部署（需經使用者確認後執行）**

```bash
./deploy.sh -y
```

`deploy.sh` 會自動套用 `005_inquiry_no.sql`。部署後確認輸出中出現 migration 已套用與健康檢查通過。

- [ ] **Step 6: 線上驗證**

依 `AGENTS.md` 的「部署後驗證」確認新程式碼真的在跑：

```bash
ssh root@139.162.76.210 'stat -c "%y" /opt/fenny-studio-booking/backend/bin/backend'
ssh root@139.162.76.210 'strings /opt/fenny-studio-booking/backend/bin/backend | grep -c "inquiry_daily_counters"'
```

Expected: mtime 為剛剛；grep 計數 ≥ 1

接著在 https://booking.fenny-studio.com 送出一筆標示為測試的詢問，確認：
- 成功頁顯示編號
- Telegram 通知標題含該編號
- 後台可用編號搜尋到

驗證後刪除該筆測試資料（作法見本次工作階段清除 id=9 的方式）。

---

## Self-Review

**Spec coverage：** 條列摘要（Task 5）、Telegram 帶編號（Task 4）、成功頁顯示編號與加 LINE 按鈕（Task 6）、編號產生與時區（Task 1–3）、後台以編號查找（Task 7）、agent-browser 端對端驗證（Task 8）—— spec 各節皆有對應任務。

**型別一致性：** `FormatInquiryNo` / `NextInquiryNo` / `InquiryNo` / `inquiry_no` / `buildInquirySummary` / `copyText` / `LINE_URL` 在定義處與使用處名稱一致。

**已知風險：** Vite 8 與 Vitest 的相容性未經實測（Task 5 Step 1 已註明衝突時的處理方式，禁止硬裝）。
