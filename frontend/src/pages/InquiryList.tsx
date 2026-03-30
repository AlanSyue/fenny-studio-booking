import { useState, useEffect } from 'react'
import { apiFetch } from '../api/client'
import type { Inquiry } from '../types'
import AdminLayout from '../components/AdminLayout'

export default function InquiryList() {
  const [inquiries, setInquiries] = useState<Inquiry[]>([])
  const [loading, setLoading] = useState(true)
  const [filter, setFilter] = useState<'all' | 'pending' | 'processed'>('all')

  const fetchInquiries = async () => {
    try {
      const data = await apiFetch<Inquiry[]>('/api/admin/inquiries')
      setInquiries(data)
    } catch {
      // handled by apiFetch
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchInquiries()
  }, [])

  const toggleProcessed = async (id: number) => {
    try {
      await apiFetch(`/api/admin/inquiries/${id}`, { method: 'PATCH' })
      fetchInquiries()
    } catch {
      alert('更新失敗')
    }
  }

  const filtered = inquiries.filter((inq) => {
    if (filter === 'pending') return !inq.is_processed
    if (filter === 'processed') return inq.is_processed
    return true
  })

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString('zh-TW', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  return (
    <AdminLayout>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold text-gray-900">詢問列表</h1>
        <div className="flex gap-2">
          {(['all', 'pending', 'processed'] as const).map((f) => (
            <button
              key={f}
              onClick={() => setFilter(f)}
              className={`px-3 py-1.5 rounded-lg text-sm font-medium transition cursor-pointer ${
                filter === f
                  ? 'bg-indigo-600 text-white'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
              }`}
            >
              {f === 'all' ? '全部' : f === 'pending' ? '未處理' : '已處理'}
            </button>
          ))}
        </div>
      </div>

      {loading ? (
        <p className="text-gray-500">載入中...</p>
      ) : filtered.length === 0 ? (
        <div className="text-center py-12 text-gray-500">
          <p>沒有詢問記錄</p>
        </div>
      ) : (
        <div className="bg-white rounded-xl shadow overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="bg-gray-50 border-b border-gray-200">
                  <th className="text-left px-4 py-3 text-xs font-medium text-gray-500 uppercase">狀態</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-gray-500 uppercase">稱呼</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-gray-500 uppercase">Email</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-gray-500 uppercase">課程</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-gray-500 uppercase">人數</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-gray-500 uppercase">其他聯絡方式</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-gray-500 uppercase">提交時間</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {filtered.map((inq) => (
                  <tr key={inq.id} className={inq.is_processed ? 'bg-gray-50' : ''}>
                    <td className="px-4 py-3">
                      <button
                        onClick={() => toggleProcessed(inq.id)}
                        className={`px-2.5 py-1 rounded-full text-xs font-medium transition cursor-pointer ${
                          inq.is_processed
                            ? 'bg-green-100 text-green-700 hover:bg-green-200'
                            : 'bg-yellow-100 text-yellow-700 hover:bg-yellow-200'
                        }`}
                      >
                        {inq.is_processed ? '已處理' : '未處理'}
                      </button>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-900">{inq.name || '-'}</td>
                    <td className="px-4 py-3 text-sm text-gray-900">{inq.email}</td>
                    <td className="px-4 py-3 text-sm text-gray-900">{inq.course_name || '-'}</td>
                    <td className="px-4 py-3 text-sm text-gray-900">{inq.headcount ?? '-'}</td>
                    <td className="px-4 py-3 text-sm text-gray-600 max-w-xs truncate">{inq.additional_contact || '-'}</td>
                    <td className="px-4 py-3 text-sm text-gray-500">{formatDate(inq.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </AdminLayout>
  )
}
