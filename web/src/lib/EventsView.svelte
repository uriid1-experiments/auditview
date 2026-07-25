<script>
  import { fetchEvents, fetchMeta, openStream, keyHue } from './api.js'
  import { createColResize } from './colResize.svelte.js'
  import EventRow from './EventRow.svelte'

  let events = $state([])
  let total = $state(0)
  let meta = $state(null)
  let live = $state(true)
  let error = $state(null)

  let key = $state('')
  let exe = $state('')
  let q = $state('')
  let result = $state('')
  let keyedOnly = $state(true)

  const filters = $derived({ key, exe, q, result, keyed: keyedOnly ? '1' : '' })

  let es = null
  let debounce = null

  // Ширины колонок (px), перетаскиваются за границу в шапке
  const cols = createColResize({
    defaults: { time: 130, keys: 160, summary: 0, exe: 280, result: 90 },
    order: ['time', 'keys', 'summary', 'exe', 'result'],
    storageKey: 'av_colw',
  })

  async function load() {
    try {
      const data = await fetchEvents(filters)
      events = data.events ?? []
      total = data.total
      error = null
    } catch (e) {
      error = String(e)
    }
    try {
      meta = await fetchMeta()
    } catch { /* metadata is best-effort */ }
  }

  async function loadMore() {
    const last = events[events.length - 1]
    if (!last) return
    const data = await fetchEvents(filters, last.id)
    events = [...events, ...(data.events ?? [])]
  }

  function matchesFilters(ev) {
    if (keyedOnly && !(ev.tags ?? []).length) return false
    if (key && !(ev.tags ?? []).includes(key)) return false
    if (exe && !(ev.process?.exe ?? '').includes(exe)) return false
    if (result && ev.result !== result) return false
    if (q) {
      const hay = JSON.stringify(ev).toLowerCase()
      if (!hay.includes(q.toLowerCase())) return false
    }
    return true
  }

  $effect(() => {
    // re-query when any filter changes, with a small debounce for typing
    void filters
    clearTimeout(debounce)
    debounce = setTimeout(load, 250)
    return () => clearTimeout(debounce)
  })

  $effect(() => {
    if (live) {
      es = openStream((row) => {
        if (row?.ev && matchesFilters(row.ev)) {
          events = [row, ...events].slice(0, 500)
          total += 1
        }
      })
    }
    return () => { es?.close(); es = null }
  })

  function pickKey(k) {
    key = key === k ? '' : k
  }

  const sortedKeys = $derived(
    Object.entries(meta?.keys ?? {}).sort((a, b) => b[1] - a[1])
  )
</script>

<div class="toolbar">
  <span class="stats">
    {#if meta}{meta.total} событий в буфере{/if}
    {#if total !== null && (key || exe || q || result)} · найдено: {total}{/if}
  </span>
  <button class="live" class:on={live} onclick={() => (live = !live)}>
    {live ? '⏸ пауза' : '▶ live'}
  </button>
</div>

<div class="keybar">
  {#each sortedKeys as [k, count]}
    <button
      class="chip"
      class:active={key === k}
      style="--hue: {keyHue(k)}"
      onclick={() => pickKey(k)}
    >{k} <span class="count">{count}</span></button>
  {/each}
</div>

<div class="filterbar">
  <input placeholder="поиск (путь, команда, пользователь…)" bind:value={q} />
  <input placeholder="фильтр по exe" bind:value={exe} list="exes" />
  <datalist id="exes">
    {#each meta?.top_exes ?? [] as e}<option value={e}></option>{/each}
  </datalist>
  <select bind:value={result}>
    <option value="">результат: любой</option>
    <option value="success">success</option>
    <option value="fail">fail</option>
  </select>
  <label class="toggle">
    <input type="checkbox" bind:checked={keyedOnly} />
    только правила
  </label>
  {#if key || exe || q || result}
    <button onclick={() => { key = ''; exe = ''; q = ''; result = '' }}>сбросить</button>
  {/if}
</div>

{#if error}
  <div class="error">Ошибка API: {error}. Сервер запущен? (sudo ./auditview)</div>
{/if}

<div class="tablewrap">
<table class="events">
  <colgroup>
    <col style="width:{cols.w.time}px" />
    <col style="width:{cols.w.keys}px" />
    {#if cols.w.summary > 0}<col style="width:{cols.w.summary}px" />{:else}<col />{/if}
    <col style="width:{cols.w.exe}px" />
    <col style="width:{cols.w.result}px" />
  </colgroup>
  <thead>
    <tr>
      {#each [['time', 'время'], ['keys', 'ключ'], ['summary', 'событие'], ['exe', 'процесс'], ['result', 'результат']] as [k, label]}
        <th ondblclick={cols.reset} title="тяните за правую границу; двойной клик — сброс ширин">
          {label}
          {#if k !== 'result'}
            <span class="grip" onpointerdown={(e) => cols.startDrag(e, k)}></span>
          {/if}
        </th>
      {/each}
    </tr>
  </thead>
  <tbody>
    {#each events as row (row.id)}
      <EventRow event={row.ev} id={row.id} />
    {/each}
  </tbody>
</table>
</div>

{#if events.length && events.length < total}
  <div class="more">
    <button onclick={loadMore}>показать ещё ({total - events.length})</button>
  </div>
{/if}

{#if !events.length && !error}
  <div class="empty">Нет событий по текущим фильтрам</div>
{/if}

<style>
  .toolbar {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 8px 16px 4px;
  }
  .stats { color: var(--fg-dim); flex: 1; }
  .live.on { border-color: var(--ok); color: var(--ok); }

  .keybar {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    padding: 4px 16px;
  }
  .chip {
    background: hsl(var(--hue) 40% 18%);
    color: hsl(var(--hue) 70% 78%);
    border: 1px solid transparent;
    border-radius: 12px;
    padding: 2px 10px;
    font-size: 12px;
  }
  .chip.active { border-color: hsl(var(--hue) 70% 60%); }
  .chip .count { opacity: 0.65; }
  @media (prefers-color-scheme: light) {
    .chip {
      background: hsl(var(--hue) 60% 90%);
      color: hsl(var(--hue) 70% 28%);
    }
  }

  .filterbar {
    display: flex;
    gap: 8px;
    padding: 8px 16px;
  }
  .filterbar input:first-child { flex: 1; }
  .toggle {
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--fg-dim);
    white-space: nowrap;
    cursor: pointer;
  }

  .tablewrap { overflow-x: auto; }
  .events {
    width: 100%;
    border-collapse: collapse;
    table-layout: fixed;
  }
  .events th {
    text-align: left;
    padding: 6px 10px;
    color: var(--fg-dim);
    font-weight: normal;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    border-bottom: 1px solid var(--border);
    position: sticky;
    top: 0;
    background: var(--bg);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .events th:not(:last-child) {
    border-right: 1px solid var(--border);
  }
  .grip {
    position: absolute;
    top: 0;
    right: -4px;
    width: 9px;
    height: 100%;
    cursor: col-resize;
    z-index: 2;
  }
  .grip:hover {
    background: var(--accent);
    opacity: 0.6;
  }

  .error {
    margin: 12px 16px;
    padding: 10px 14px;
    border: 1px solid var(--fail);
    border-radius: 6px;
    color: var(--fail);
  }
  .empty, .more {
    text-align: center;
    color: var(--fg-dim);
    padding: 30px;
  }
</style>
