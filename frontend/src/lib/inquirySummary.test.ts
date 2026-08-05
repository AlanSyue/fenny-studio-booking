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
