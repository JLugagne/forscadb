import { useEffect, useRef, type ReactNode } from 'react'
import { X } from 'lucide-react'

interface ModalProps {
  open: boolean
  onClose: () => void
  title: string
  subtitle?: string
  children: ReactNode
  width?: string
}

export function Modal({ open, onClose, title, subtitle, children, width = 'max-w-md' }: ModalProps) {
  const overlayRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    if (open) document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [open, onClose])

  if (!open) return null

  return (
    <div
      ref={overlayRef}
      className="fixed inset-0 z-50 flex items-center justify-center p-6 animate-fade-in"
      onClick={e => { if (e.target === overlayRef.current) onClose() }}
    >
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" />
      <div className={`relative ${width} w-full bg-surface-1 border border-border-base rounded-xl shadow-2xl shadow-black/50 animate-scale-in`}>
        <div className="flex items-start justify-between px-5 pt-5 pb-0">
          <div>
            <h2 className="text-[15px] font-semibold text-text-primary">{title}</h2>
            {subtitle && <p className="mt-0.5 text-[12px] text-text-muted">{subtitle}</p>}
          </div>
          <button onClick={onClose} className="p-1 rounded-md text-text-muted hover:text-text-secondary transition-colors">
            <X size={16} />
          </button>
        </div>
        <div className="px-5 pb-5 pt-4">{children}</div>
      </div>
    </div>
  )
}
