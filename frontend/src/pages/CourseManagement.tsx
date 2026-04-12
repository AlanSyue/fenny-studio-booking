import { useState, useEffect } from 'react'
import { apiFetch } from '../api/client'
import type { Course } from '../types'
import AdminLayout from '../components/AdminLayout'

export default function CourseManagement() {
  const [courses, setCourses] = useState<Course[]>([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)
  const [formName, setFormName] = useState('')
  const [formDescription, setFormDescription] = useState('')
  const [formProductURL, setFormProductURL] = useState('')
  const [syncing, setSyncing] = useState(false)
  const [formImage, setFormImage] = useState<File | null>(null)
  const [imagePreview, setImagePreview] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [imageMode, setImageMode] = useState<'upload' | 'url'>('upload')
  const [formURL, setFormURL] = useState('')
  const [fetchingImage, setFetchingImage] = useState(false)
  const [formImagePath, setFormImagePath] = useState<string | null>(null)
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'inactive'>('active')

  const fetchCourses = async () => {
    try {
      const data = await apiFetch<Course[]>('/api/admin/courses')
      setCourses(data)
    } catch {
      // handled by apiFetch
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchCourses()
  }, [])

  const resetForm = () => {
    setFormName('')
    setFormDescription('')
    setFormProductURL('')
    setFormImage(null)
    setImagePreview(null)
    setEditingId(null)
    setShowForm(false)
    setImageMode('upload')
    setFormURL('')
    setFormImagePath(null)
  }

  const handleSyncWoo = async () => {
    if (!confirm('確定要從 flowers.fenny-studio.com 同步材料包嗎？')) return
    setSyncing(true)
    try {
      const data = await apiFetch<{
        synced: number
        created: number
        updated: number
        deactivated?: number
        skipped?: number
        category?: string
      }>('/api/admin/courses/sync-woocommerce', { method: 'POST' })
      const deactivated = data.deactivated ?? 0
      const skipped = data.skipped ?? 0
      const category = data.category ?? ''
      if (data.category != null && data.deactivated != null) {
        const skippedSegment = skipped > 0 ? `，略過 ${skipped}` : ''
        alert(
          `已同步 ${data.synced} 筆（新增 ${data.created}，更新 ${data.updated}，停用 ${deactivated}${skippedSegment}）— 分類：${category}`
        )
      } else {
        alert(`已同步 ${data.synced} 筆（新增 ${data.created}，更新 ${data.updated}）`)
      }
      fetchCourses()
    } catch (err) {
      const message = err instanceof Error ? err.message : ''
      let display = '同步失敗'
      if (message) {
        try {
          const parsed = JSON.parse(message)
          if (parsed && typeof parsed.error === 'string') {
            display = parsed.error
          } else {
            display = message
          }
        } catch {
          display = message
        }
      }
      alert(display)
    } finally {
      setSyncing(false)
    }
  }

  const handleImageChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      setFormImage(file)
      setImagePreview(URL.createObjectURL(file))
    }
  }

  const handleFetchImage = async () => {
    if (!formURL.trim()) return
    setFetchingImage(true)
    try {
      const data = await apiFetch<{ image_path: string }>('/api/admin/fetch-image', {
        method: 'POST',
        body: JSON.stringify({ url: formURL }),
      })
      setFormImagePath(data.image_path)
      setImagePreview(`/uploads/${data.image_path}`)
    } catch {
      alert('無法從網址取得圖片')
    } finally {
      setFetchingImage(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!formName.trim()) return

    setSubmitting(true)
    const formData = new FormData()
    formData.append('name', formName)
    formData.append('description', formDescription)
    formData.append('product_url', formProductURL)
    if (formImage) {
      formData.append('image', formImage)
    }
    if (formImagePath && imageMode === 'url') {
      formData.append('image_path', formImagePath)
    }

    try {
      if (editingId) {
        await apiFetch(`/api/admin/courses/${editingId}`, {
          method: 'PUT',
          body: formData,
        })
      } else {
        await apiFetch('/api/admin/courses', {
          method: 'POST',
          body: formData,
        })
      }
      resetForm()
      fetchCourses()
    } catch {
      alert('儲存失敗')
    } finally {
      setSubmitting(false)
    }
  }

  const handleEdit = (course: Course) => {
    setEditingId(course.id)
    setFormName(course.name)
    setFormDescription(course.description || '')
    setFormProductURL(course.product_url || '')
    setFormImage(null)
    setImagePreview(course.image_path ? `/uploads/${course.image_path}` : null)
    setFormImagePath(course.image_path)
    setShowForm(true)
  }

  const handleDelete = async (id: number) => {
    if (!confirm('確定要刪除此課程嗎？')) return
    try {
      await apiFetch(`/api/admin/courses/${id}`, { method: 'DELETE' })
      fetchCourses()
    } catch {
      alert('刪除失敗')
    }
  }

  const visibleCourses = courses.filter((c) => {
    if (statusFilter === 'active') return c.is_active
    if (statusFilter === 'inactive') return !c.is_active
    return true
  })

  return (
    <AdminLayout>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold text-gray-900">課程管理</h1>
        <div className="flex gap-2 items-center">
          <span className="text-sm text-gray-400">
            (顯示 {visibleCourses.length} / 共 {courses.length})
          </span>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value as 'all' | 'active' | 'inactive')}
            className="border border-gray-300 rounded-lg px-3 py-2 text-sm"
          >
            <option value="all">全部</option>
            <option value="active">僅顯示上架中</option>
            <option value="inactive">僅顯示已停用</option>
          </select>
          <button
            onClick={handleSyncWoo}
            disabled={syncing}
            className="bg-emerald-600 hover:bg-emerald-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition disabled:opacity-50 cursor-pointer"
          >
            {syncing ? '同步中...' : '從官網同步材料包'}
          </button>
          <button
            onClick={() => {
              resetForm()
              setShowForm(true)
            }}
            className="bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition cursor-pointer"
          >
            新增課程
          </button>
        </div>
      </div>

      {showForm && (
        <div className="bg-white rounded-xl shadow p-6 mb-6">
          <h2 className="text-lg font-semibold mb-4">
            {editingId ? '編輯課程' : '新增課程'}
          </h2>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">課程名稱</label>
              <input
                type="text"
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-gray-900 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200 outline-none transition"
                placeholder="輸入課程名稱"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">課程說明</label>
              <textarea
                value={formDescription}
                onChange={(e) => setFormDescription(e.target.value)}
                rows={3}
                className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-gray-900 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200 outline-none transition resize-none"
                placeholder="輸入課程說明（選填）"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">芬妮官網商品連結</label>
              <input
                type="text"
                value={formProductURL}
                onChange={(e) => setFormProductURL(e.target.value)}
                className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-gray-900 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200 outline-none transition"
                placeholder="https://flowers.fenny-studio.com/..."
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">課程圖片</label>
              <div className="flex gap-2 mb-2">
                <button
                  type="button"
                  onClick={() => setImageMode('upload')}
                  className={`px-3 py-1 rounded-lg text-sm font-medium transition cursor-pointer ${
                    imageMode === 'upload'
                      ? 'bg-indigo-600 text-white'
                      : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                  }`}
                >
                  上傳圖片
                </button>
                <button
                  type="button"
                  onClick={() => setImageMode('url')}
                  className={`px-3 py-1 rounded-lg text-sm font-medium transition cursor-pointer ${
                    imageMode === 'url'
                      ? 'bg-indigo-600 text-white'
                      : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                  }`}
                >
                  從網址抓圖
                </button>
              </div>
              {imageMode === 'upload' ? (
                <input
                  type="file"
                  accept="image/*"
                  onChange={handleImageChange}
                  className="block w-full text-sm text-gray-500 file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:font-medium file:bg-indigo-50 file:text-indigo-600 hover:file:bg-indigo-100 file:cursor-pointer"
                />
              ) : (
                <div className="flex gap-2">
                  <input
                    type="text"
                    value={formURL}
                    onChange={(e) => setFormURL(e.target.value)}
                    className="flex-1 rounded-lg border border-gray-300 px-4 py-2.5 text-gray-900 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200 outline-none transition"
                    placeholder="輸入網頁網址，自動抓取首圖"
                  />
                  <button
                    type="button"
                    onClick={handleFetchImage}
                    disabled={fetchingImage}
                    className="bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2.5 rounded-lg text-sm font-medium transition disabled:opacity-50 cursor-pointer whitespace-nowrap"
                  >
                    {fetchingImage ? '抓取中...' : '抓取圖片'}
                  </button>
                </div>
              )}
              {imagePreview && (
                <img
                  src={imagePreview}
                  alt="Preview"
                  className="mt-3 h-32 w-32 object-cover rounded-lg border"
                />
              )}
            </div>
            <div className="flex gap-3">
              <button
                type="submit"
                disabled={submitting}
                className="bg-indigo-600 hover:bg-indigo-700 text-white px-6 py-2 rounded-lg text-sm font-medium transition disabled:opacity-50 cursor-pointer"
              >
                {submitting ? '儲存中...' : '儲存'}
              </button>
              <button
                type="button"
                onClick={resetForm}
                className="bg-gray-100 hover:bg-gray-200 text-gray-700 px-6 py-2 rounded-lg text-sm font-medium transition cursor-pointer"
              >
                取消
              </button>
            </div>
          </form>
        </div>
      )}

      {loading ? (
        <p className="text-gray-500">載入中...</p>
      ) : courses.length === 0 ? (
        <div className="text-center py-12 text-gray-500">
          <p>尚未建立任何課程</p>
        </div>
      ) : visibleCourses.length === 0 ? (
        <div className="text-center py-12 text-gray-500">
          <p>沒有符合篩選條件的課程</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {visibleCourses.map((course) => (
            <div
              key={course.id}
              className={`bg-white rounded-xl shadow overflow-hidden ${!course.is_active ? 'opacity-60' : ''}`}
            >
              {course.image_path ? (
                <img
                  src={`/uploads/${course.image_path}`}
                  alt={course.name}
                  className="w-full h-40 object-cover"
                />
              ) : (
                <div className="w-full h-40 bg-gray-100 flex items-center justify-center text-gray-400">
                  無圖片
                </div>
              )}
              <div className="p-4">
                <div className="flex justify-between items-start">
                  <div>
                    <h3 className="font-semibold text-gray-900 flex items-center gap-1 flex-wrap">
                      <span>{course.name}</span>
                      {course.product_url && (
                        <a
                          href={course.product_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-indigo-600 hover:text-indigo-800 text-sm"
                          title="開啟商品頁"
                        >
                          ↗
                        </a>
                      )}
                      {!course.is_active && (
                        <span className="ml-1 text-xs px-2 py-0.5 rounded-full bg-gray-200 text-gray-600">
                          已停用
                        </span>
                      )}
                    </h3>
                    {course.description && (
                      <p className="text-sm text-gray-500 mt-1">{course.description}</p>
                    )}
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={() => handleEdit(course)}
                      className="text-sm text-indigo-600 hover:text-indigo-800 cursor-pointer"
                    >
                      編輯
                    </button>
                    {course.is_active && (
                      <button
                        onClick={() => handleDelete(course.id)}
                        className="text-sm text-red-500 hover:text-red-700 cursor-pointer"
                      >
                        刪除
                      </button>
                    )}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </AdminLayout>
  )
}
