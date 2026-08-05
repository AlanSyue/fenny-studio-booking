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
