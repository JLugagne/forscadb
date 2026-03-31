import { useRef, useEffect, useCallback } from 'react'
import { EditorView, keymap, placeholder as placeholderExt } from '@codemirror/view'
import { EditorState } from '@codemirror/state'
import { sql, PostgreSQL } from '@codemirror/lang-sql'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { autocompletion, closeBrackets, closeBracketsKeymap } from '@codemirror/autocomplete'
import { syntaxHighlighting, HighlightStyle, bracketMatching } from '@codemirror/language'
import { tags } from '@lezer/highlight'
import { searchKeymap, highlightSelectionMatches } from '@codemirror/search'
import type { SQLTable } from '../../types/database'

// Theme
const dataforgeDark = EditorView.theme({
  '&': {
    backgroundColor: '#13131c',
    color: '#f0eff2',
    fontSize: '13px',
    fontFamily: "'JetBrains Mono', monospace",
  },
  '.cm-content': { padding: '12px 0', caretColor: '#6e8efb' },
  '.cm-line': { padding: '0 8px' },
  '&.cm-focused .cm-cursor': { borderLeftColor: '#6e8efb' },
  '&.cm-focused .cm-selectionBackground, .cm-selectionBackground': { backgroundColor: '#6e8efb25 !important' },
  '.cm-activeLine': { backgroundColor: '#1a1a2610' },
  '.cm-gutters': { backgroundColor: '#13131c', color: '#4a4a60', border: 'none', minWidth: '32px' },
  '.cm-activeLineGutter': { backgroundColor: 'transparent', color: '#6e6e82' },
  '.cm-tooltip': { backgroundColor: '#1a1a26', border: '1px solid rgba(255,255,255,0.12)', borderRadius: '8px' },
  '.cm-tooltip.cm-tooltip-autocomplete > ul': { fontFamily: "'JetBrains Mono', monospace", fontSize: '12px' },
  '.cm-tooltip.cm-tooltip-autocomplete > ul > li': { padding: '3px 8px' },
  '.cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected]': { backgroundColor: '#2c2c3c', color: '#f0eff2' },
  '.cm-matchingBracket': { backgroundColor: '#6e8efb20', outline: '1px solid #6e8efb40' },
  '&.cm-focused': { outline: 'none' },
  '.cm-scroller': { overflow: 'auto', scrollbarWidth: 'thin', scrollbarColor: 'rgba(255,255,255,0.08) transparent' },
}, { dark: true })

const highlighting = HighlightStyle.define([
  { tag: tags.keyword, color: '#6e8efb', fontWeight: 'bold' },
  { tag: tags.operator, color: '#e879a8' },
  { tag: tags.string, color: '#4ade80' },
  { tag: tags.number, color: '#f9a858' },
  { tag: tags.comment, color: '#555566', fontStyle: 'italic' },
  { tag: tags.typeName, color: '#67e8f9' },
  { tag: tags.function(tags.variableName), color: '#c4b5fd' },
  { tag: tags.variableName, color: '#ededf0' },
  { tag: tags.bool, color: '#6e8efb' },
  { tag: tags.null, color: '#555566' },
  { tag: tags.punctuation, color: '#555566' },
])

interface SQLEditorProps {
  value: string
  onChange: (value: string) => void
  onRun: () => void
  height?: string
  tables?: SQLTable[]
}

export function SQLEditor({ value, onChange, onRun, height = '160px', tables = [] }: SQLEditorProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const viewRef = useRef<EditorView | null>(null)
  const onRunRef = useRef(onRun)
  onRunRef.current = onRun

  const onChangeRef = useRef(onChange)
  onChangeRef.current = onChange

  const initView = useCallback(() => {
    if (!containerRef.current) return

    const schemaObj: Record<string, string[]> = {}
    for (const t of tables) {
      schemaObj[t.name] = t.columns.map((c: { name: string }) => c.name)
    }

    const runKeymap = keymap.of([{
      key: 'Mod-Enter',
      run: () => { onRunRef.current(); return true },
    }])

    const state = EditorState.create({
      doc: value,
      extensions: [
        runKeymap,
        history(),
        closeBrackets(),
        bracketMatching(),
        autocompletion(),
        highlightSelectionMatches(),
        sql({ dialect: PostgreSQL, schema: schemaObj, upperCaseKeywords: true }),
        dataforgeDark,
        syntaxHighlighting(highlighting),
        placeholderExt('SELECT * FROM ...'),
        keymap.of([...defaultKeymap, ...historyKeymap, ...closeBracketsKeymap, ...searchKeymap]),
        EditorView.lineWrapping,
        EditorView.updateListener.of(update => {
          if (update.docChanged) {
            onChangeRef.current(update.state.doc.toString())
          }
        }),
      ],
    })

    viewRef.current = new EditorView({ state, parent: containerRef.current })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    initView()
    return () => { viewRef.current?.destroy() }
  }, [initView])

  // Sync external value changes
  useEffect(() => {
    const view = viewRef.current
    if (!view) return
    const current = view.state.doc.toString()
    if (current !== value) {
      view.dispatch({ changes: { from: 0, to: current.length, insert: value } })
    }
  }, [value])

  return (
    <div ref={containerRef} className="border-b border-border-dim" style={{ height, overflow: 'auto' }} />
  )
}
