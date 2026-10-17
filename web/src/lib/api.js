import { t } from './i18n.svelte.js'

export async function fetchEvents(filters, beforeId = null) {
  const params = new URLSearchParams()
  for (const [k, v] of Object.entries(filters)) {
    if (v) params.set(k, v)
  }
  if (beforeId) params.set('before_id', beforeId)
  params.set('limit', '200')
  const res = await fetch(`/api/events?${params}`)
  if (!res.ok) throw new Error(`API ${res.status}`)
  return res.json()
}

export async function fetchMeta() {
  const res = await fetch('/api/meta')
  if (!res.ok) throw new Error(`API ${res.status}`)
  return res.json()
}

export async function fetchEventDetail(id) {
  const res = await fetch(`/api/event/${id}`)
  if (!res.ok) throw new Error(`API ${res.status}`)
  return res.json()
}

export function openStream(onEvent) {
  const es = new EventSource('/api/stream')
  es.onmessage = (msg) => {
    try {
      onEvent(JSON.parse(msg.data))
    } catch {
      /* ignore malformed frames */
    }
  }
  return es
}

// Append a never-rule to the noise exclusions file (created if missing).
// Returns the file name it landed in.
export async function appendNoiseRule(line) {
  const name = '00-noise-exclusions.rules'
  let content = t('noise.header') + '\n'
  const res = await fetch(`/api/rules/${name}`)
  if (res.ok) content = (await res.json()).content
  if (!content.endsWith('\n')) content += '\n'
  content += line + '\n'
  const put = await fetch(`/api/rules/${name}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content }),
  })
  if (!put.ok) throw new Error(await put.text())
  return name
}

// Stable hue per audit key so chips keep their color between renders.
export function keyHue(key) {
  let h = 0
  for (let i = 0; i < key.length; i++) h = (h * 31 + key.charCodeAt(i)) % 360
  return h
}
