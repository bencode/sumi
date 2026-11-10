<script lang="ts">
  import { onMount } from 'svelte'
  import { Compartment, EditorState } from '@codemirror/state'
  import { EditorView, basicSetup } from 'codemirror'
  import { setDiagnostics, type Diagnostic } from '@codemirror/lint'
  import type { RenderError } from '../lib/render'

  type Props = {
    source: string
    label: string
    dark: boolean
    error: RenderError | null
    onchange: (source: string) => void
    oncomposition: (active: boolean) => void
  }

  let { source, label, dark, error, onchange, oncomposition }: Props = $props()
  let host: HTMLDivElement
  let view = $state.raw<EditorView>()
  const appearance = new Compartment()

  function theme(isDark: boolean) {
    return EditorView.theme({
      '&': { backgroundColor: 'var(--surface)', color: 'var(--ink)' },
      '.cm-content': { fontFamily: 'var(--mono)', padding: '12px 0', caretColor: 'var(--ink)' },
      '.cm-gutters': { backgroundColor: 'var(--surface)', color: 'var(--muted)', border: 'none' },
      '.cm-activeLine, .cm-activeLineGutter': { backgroundColor: 'var(--editor-line)' },
      '.cm-selectionBackground, &.cm-focused .cm-selectionBackground': { backgroundColor: 'var(--selection)' },
      '&.cm-focused': { outline: 'none' },
      '.cm-scroller': { fontFamily: 'var(--mono)', lineHeight: '1.85', overflow: 'auto' },
    }, { dark: isDark })
  }

  onMount(() => {
    const editor = new EditorView({
      parent: host,
      state: EditorState.create({
        doc: source,
        extensions: [
          basicSetup,
          EditorView.contentAttributes.of({ 'aria-label': label, spellcheck: 'false' }),
          appearance.of(theme(dark)),
          EditorView.updateListener.of(update => {
            if (update.docChanged) onchange(update.state.doc.toString())
          }),
          EditorView.domEventHandlers({
            compositionstart() { oncomposition(true); return false },
            compositionend() { oncomposition(false); return false },
          }),
        ],
      }),
    })
    view = editor
    return () => editor.destroy()
  })

  $effect(() => {
    if (view && view.state.doc.toString() !== source) {
      view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: source } })
    }
  })

  $effect(() => {
    if (view) view.dispatch({ effects: appearance.reconfigure(theme(dark)) })
  })

  $effect(() => {
    if (!view) return
    const diagnostics: Diagnostic[] = []
    if (error?.line !== undefined) {
      const line = view.state.doc.line(Math.max(1, Math.min(error.line, view.state.doc.lines)))
      const offset = Array.from(line.text).slice(0, Math.max(0, (error.column ?? 1) - 1)).join('').length
      const from = line.from + offset
      const character = Array.from(line.text.slice(offset))[0] ?? ''
      diagnostics.push({ from, to: from + character.length, severity: 'error', message: error.message })
    }
    view.dispatch(setDiagnostics(view.state, diagnostics))
  })
</script>

<div class="editor" bind:this={host}></div>
