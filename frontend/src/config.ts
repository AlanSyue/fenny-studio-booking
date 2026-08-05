/**
 * LINE 官方帳號加好友連結。
 *
 * 內建預設值，確保部署時忘記設定環境變數也不會產生壞連結；
 * 要換帳號時設 VITE_LINE_URL 覆蓋即可。設為空字串則隱藏按鈕。
 */
export const LINE_URL =
  import.meta.env.VITE_LINE_URL ?? 'https://line.me/R/ti/p/@165nacbf'
