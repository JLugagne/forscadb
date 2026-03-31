import { Modal } from './Modal'
import { AlertTriangle } from 'lucide-react'

interface ConfirmDialogProps {
  open: boolean
  onClose: () => void
  onConfirm: () => void
  title: string
  message: string
  confirmLabel?: string
  variant?: 'danger' | 'warning'
}

export function ConfirmDialog({ open, onClose, onConfirm, title, message, confirmLabel = 'Confirm', variant = 'danger' }: ConfirmDialogProps) {
  return (
    <Modal open={open} onClose={onClose} title={title}>
      <div className="flex gap-3 items-start mb-5">
        <div className={`p-2 rounded-lg shrink-0 ${variant === 'danger' ? 'bg-red-500/8' : 'bg-yellow-500/8'}`}>
          <AlertTriangle size={16} className={variant === 'danger' ? 'text-red-400' : 'text-yellow-400'} />
        </div>
        <p className="text-[13px] text-text-secondary leading-relaxed pt-0.5">{message}</p>
      </div>
      <div className="flex gap-2 justify-end">
        <button onClick={onClose} className="px-3 py-1.5 text-[13px] font-medium rounded-lg bg-surface-3 text-text-secondary hover:text-text-primary transition-colors">
          Cancel
        </button>
        <button
          onClick={() => { onConfirm(); onClose() }}
          className={`px-3 py-1.5 text-[13px] font-medium rounded-lg transition-colors ${
            variant === 'danger' ? 'bg-red-500/10 text-red-400 hover:bg-red-500/20' : 'bg-yellow-500/10 text-yellow-400 hover:bg-yellow-500/20'
          }`}
        >
          {confirmLabel}
        </button>
      </div>
    </Modal>
  )
}
