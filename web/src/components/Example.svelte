<script lang="ts">
  import { onMount, untrack } from 'svelte'
  import type { Example } from '../lib/examples'
  import { renderSource, type RenderError } from '../lib/render'
  import MermaidEditor from './MermaidEditor.svelte'

  let { example, dark }: { example: Example; dark: boolean } = $props()
  let source = $state(untrack(() => example.source))
  let status = $state<'waiting' | 'rendering' | 'ready' | 'error'>('rendering')
  let error = $state<RenderError | null>(null)
  let previews = $state<{ light: string; dark: string; width: number; height: number } | null>(null)
  let notice = $state('')
  let generation = 0
  let composing = false
  let timer: ReturnType<typeof setTimeout> | undefined
  let request: AbortController | undefined
  let frame: HTMLDivElement

  // The width a diagram has in the preview before it scrolls.
  function frameWidth() {
    const style = getComputedStyle(frame)
    return Math.floor(frame.clientWidth - parseFloat(style.paddingLeft) - parseFloat(style.paddingRight))
  }

  function releasePreviews() {
    if (!previews) return
    URL.revokeObjectURL(previews.light)
    URL.revokeObjectURL(previews.dark)
  }

  function schedule() {
    clearTimeout(timer)
    if (!composing) timer = setTimeout(() => { void render() }, 400)
  }

  function change(value: string) {
    if (value === source) return
    source = value
    generation += 1
    request?.abort()
    error = null
    notice = ''
    status = 'waiting'
    schedule()
  }

  function composition(active: boolean) {
    composing = active
    if (active) {
      generation += 1
      request?.abort()
      clearTimeout(timer)
    } else {
      schedule()
    }
  }

  async function render() {
    clearTimeout(timer)
    request?.abort()
    const controller = new AbortController()
    request = controller
    const current = ++generation
    status = 'rendering'
    error = null
    try {
      const response = await renderSource(source, frameWidth(), controller.signal)
      if (current !== generation) return
      if (!response.ok) {
        error = response.error
        status = 'error'
        return
      }
      releasePreviews()
      previews = {
        light: URL.createObjectURL(new Blob([response.data.lightSVG], { type: 'image/svg+xml' })),
        dark: URL.createObjectURL(new Blob([response.data.darkSVG], { type: 'image/svg+xml' })),
        width: response.data.width,
        height: response.data.height,
      }
      status = 'ready'
    } catch (failure) {
      if (current !== generation || (failure instanceof Error && failure.name === 'AbortError')) return
      console.error('Could not render example:', failure)
      error = { message: 'Rendering service unavailable. Check the connection and retry.' }
      status = 'error'
    }
  }

  function reset() {
    change(example.source)
    void render()
  }

  async function copy() {
    try {
      await navigator.clipboard.writeText(source)
      notice = 'Source copied.'
    } catch (failure) {
      console.error('Could not copy source:', failure)
      notice = 'Copy failed. Select the source and copy with your keyboard or selection menu.'
    }
  }

  function download() {
    const url = URL.createObjectURL(new Blob([source], { type: 'text/plain;charset=utf-8' }))
    const link = document.createElement('a')
    link.href = url
    link.download = `${example.id}.mmd`
    link.click()
    setTimeout(() => URL.revokeObjectURL(url), 1000)
    notice = 'Download started.'
  }

  onMount(() => {
    void render()
    return () => {
      generation += 1
      clearTimeout(timer)
      request?.abort()
      releasePreviews()
    }
  })
</script>

<article class="example" id={example.anchor} aria-labelledby={`title-${example.anchor}`}>
  <header class="example-heading">
    <div>
      <h3 id={`title-${example.anchor}`}><a href={`#${example.anchor}`}>{example.title}<span aria-hidden="true">#</span></a></h3>
      <p>{example.description}</p>
    </div>
    <span class:ready={status === 'ready'} class:failed={status === 'error'} class="render-status" role="status">
      <span class="status-dot" aria-hidden="true"></span>
      {status === 'ready' ? 'Rendered' : status === 'error' ? 'Needs attention' : status === 'waiting' ? 'Edited' : 'Rendering…'}
    </span>
  </header>
  <div class="example-body">
    <div class="source-panel">
      <div class="panel-label"><span>Mermaid source</span><span>Editable</span></div>
      <MermaidEditor {source} {dark} {error} label={`${example.title} Mermaid source`} onchange={change} oncomposition={composition} />
      <div class="source-actions">
        <button onclick={copy}>Copy source</button>
        <button onclick={download}>Download .mmd</button>
        <button onclick={reset} disabled={source === example.source}>Reset example</button>
      </div>
      {#if notice}<p class="action-notice" role="status">{notice}</p>{/if}
    </div>
    <figure class="diagram-panel">
      <figcaption class="panel-label"><span>sumi output</span><span>SVG</span></figcaption>
      {#if error}
        <div class="render-error" role="status">
          <p>{#if error.line}<strong>Line {error.line}, column {error.column ?? 1}.</strong>{' '}{/if}{error.message}</p>
          <button onclick={() => { void render() }}>Retry rendering</button>
        </div>
      {/if}
      {#if previews && status !== 'ready'}<p class="preview-note">Showing the last successful render.</p>{/if}
      <div class="diagram-scroll" role="region" aria-label={`${example.title} diagram, scroll for full image`} bind:this={frame}>
        {#if previews}
          <a class="diagram-image" href={dark ? previews.dark : previews.light} target="_blank" rel="noreferrer" aria-label={`Open ${example.title} as a full-size SVG`}>
            <img src={dark ? previews.dark : previews.light} alt={`${example.title}. ${example.description}`} width={previews.width} height={previews.height}>
          </a>
        {:else}
          <div class="preview-placeholder">{status === 'error' ? 'No preview yet.' : 'Drawing your diagram…'}</div>
        {/if}
      </div>
    </figure>
  </div>
</article>
