import { useEffect, useMemo, useState } from 'react'
import type { Course } from '../types'
import ImageLightbox from './ImageLightbox'

interface Props {
  open: boolean
  onClose: () => void
  courses: Course[]
  selectedIds: number[]
  onToggle: (id: number) => void
}

export default function CoursePickerModal({
  open,
  onClose,
  courses,
  selectedIds,
  onToggle,
}: Props) {
  const [query, setQuery] = useState('')
  const [lightboxSrc, setLightboxSrc] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = prev
    }
  }, [open])

  useEffect(() => {
    if (!open) return
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !lightboxSrc) onClose()
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [open, onClose, lightboxSrc])

  const filteredCourses = useMemo(
    () =>
      courses.filter((c) =>
        c.name.toLowerCase().includes(query.toLowerCase())
      ),
    [courses, query]
  )

  if (!open) return null

  return (
    <>
      <div
        className="fixed inset-0 z-50 bg-black/50 flex items-end md:items-center justify-center md:p-4"
        onClick={onClose}
      >
        <div
          className="relative bg-white w-full md:max-w-2xl flex flex-col rounded-t-2xl md:rounded-2xl max-h-[90vh] md:max-h-[80vh]"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="flex items-center justify-between p-4 border-b border-gray-200">
            <h2 className="text-lg font-semibold text-gray-900">選擇課程</h2>
            <button
              type="button"
              onClick={onClose}
              className="text-gray-400 hover:text-gray-600 text-2xl leading-none cursor-pointer"
              aria-label="關閉"
            >
              ×
            </button>
          </div>

          <div className="p-4 border-b border-gray-200">
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="搜尋課程名稱…"
              className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-gray-900 focus:border-amber-500 focus:ring-2 focus:ring-amber-200 outline-none transition"
            />
          </div>

          <div className="flex-1 overflow-y-auto">
            {filteredCourses.length === 0 ? (
              <div className="p-8 text-center text-gray-500 text-sm">
                沒有符合的課程
              </div>
            ) : (
              <div className="p-2 space-y-1">
                {filteredCourses.map((c) => {
                  const selected = selectedIds.includes(c.id)
                  return (
                    <div
                      key={c.id}
                      role="button"
                      tabIndex={0}
                      onClick={() => onToggle(c.id)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault()
                          onToggle(c.id)
                        }
                      }}
                      className={`w-full flex items-center gap-3 p-3 rounded-lg text-left transition cursor-pointer ${
                        selected
                          ? 'bg-amber-50 border border-amber-400'
                          : 'border border-transparent hover:bg-gray-50'
                      }`}
                    >
                      <span
                        className={`flex-shrink-0 w-5 h-5 rounded border-2 flex items-center justify-center ${
                          selected
                            ? 'bg-amber-500 border-amber-500 text-white'
                            : 'border-gray-300'
                        }`}
                      >
                        {selected && <span className="text-xs leading-none">✓</span>}
                      </span>
                      {c.image_path ? (
                        <button
                          type="button"
                          onClick={(e) => {
                            e.stopPropagation()
                            setLightboxSrc(`/uploads/${c.image_path}`)
                          }}
                          aria-label="查看大圖"
                          className="flex-shrink-0 cursor-zoom-in hover:opacity-90 transition"
                        >
                          <img
                            src={`/uploads/${c.image_path}`}
                            alt={c.name}
                            className="w-14 h-14 rounded object-cover flex-shrink-0"
                          />
                        </button>
                      ) : (
                        <div className="w-14 h-14 rounded bg-gray-100 flex-shrink-0" />
                      )}
                      <div className="flex-1 min-w-0">
                        <p className="font-medium text-gray-900 text-sm truncate">
                          {c.name}
                        </p>
                        {c.product_url && (
                          <a
                            href={c.product_url}
                            target="_blank"
                            rel="noopener noreferrer"
                            onClick={(e) => e.stopPropagation()}
                            className="inline-block mt-0.5 text-xs text-amber-600 hover:text-amber-700"
                          >
                            看完整介紹 ↗
                          </a>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </div>

          <div className="p-4 border-t border-gray-200">
            <button
              type="button"
              onClick={onClose}
              className="w-full bg-amber-500 hover:bg-amber-600 text-white font-medium py-3 rounded-lg transition cursor-pointer"
            >
              完成（已選 {selectedIds.length} 門）
            </button>
          </div>
        </div>
      </div>
      <ImageLightbox src={lightboxSrc} alt="課程圖片" onClose={() => setLightboxSrc(null)} />
    </>
  )
}
