import { useRef, useEffect, useCallback } from 'react'
import { EditorView, keymap } from '@codemirror/view'
import { EditorState } from '@codemirror/state'
import { json } from '@codemirror/lang-json'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { autocompletion, closeBrackets, closeBracketsKeymap } from '@codemirror/autocomplete'
import { syntaxHighlighting, HighlightStyle, bracketMatching, foldGutter } from '@codemirror/language'
import { tags } from '@lezer/highlight'

const dataforgeJson = EditorView.theme({
  '&': {
    backgroundColor: '#1a1a26',
    color: '#f0eff2',
    fontSize: '13px',
    fontFamily: "'JetBrains Mono', monospace",
  },
  '.cm-content': { padding: '8px 0', caretColor: '#4ade80' },
  '.cm-line': { padding: '0 8px' },
  '&.cm-focused .cm-cursor': { borderLeftColor: '#4ade80' },
  '&.cm-focused .cm-selectionBackground, .cm-selectionBackground': { backgroundColor: '#6e8efb25 !important' },
  '.cm-activeLine': { backgroundColor: '#22223010' },
  '.cm-gutters': { backgroundColor: '#1a1a26', color: '#4a4a60', border: 'none', minWidth: '28px' },
  '.cm-activeLineGutter': { backgroundColor: 'transparent', color: '#6e6e82' },
  '.cm-matchingBracket': { backgroundColor: '#4ade8020', outline: '1px solid #4ade8040' },
  '&.cm-focused': { outline: 'none' },
  '.cm-foldGutter span': { color: '#6e6e82' },
  '.cm-scroller': { overflow: 'auto', scrollbarWidth: 'thin', scrollbarColor: 'rgba(255,255,255,0.08) transparent' },
}, { dark: true })

const highlighting = HighlightStyle.define([
  { tag: tags.propertyName, color: '#c4b5fd' },
  { tag: tags.string, color: '#4ade80' },
  { tag: tags.number, color: '#f9a858' },
  { tag: tags.bool, color: '#6e8efb' },
  { tag: tags.null, color: '#555566' },
  { tag: tags.punctuation, color: '#555566' },
])

interface JsonEditorProps {
  value: string
  onChange?: (value: string) => void
  height?: string
  readOnly?: boolean
}

export function JsonEditor({ value, onChange, height = '200px', readOnly = false }: JsonEditorProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const viewRef = useRef<EditorView | null>(null)
  const onChangeRef = useRef(onChange)
  onChangeRef.current = onChange

  const initView = useCallback(() => {
    if (!containerRef.current) return

    const state = EditorState.create({
      doc: value,
      extensions: [
        history(),
        closeBrackets(),
        bracketMatching(),
        autocompletion(),
        foldGutter(),
        json(),
        dataforgeJson,
        syntaxHighlighting(highlighting),
        keymap.of([...defaultKeymap, ...historyKeymap, ...closeBracketsKeymap]),
        EditorView.lineWrapping,
        EditorState.readOnly.of(readOnly),
        EditorView.updateListener.of(update => {
          if (update.docChanged) {
            onChangeRef.current?.(update.state.doc.toString())
          }
        }),
      ],
    })

    viewRef.current = new EditorView({ state, parent: containerRef.current })
    if (!readOnly) viewRef.current.focus()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [readOnly])

  useEffect(() => {
    initView()
    return () => { viewRef.current?.destroy() }
  }, [initView])

  useEffect(() => {
    const view = viewRef.current
    if (!view) return
    const current = view.state.doc.toString()
    if (current !== value) {
      view.dispatch({ changes: { from: 0, to: current.length, insert: value } })
    }
  }, [value])

  return (
    <div
      ref={containerRef}
      className="rounded-lg overflow-hidden border border-border-base"
      style={{ height }}
    />
  )
}
