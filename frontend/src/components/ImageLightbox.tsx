import { useEffect } from 'react'

interface Props {
  src: string | null
  alt?: string
  onClose: () => void
}

export default function ImageLightbox({ src, alt = '', onClose }: Props) {
  useEffect(() => {
    if (!src) return
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handleKey)
    return () => document.removeEventListener('keydown', handleKey)
  }, [src, onClose])

  if (!src) return null

  return (
    <div
      className="fixed inset-0 z-[60] bg-black/90 flex items-center justify-center p-4 cursor-zoom-out"
      onClick={onClose}
    >
      <button
        type="button"
        onClick={onClose}
        className="absolute top-4 right-4 text-white text-3xl leading-none cursor-pointer hover:opacity-80"
        aria-label="關閉"
      >
        ×
      </button>
      <img
        src={src}
        alt={alt}
        className="max-w-full max-h-full object-contain"
        onClick={(e) => e.stopPropagation()}
      />
    </div>
  )
}
