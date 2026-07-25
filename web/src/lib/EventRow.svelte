<script>
  import { appendNoiseRule, fetchEventDetail, keyHue } from './api.js'

  let { event, id } = $props()

  let expanded = $state(false)
  let detail = $state(null)
  let muteOpen = $state(false)
  let muteLine = $state('')
  let muteStatus = $state('')

  function openMute() {
    const exe = event.process?.exe ?? ''
    const path = event.paths?.find((p) => p.name)?.name
      ?? event.summary?.object?.primary ?? ''
    muteLine = `-a never,exit${exe ? ` -F exe=${exe}` : ''} -F path=${path} -F perm=r`
    muteStatus = ''
    muteOpen = true
  }

  async function addMute() {
    try {
      const name = await appendNoiseRule(muteLine)
      muteStatus = `✓ Добавлено в ${name}. Осталось: вкладка «Правила» → «Применить правила».`
      muteOpen = false
    } catch (e) {
      muteStatus = 'Ошибка: ' + e
    }
  }

  async function toggle() {
    expanded = !expanded
    if (expanded && !detail) {
      try {
        detail = await fetchEventDetail(id)
      } catch {
        detail = { raw: ['(не удалось загрузить сырые записи)'] }
      }
    }
  }

  function fmtTime(ts) {
    return new Date(ts).toLocaleString('ru-RU', {
      day: '2-digit', month: '2-digit',
      hour: '2-digit', minute: '2-digit', second: '2-digit',
    })
  }

  const summary = $derived.by(() => {
    const actor = event.summary?.actor?.primary
    const parts = [
      actor && actor !== 'unset' ? actor : null,
      event.summary?.action,
      event.summary?.object?.primary,
    ].filter(Boolean)
    if (parts.length) return parts.join(' · ')
    // Keyless kernel records (BPF load/unload, service events, …)
    return [event.record_type, event.data?.op].filter(Boolean).join(' · ')
  })
</script>

