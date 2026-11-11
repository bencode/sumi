export type RenderResponse = {
  lightSVG: string
  darkSVG: string
  width: number
  height: number
}

export type RenderError = {
  message: string
  line?: number
  column?: number
  unsupported?: boolean
}

export type RenderResult =
  | { ok: true; data: RenderResponse }
  | { ok: false; error: RenderError }

// width is how wide a gantt diagram is drawn: the preview's, as Mermaid fills its element.
export async function renderSource(source: string, width: number, signal: AbortSignal): Promise<RenderResult> {
  const response = await fetch('/api/render', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ source, width }),
    signal,
  })
  if (response.ok) return { ok: true, data: await response.json() as RenderResponse }
  const contentType = response.headers.get('Content-Type') ?? ''
  if (!contentType.includes('application/json')) {
    throw new Error(`The rendering service returned HTTP ${response.status}.`)
  }
  const error = await response.json() as RenderError
  if (typeof error.message !== 'string') throw new Error('The rendering service returned an invalid response.')
  return { ok: false, error }
}
