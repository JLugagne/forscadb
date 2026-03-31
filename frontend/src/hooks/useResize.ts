import { useState, useCallback, useRef, useEffect } from 'react'

type Direction = 'horizontal' | 'vertical'

interface UseResizeOptions {
  direction: Direction
  initialSize: number
  minSize?: number
  maxSize?: number
  inverted?: boolean
  onResize?: (size: number) => void
}

export function useResize({ direction, initialSize, minSize = 50, maxSize = 800, inverted = false, onResize }: UseResizeOptions) {
  const [size, setSize] = useState(initialSize)
  const dragging = useRef(false)
  const startPos = useRef(0)
  const startSize = useRef(0)

  const onMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    dragging.current = true
    startPos.current = direction === 'horizontal' ? e.clientX : e.clientY
    startSize.current = size
    document.body.style.cursor = direction === 'horizontal' ? 'col-resize' : 'row-resize'
    document.body.style.userSelect = 'none'
  }, [size, direction])

  useEffect(() => {
    const onMouseMove = (e: MouseEvent) => {
      if (!dragging.current) return
      const rawDelta = (direction === 'horizontal' ? e.clientX : e.clientY) - startPos.current
      const delta = inverted ? -rawDelta : rawDelta
      const next = Math.max(minSize, Math.min(maxSize, startSize.current + delta))
      setSize(next)
      onResize?.(next)
    }

    const onMouseUp = () => {
      if (!dragging.current) return
      dragging.current = false
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    }

    document.addEventListener('mousemove', onMouseMove)
    document.addEventListener('mouseup', onMouseUp)
    return () => {
      document.removeEventListener('mousemove', onMouseMove)
      document.removeEventListener('mouseup', onMouseUp)
    }
  }, [direction, minSize, maxSize, onResize])

  return { size, onMouseDown }
}