<tr class="row" class:expanded onclick={toggle}>
  <td class="time">{fmtTime(event['@timestamp'])}</td>
  <td class="keys">
    {#each event.tags ?? [] as tag}
      <span class="chip" style="--hue: {keyHue(tag)}">{tag}</span>
    {/each}
  </td>
  <td class="summary" title={summary}>{summary}</td>
  <td class="exe" title={event.process?.exe}>{event.process?.exe ?? ''}</td>
  <td class="result {event.result === 'success' ? 'ok' : 'fail'}">{event.result ?? ''}</td>
</tr>

{#if expanded}
  <tr class="detail-row">
    <td colspan="5">
      <div class="detail">
        <div class="grid">
          <section>
            <h4>Процесс</h4>
            <dl>
              <dt>exe</dt><dd><code>{event.process?.exe ?? '—'}</code></dd>
              <dt>cmdline</dt><dd><code>{event.process?.title ?? '—'}</code></dd>
              <dt>pid / ppid</dt><dd>{event.process?.pid ?? '—'} / {event.process?.ppid ?? '—'}</dd>
              <dt>cwd</dt><dd><code>{event.process?.cwd ?? '—'}</code></dd>
              <dt>tty / session</dt><dd>{event.data?.tty ?? '—'} / {event.session ?? '—'}</dd>
            </dl>
          </section>
          <section>
            <h4>Пользователь</h4>
            <dl>
              <dt>uid</dt><dd>{event.user?.names?.uid ?? event.user?.ids?.uid ?? '—'}</dd>
              <dt>auid (кто залогинился)</dt><dd>{event.user?.names?.auid ?? event.user?.ids?.auid ?? '—'}</dd>
              <dt>euid</dt><dd>{event.user?.names?.euid ?? event.user?.ids?.euid ?? '—'}</dd>
            </dl>
            <h4>Syscall</h4>
            <dl>
              <dt>вызов</dt><dd>{event.data?.syscall ?? event.record_type}</dd>
              <dt>результат</dt><dd>{event.result} (exit={event.data?.exit ?? '—'})</dd>
              <dt>seq</dt><dd>{event.sequence}</dd>
            </dl>
          </section>
        </div>

        {#if event.paths?.length}
          <h4>Файлы</h4>
          <table class="paths">
            <thead><tr><th>путь</th><th>тип</th><th>mode</th><th>владелец</th><th>inode</th></tr></thead>
            <tbody>
              {#each event.paths as p}
                <tr>
                  <td><code>{p.name ?? '—'}</code></td>
                  <td>{p.nametype ?? ''}</td>
                  <td>{p.mode ?? ''}</td>
                  <td>{p.ouid ?? ''}:{p.ogid ?? ''}</td>
                  <td>{p.inode ?? ''}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}

        {#if detail?.raw?.length}
          <h4>Сырые записи</h4>
          <pre class="raw">{detail.raw.join('\n')}</pre>
        {/if}

        <div class="mute">
          {#if !muteOpen}
            <button class="mute-btn" onclick={(e) => { e.stopPropagation(); openMute() }}>
              🔇 Заглушить
            </button>
          {:else}
            <div class="mute-form">
              <div class="mute-hint">
                Эта программа больше не будет создавать события по этому пути
                (другие программы — будут). Строку можно поправить:
              </div>
              <input class="mute-line" bind:value={muteLine} spellcheck="false" />
              <div class="mute-actions">
                <button class="ok-btn" onclick={addMute}>Добавить исключение</button>
                <button onclick={() => (muteOpen = false)}>Отмена</button>
              </div>
            </div>
          {/if}
          {#if muteStatus}<div class="mute-status">{muteStatus}</div>{/if}
        </div>
      </div>
    </td>
  </tr>
{/if}

<style>
  .row {
    cursor: pointer;
    border-bottom: 1px solid var(--border);
  }
  .row:hover, .row.expanded { background: var(--bg-hover); }
  .row td {
    padding: 6px 10px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .row td:not(:last-child) {
    border-right: 1px solid color-mix(in srgb, var(--border) 55%, transparent);
  }
  .time { color: var(--fg-dim); }
  .exe { color: var(--fg-dim); }
  .ok { color: var(--ok); }
  .fail { color: var(--fail); }

  .chip {
    display: inline-block;
    background: hsl(var(--hue) 40% 22%);
    color: hsl(var(--hue) 70% 78%);
    border-radius: 10px;
    padding: 1px 8px;
    margin-right: 4px;
    font-size: 11px;
  }
  @media (prefers-color-scheme: light) {
    .chip {
      background: hsl(var(--hue) 60% 90%);
      color: hsl(var(--hue) 70% 30%);
    }
  }

  .detail-row td {
    background: var(--bg-alt);
    border-bottom: 1px solid var(--border);
    padding: 12px 16px;
  }
  .detail h4 {
    margin: 12px 0 6px;
    color: var(--fg-dim);
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 0 24px;
  }
  dl {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 3px 12px;
    margin: 0;
  }
  dt { color: var(--fg-dim); }
  dd { margin: 0; word-break: break-all; }

  .paths {
    width: 100%;
    border-collapse: collapse;
    font-size: 12px;
  }
  .paths th, .paths td {
    text-align: left;
    padding: 3px 10px 3px 0;
    border-bottom: 1px solid var(--border);
  }
  .paths th { color: var(--fg-dim); font-weight: normal; }

  .raw {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 10px;
    overflow-x: auto;
    font-size: 11px;
    line-height: 1.5;
    margin: 0;
  }

  .mute { margin-top: 12px; }
  .mute-btn { color: var(--fg-dim); }
  .mute-form {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .mute-hint { color: var(--fg-dim); font-size: 12px; }
  .mute-line { font-size: 12px; width: 100%; }
  .mute-actions { display: flex; gap: 8px; }
  .ok-btn { border-color: var(--ok); color: var(--ok); }
  .mute-status { margin-top: 6px; color: var(--ok); font-size: 12px; }
</style>
