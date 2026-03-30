import { useState, useEffect } from 'react'
import { apiFetch } from '../api/client'
import type { Course } from '../types'

export default function InquiryForm() {
  const [courses, setCourses] = useState<Course[]>([])
  const [loading, setLoading] = useState(false)
  const [submitted, setSubmitted] = useState(false)
  const [error, setError] = useState('')
  const [form, setForm] = useState({
    name: '',
    email: '',
    course_id: null as number | null,
    headcount: '' as string,
    additional_contact: '',
  })

  useEffect(() => {
    apiFetch<Course[]>('/api/courses').then(setCourses).catch(() => {})
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    if (!form.email) {
      setError('請填寫 Email')
      return
    }

    setLoading(true)
    try {
      await apiFetch('/api/inquiries', {
        method: 'POST',
        body: JSON.stringify({
          name: form.name || null,
          email: form.email,
          course_id: form.course_id,
          headcount: form.headcount || null,
          additional_contact: form.additional_contact || null,
        }),
      })
      setSubmitted(true)
    } catch (err) {
      setError('提交失敗，請稍後再試')
    } finally {
      setLoading(false)
    }
  }

  if (submitted) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-amber-50 to-orange-50 flex items-center justify-center p-4">
        <div className="bg-white rounded-2xl shadow-lg p-8 max-w-md w-full text-center">
          <div className="text-5xl mb-4">✨</div>
          <h2 className="text-2xl font-bold text-gray-900 mb-2">感謝您的詢問！</h2>
          <p className="text-gray-600">我們會盡快與您聯繫。</p>
          <button
            onClick={() => {
              setSubmitted(false)
              setForm({ name: '', email: '', course_id: null, headcount: '', additional_contact: '' })
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
          <h1 className="text-3xl font-bold text-gray-900">Fenny Studio</h1>
          <p className="text-gray-600 mt-2">手作課程詢問表單</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              怎麼稱呼
            </label>
            <input
              type="text"
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
            <div className="grid grid-cols-2 gap-3">
              {courses.map((c) => (
                <button
                  key={c.id}
                  type="button"
                  onClick={() =>
                    setForm({ ...form, course_id: form.course_id === c.id ? null : c.id })
                  }
                  className={`rounded-xl border-2 overflow-hidden text-left transition cursor-pointer ${
                    form.course_id === c.id
                      ? 'border-amber-500 ring-2 ring-amber-200'
                      : 'border-gray-200 hover:border-gray-300'
                  }`}
                >
                  {c.image_path ? (
                    <img
                      src={`/uploads/${c.image_path}`}
                      alt={c.name}
                      className="w-full h-28 object-cover"
                    />
                  ) : (
                    <div className="w-full h-28 bg-gray-100 flex items-center justify-center text-gray-400 text-sm">
                      無圖片
                    </div>
                  )}
                  <div className="p-2.5">
                    <p className="font-medium text-gray-900 text-sm">{c.name}</p>
                    {c.description && (
                      <p className="text-xs text-gray-500 mt-0.5 line-clamp-2">{c.description}</p>
                    )}
                  </div>
                </button>
              ))}
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              預估人數
            </label>
            <div className="flex flex-wrap gap-2">
              {['1-5人', '6-10人', '11-20人', '21-30人', '30人以上'].map((option) => (
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
    </div>
  )
}
