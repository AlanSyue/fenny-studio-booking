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
