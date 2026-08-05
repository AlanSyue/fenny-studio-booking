# AGENTS.md

## 專案結構

- `backend/` — Go 1.25.5（chi + pgx），Postgres。migrations 放 `backend/db/migrations/`
- `frontend/` — React 19 + Vite + Tailwind 4
- `deploy.sh` — 唯一的正式站部署入口

## 部署流程

### 一律使用 deploy.sh

```bash
./deploy.sh        # 互動式，要求本機 main 乾淨
./deploy.sh -y     # 跳過確認
```

`deploy.sh` 會完成整套流程：更新遠端 repo → 套用新 migration → `go build -o bin/backend .` → 前端 `npm ci && npm run build` → `systemctl restart fenny-booking` → 健康檢查。

**絕對不要在正式站手動 `go build`。** 這是 2026-08-05 一次線上事故的直接成因，見下方「歷史事故」。

### 正式站資訊

| 項目 | 值 |
|---|---|
| 主機 | `root@139.162.76.210` |
| 路徑 | `/opt/fenny-studio-booking` |
| systemd unit | `fenny-booking` |
| 服務執行檔 | `/opt/fenny-studio-booking/backend/bin/backend` |
| 執行身分 | `fenny` |
| 公開網址 | https://booking.fenny-studio.com |

### 環境變數

正式站的 `backend/.env` 是**遠端獨立維護的事實來源**，`deploy.sh` 不會覆蓋它。新增環境變數時，除了更新 `backend/.env.example`，**必須同時手動更新遠端的 `backend/.env`**，否則新功能會靜默失效。

systemd 同時透過 `EnvironmentFile` 載入這個檔案，`WorkingDirectory` 也設在 `backend/`，因此 `godotenv.Load()` 能正常讀到。

### 部署後驗證

部署訊息顯示成功不等於新程式碼真的在跑。確認實際生效：

```bash
# binary 的建置時間應為剛剛
ssh root@139.162.76.210 'stat -c "%y" /opt/fenny-studio-booking/backend/bin/backend'

# 確認新程式碼真的被編譯進去（換成本次改動特有的字串）
ssh root@139.162.76.210 'strings /opt/fenny-studio-booking/backend/bin/backend | grep -c "api.telegram.org"'

# 確認跑的是新行程
ssh root@139.162.76.210 'ps -eo pid,lstart,cmd | grep "[b]in/backend"'
```

## 歷史事故：2026-08-05 Telegram 通知從未送出

**症狀：** 新詢問的 Telegram 通知一直收不到，且服務 log 中沒有任何錯誤。

**根因：** commit `7f155f1`（2026-06-28）加入 Telegram 通知功能後，是在正式站**手動** `go build` 的，產物叫 `backend/server`；但 systemd 的 `ExecStart` 指向 `backend/bin/backend` —— 那顆停留在 2026-04-19 的建置版本。線上跑的執行檔裡根本沒有發送通知的程式碼，所以它不會嘗試、不會失敗、log 自然一片空白。

**教訓：**

1. 後端上線一律走 `./deploy.sh`，不要手動 build
2. **「log 沒有錯誤」不等於「功能正常」** —— 也可能是那段程式碼根本不存在於執行檔中
3. 診斷通知類問題時，先用 `strings <binary> | grep` 確認功能是否真的被編譯進線上執行檔，再往設定或憑證方向查

## 驗證慣例

- 端對端驗證使用 agent-browser skill，不使用 Playwright
- 靜態資源（favicon / OG image / logo 等）一律下載到 `frontend/public/`，不 hotlink 遠端 URL
