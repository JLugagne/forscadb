interface ResizeHandleProps {
  direction: 'horizontal' | 'vertical'
  onMouseDown: (e: React.MouseEvent) => void
}

export function ResizeHandle({ direction, onMouseDown }: ResizeHandleProps) {
  if (direction === 'horizontal') {
    return (
      <div
        onMouseDown={onMouseDown}
        className="w-[3px] shrink-0 cursor-col-resize group relative z-10 hover:bg-accent-sql/20 active:bg-accent-sql/30 transition-colors"
      >
        <div className="absolute inset-y-0 -left-[3px] -right-[3px]" />
      </div>
    )
  }

  return (
    <div
      onMouseDown={onMouseDown}
      className="h-[3px] shrink-0 cursor-row-resize group relative z-10 hover:bg-accent-sql/20 active:bg-accent-sql/30 transition-colors"
    >
      <div className="absolute inset-x-0 -top-[3px] -bottom-[3px]" />
    </div>
  )
}
