# 詢問編號與 LINE 導流 — 設計文件

日期：2026-08-05
狀態：已與使用者確認，待實作

## 背景

團體手作課程的公開詢問表單（`/`）目前送出後只顯示一句「感謝您的詢問！我們會盡快與您聯繫。」，顧客拿不到任何可追蹤的識別碼，店家也無法在 LINE 上對應顧客身分。

店家希望顧客送出表單後主動加入 LINE 官方帳號繼續討論，但 LINE 官方帳號無法主動傳訊給尚未加好友的使用者，因此必須由成功頁引導顧客加好友，並讓顧客在 LINE 中報上一組編號，店家據此在後台找到對應的詢問資料。

## 決策紀錄

1. **不實作 LINE push 通知。** LINE Notify 已於 2025-03-31 終止服務；改用 Messaging API push 需要 Channel Access Token 與店家本人的 userId（需額外實作 webhook 取得）。經確認，新詢問通知**維持既有的 Telegram 管道**，LINE 只負責顧客導流。
2. **編號採每日流水號 `FEN-YYYYMMDD-NNN`**，由獨立的每日計數表以 Postgres 原子操作產生（見下）。
3. **成功頁提供「複製詢問內容給店家」按鈕**，一鍵複製「編號 + 完整條列摘要」，顧客加 LINE 後直接貼上。
4. **LINE 連結內建預設值**：`https://line.me/R/ti/p/@165nacbf`，可用環境變數覆蓋。

## 不做的事（YAGNI）

- 不做 LINE Messaging API push、不做 LINE webhook、不做 LINE 帳號綁定
- 不做後台設定頁管理 LINE 連結（改用環境變數 + 程式碼預設值）
- 不做顧客用編號自助查詢進度的頁面

## 後端設計

### Migration `005_inquiry_no.sql`

```sql
ALTER TABLE inquiries ADD COLUMN inquiry_no VARCHAR(20) UNIQUE;

CREATE TABLE inquiry_daily_counters (
    day      DATE PRIMARY KEY,
    last_seq INTEGER NOT NULL
);
```

既有詢問資料依 `created_at`（轉為 Asia/Taipei）分日排序回填 `inquiry_no`，讓所有單都有號可查；回填同時把各日的最大流水號寫入 `inquiry_daily_counters`，確保回填後新產生的號不會與回填號碰撞。

回填完成後，於同一個 migration 內將欄位改為 `NOT NULL`：

```sql
ALTER TABLE inquiries ALTER COLUMN inquiry_no SET NOT NULL;
```

理由：`model.Inquiry.InquiryNo` 以 `string`（非指標）宣告，若資料庫允許 NULL，掃描到 NULL 會直接造成 scan error。加上 `NOT NULL` 讓資料庫保證這個不變條件，也擋掉未來有人手動插入無編號詢問單的可能。

### 取號機制

在 `handler/inquiry.go` 的 `Create` 既有 transaction 內，於 INSERT 之前取號：

```sql
INSERT INTO inquiry_daily_counters (day, last_seq)
VALUES ($1, 1)
ON CONFLICT (day) DO UPDATE SET last_seq = inquiry_daily_counters.last_seq + 1
RETURNING last_seq;
```

這是 Postgres 的原子操作，同時湧入多筆詢問也不可能取得相同流水號，不需要額外的 advisory lock。

**時區：** 日期一律以 `Asia/Taipei` 計算。伺服器時間為 UTC，若直接用 UTC 日期，台灣時間晚上 8 點之後送出的詢問會被編成隔天的號。

**格式化：** `FEN-` + `YYYYMMDD` + `-` + 流水號補零至 3 位（超過 999 自然變 4 位）。抽成純函式 `formatInquiryNo(day time.Time, seq int) string` 以便單元測試。

**錯誤處理：** 取號失敗則整筆 transaction rollback，回傳 500。寧可讓顧客重送，也不要產生沒有編號的詢問單。

### 其他後端調整

- `model.Inquiry` 新增 `InquiryNo string \`json:"inquiry_no"\``；`Create` / `List` / `ToggleProcessed` 的 SQL 與 Scan 都要帶上這個欄位。
- `service/telegram.go` 的 `buildInquiryMessage` 標題改為 `📩 <b>收到新詢問</b> FEN-20260805-001`，其餘條列欄位不變。通知維持非同步、失敗只寫 log，不影響顧客送出結果。

## 前端設計

### 成功頁 `pages/InquiryForm.tsx`

`handleSubmit` 改為接住 API 回傳的完整 inquiry（含 `inquiry_no` 與課程名稱）存入 state。成功頁版面由上而下：

1. `✅ 已收到您的詢問！`
2. 詢問編號區塊（大字、等寬字體，點擊可複製編號本身）
3. 「我們通常會於 1 個工作天內回覆。為了加快討論，請加入 LINE 官方帳號。」
4. 【加入 LINE】主要按鈕 → 開新分頁至 LINE 連結（`target="_blank"` + `rel="noopener noreferrer"`）
5. [複製詢問內容給店家] 次要按鈕
6. 「填寫新的詢問」（保留現有行為）

複製成功後按鈕文字變為「已複製 ✓」，2 秒後復原。

### 摘要組字 `lib/inquirySummary.ts`

獨立純函式 `buildInquirySummary(inquiry): string`。第一行 `【詢問編號 FEN-20260805-001】`，其後條列：稱呼、Email、課程、人數、地點（縣市 + 區 + 地址）、預算、其他聯絡方式。空值欄位直接略過。

抽成獨立檔案的理由：可單獨寫單元測試，且避免已達 328 行的 `InquiryForm.tsx` 繼續膨脹。

### LINE 連結設定 `config.ts`

```ts
export const LINE_URL =
  import.meta.env.VITE_LINE_URL ?? 'https://line.me/R/ti/p/@165nacbf'
```

前端目前沒有任何 `.env` 機制（`API_BASE` 為空字串走同源），內建預設值可確保部署時忘記設定環境變數也不會產生壞連結。若 `LINE_URL` 為空字串則隱藏按鈕。

### 複製功能的降級處理

顧客常從 LINE 內建瀏覽器開啟頁面，該環境不保證有 Clipboard API。降級順序：

1. `navigator.clipboard.writeText()`
2. 隱藏 `<textarea>` + `document.execCommand('copy')`
3. 兩者皆失敗 → 將摘要展開為可長按全選的區塊，並提示「請長按複製」

### 後台 `pages/InquiryList.tsx`

- 表格最前方新增「編號」欄
- 新增搜尋框，比對編號／稱呼／Email（前端過濾，與現有的全部／未處理／已處理篩選並存）

這一步是整個需求的閉環：顧客在 LINE 傳來編號 → 店家貼進搜尋框 → 立即找到該筆詢問。

## 驗證方式

- **單元測試：** Go 的 `formatInquiryNo`（含補零、跨日、時區）；前端的 `buildInquirySummary`（含空欄位略過）
- **並行安全：** 由 `ON CONFLICT DO UPDATE` 的原子性保證，不另寫壓力測試
- **端對端：** 用 agent-browser 實際跑一次完整流程 —— 送出表單 → 成功頁顯示編號 → 複製摘要 → 後台以編號搜尋找到該筆詢問

## 相關背景

實作前已修復一項既有問題：正式站 systemd 執行的 `backend/bin/backend` 停留在 2026-04-19 建置的版本，不含 commit `7f155f1` 加入的 Telegram 通知程式碼，導致通知從未送出。已重新部署並驗證通知可正常發送。
