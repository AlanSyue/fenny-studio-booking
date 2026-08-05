import { useState, useEffect } from 'react'
import { apiFetch } from '../api/client'
import type { Course, Inquiry } from '../types'
import { TAIWAN_REGIONS } from '../data/taiwanRegions'
import CoursePickerModal from '../components/CoursePickerModal'
import { buildInquirySummary } from '../lib/inquirySummary'
import { copyText } from '../lib/clipboard'
import { LINE_URL } from '../config'

export default function InquiryForm() {
  const [courses, setCourses] = useState<Course[]>([])
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<Inquiry | null>(null)
  const [copied, setCopied] = useState(false)
  const [copyFailed, setCopyFailed] = useState(false)
  const [error, setError] = useState('')
  const [pickerOpen, setPickerOpen] = useState(false)
  const [form, setForm] = useState({
    name: '',
    email: '',
    course_ids: [] as number[],
    headcount: '',
    city: '',
    district: '',
    address: '',
    budget: '',
    additional_contact: '',
  })

  useEffect(() => {
    apiFetch<Course[]>('/api/courses').then(setCourses).catch(() => {})
  }, [])

  const resetForm = () =>
    setForm({
      name: '',
      email: '',
      course_ids: [],
      headcount: '',
      city: '',
      district: '',
      address: '',
      budget: '',
      additional_contact: '',
    })

  const toggleCourse = (id: number) => {
    setForm((prev) => ({
      ...prev,
      course_ids: prev.course_ids.includes(id)
        ? prev.course_ids.filter((cid) => cid !== id)
        : [...prev.course_ids, id],
    }))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    if (!form.name) {
      setError('請填寫怎麼稱呼')
      return
    }
    if (!form.email) {
      setError('請填寫 Email')
      return
    }
    if (!form.city) {
      setError('請選擇活動縣市')
      return
    }
    if (!form.district) {
      setError('請選擇活動區/鄉/鎮')
      return
    }

    setLoading(true)
    try {
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
    } catch (err) {
      setError('提交失敗，請稍後再試')
    } finally {
      setLoading(false)
    }
  }

  const districts =
    TAIWAN_REGIONS.find((r) => r.city === form.city)?.districts ?? []

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

  return (
    <div className="min-h-screen bg-gradient-to-br from-amber-50 to-orange-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-2xl shadow-lg p-8 max-w-lg w-full">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-gray-900">團體手作課程報價</h1>
          <p className="text-gray-600 mt-2">芬妮手作 Fenny Studio・企業活動・親子包班・材料包客製</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              怎麼稱呼 <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              required
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-gray-900 focus:border-amber-500 focus:ring-2 focus:ring-amber-200 outline-none transition"
              placeholder="您的稱呼"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Email <span className="text-red-500">*</span>
            </label>
            <input
              type="email"
              required
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
              className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-gray-900 focus:border-amber-500 focus:ring-2 focus:ring-amber-200 outline-none transition"
              placeholder="your@email.com"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              想要詢問的手作課程項目
            </label>

            {/* Trigger button */}
            <button
              type="button"
              onClick={() => setPickerOpen(true)}
              className="w-full rounded-lg border border-gray-300 hover:border-amber-500 px-4 py-3 text-left transition flex items-center justify-between cursor-pointer"
            >
              <span className="text-gray-700">
                {form.course_ids.length > 0
                  ? `已選 ${form.course_ids.length} 門課程`
                  : '點選課程（可多選）'}
              </span>
              <span className="text-gray-400">›</span>
            </button>

            {/* Selected chips */}
            {form.course_ids.length > 0 && (
              <div className="flex flex-wrap gap-2 mt-3">
                {courses
                  .filter((c) => form.course_ids.includes(c.id))
                  .map((c) => (
                    <span
                      key={c.id}
                      className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-amber-100 text-amber-800 text-sm"
                    >
                      <span className="max-w-[200px] truncate">{c.name}</span>
                      <button
                        type="button"
                        onClick={() => toggleCourse(c.id)}
                        className="hover:text-amber-900 text-base leading-none cursor-pointer"
                        aria-label={`移除 ${c.name}`}
                      >
                        ×
                      </button>
                    </span>
                  ))}
              </div>
            )}
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              預估人數
            </label>
            <div className="flex flex-wrap gap-2">
              {['10-20人', '20-30人', '30-40人', '40-50人', '50人以上'].map((option) => (
                <button
                  key={option}
                  type="button"
                  onClick={() =>
                    setForm({ ...form, headcount: form.headcount === option ? '' : option })
                  }
                  className={`px-4 py-2 rounded-lg text-sm font-medium transition cursor-pointer ${
                    form.headcount === option
                      ? 'bg-amber-500 text-white'
                      : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                  }`}
                >
                  {option}
                </button>
              ))}
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              活動地點
            </label>
            <div className="space-y-3">
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">
                  縣市 <span className="text-red-500">*</span>
                </label>
                <select
                  value={form.city}
                  onChange={(e) =>
                    setForm({ ...form, city: e.target.value, district: '' })
                  }
                  className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-gray-900 focus:border-amber-500 focus:ring-2 focus:ring-amber-200 outline-none transition bg-white"
                >
                  <option value="">請選擇縣市</option>
                  {TAIWAN_REGIONS.map((r) => (
                    <option key={r.city} value={r.city}>
                      {r.city}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">
                  區/鄉/鎮 <span className="text-red-500">*</span>
                </label>
                <select
                  value={form.district}
                  onChange={(e) => setForm({ ...form, district: e.target.value })}
                  disabled={!form.city}
                  className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-gray-900 focus:border-amber-500 focus:ring-2 focus:ring-amber-200 outline-none transition bg-white disabled:bg-gray-100 disabled:text-gray-400"
                >
                  <option value="">請選擇區/鄉/鎮</option>
                  {districts.map((d) => (
                    <option key={d} value={d}>
                      {d}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <input
                  type="text"
                  value={form.address}
                  onChange={(e) => setForm({ ...form, address: e.target.value })}
                  className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-gray-900 focus:border-amber-500 focus:ring-2 focus:ring-amber-200 outline-none transition"
                  placeholder="詳細地址（選填）"
                />
              </div>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              總預算 / 備註
            </label>
            <textarea
              value={form.budget}
              onChange={(e) => setForm({ ...form, budget: e.target.value })}
              rows={3}
              className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-gray-900 focus:border-amber-500 focus:ring-2 focus:ring-amber-200 outline-none transition resize-none"
              placeholder="例如：預算 5 萬以內，或其他備註"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              其他聯絡方式補充
            </label>
            <textarea
              value={form.additional_contact}
              onChange={(e) => setForm({ ...form, additional_contact: e.target.value })}
              rows={3}
              className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-gray-900 focus:border-amber-500 focus:ring-2 focus:ring-amber-200 outline-none transition resize-none"
              placeholder="例如：LINE ID、電話號碼等"
            />
          </div>

          {error && (
            <p className="text-red-500 text-sm">{error}</p>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-amber-500 hover:bg-amber-600 text-white font-medium py-3 rounded-lg transition disabled:opacity-50 cursor-pointer"
          >
            {loading ? '提交中...' : '送出詢問'}
          </button>
        </form>
      </div>

      <CoursePickerModal
        open={pickerOpen}
        onClose={() => setPickerOpen(false)}
        courses={courses}
        selectedIds={form.course_ids}
        onToggle={toggleCourse}
      />
    </div>
  )
}
