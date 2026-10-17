<script>
  import { keyHue } from './api.js'
  import { createColResize } from './colResize.svelte.js'
  import { fmtTime, t } from './i18n.svelte.js'

  const cols = createColResize({
    defaults: { time: 130, rule: 220, key: 140, exe: 220, object: 0, sent: 200 },
    order: ['time', 'rule', 'key', 'exe', 'object', 'sent'],
    storageKey: 'av_alert_colw',
  })

  let alerts = $state([])
  let config = $state(null)
  let testResult = $state(null)
  let testing = $state(false)
  let error = $state(null)

  async function load() {
    try {
      const [ar, cr] = await Promise.all([
        fetch('/api/alerts?limit=200'),
        fetch('/api/alerts/config'),
      ])
      if (!ar.ok) throw new Error(await ar.text())
      alerts = (await ar.json()).alerts ?? []
      if (cr.ok) config = await cr.json()
      error = null
    } catch (e) {
      error = String(e)
    }
  }

  $effect(() => {
    load()
    const t = setInterval(load, 15000)
    return () => clearInterval(t)
  })

  async function sendTest() {
    testing = true
    testResult = null
    try {
      const res = await fetch('/api/alerts/test', { method: 'POST' })
      testResult = (await res.json()).result
    } catch (e) {
      testResult = String(e)
    }
    testing = false
  }

  const channels = $derived.by(() => {
    if (!config?.channels) return []
    const c = config.channels
    return [
      [t('alerts.desktop') + (c.desktop?.user ? ` (${c.desktop.user})` : ''), c.desktop?.enabled],
      ['telegram', c.telegram?.enabled],
      [t('alerts.command'), c.command?.enabled],
    ]
  })

  const columns = $derived([
    ['time', t('col.time')],
    ['rule', t('col.rule')],
    ['key', t('col.key')],
    ['exe', t('col.process')],
    ['object', t('col.object')],
    ['sent', t('col.delivery')],
  ])
</script>

<div class="alerts">
  <div class="topbar">
    <div class="channels">
      {#each channels as [name, on]}
        <span class="badge" class:ok={on}>{name}: {on ? t('alerts.on') : t('alerts.off')}</span>
      {/each}
    </div>
    <span class="spacer"></span>
    <button onclick={sendTest} disabled={testing}>
      {testing ? t('alerts.sending') : t('alerts.test')}
    </button>
  </div>

  {#if testResult}<div class="test-result">{testResult}</div>{/if}
  {#if error}<div class="error">{error}</div>{/if}

  {#if config?.rules}
    <details class="rules-cfg">
      <summary>{t('alerts.rulesSummary', { n: config.rules.filter((r) => !r.disabled).length })}</summary>
      <ul>
        {#each config.rules as r}
          <li class:off={r.disabled}>
            <b>{r.name}</b> — {t('alerts.keys')} {r.keys.join(', ')}
            · {t('alerts.cooldown', { n: r.cooldown_sec })}
            {#if r.exclude_exe?.length} · {t('alerts.exceptExe')} {r.exclude_exe.join(', ')}{/if}
            {#if r.disabled} · <i>{t('alerts.disabled')}</i>{/if}
          </li>
        {/each}
      </ul>
    </details>
  {/if}

  <div class="tablewrap">
  <table class="log">
    <colgroup>
      <col style="width:{cols.w.time}px" />
      <col style="width:{cols.w.rule}px" />
      <col style="width:{cols.w.key}px" />
      <col style="width:{cols.w.exe}px" />
      {#if cols.w.object > 0}<col style="width:{cols.w.object}px" />{:else}<col />{/if}
      <col style="width:{cols.w.sent}px" />
    </colgroup>
    <thead>
      <tr>
        {#each columns as [k, label]}
          <th ondblclick={cols.reset} title={t('col.hint')}>
            {label}
            {#if k !== 'sent'}
              <span class="grip" onpointerdown={(e) => cols.startDrag(e, k)}></span>
            {/if}
          </th>
        {/each}
      </tr>
    </thead>
    <tbody>
      {#each alerts as a (a.id)}
        <tr>
          <td class="time">{fmtTime(a.ts)}</td>
          <td>{a.rule}</td>
          <td>{#if a.key}<span class="chip" style="--hue: {keyHue(a.key)}">{a.key}</span>{/if}</td>
          <td class="dim" title={a.exe}>{a.exe}</td>
          <td class="dim" title={a.object}>{a.object}</td>
          <td class="dim sent" title={a.sent}>{a.sent}</td>
        </tr>
      {/each}
    </tbody>
  </table>
  </div>

  {#if !alerts.length && !error}
    <div class="empty">{t('alerts.empty')}</div>
  {/if}
</div>

<style>
  .alerts { padding: 8px 16px 40px; }

  .topbar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 0;
  }
  .spacer { flex: 1; }
  .channels { display: flex; gap: 8px; }
  .badge {
    padding: 2px 10px;
    border-radius: 10px;
    background: var(--chip-bg);
    color: var(--fg-dim);
    font-size: 12px;
  }
  .badge.ok { color: var(--ok); }

  .test-result, .error {
    margin: 8px 0;
    padding: 8px 12px;
    border: 1px solid var(--border);
    border-radius: 6px;
    font-size: 12px;
  }
  .error { border-color: var(--fail); color: var(--fail); }

  .rules-cfg {
    margin: 8px 0;
    color: var(--fg-dim);
    font-size: 12px;
  }
  .rules-cfg summary { cursor: pointer; }
  .rules-cfg li.off { opacity: 0.5; }

  .log {
    width: 100%;
    border-collapse: collapse;
    table-layout: fixed;
    margin-top: 8px;
  }
  .tablewrap { overflow-x: auto; }
  .log th {
    text-align: left;
    padding: 6px 10px;
    color: var(--fg-dim);
    font-weight: normal;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    border-bottom: 1px solid var(--border);
    position: relative;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .log th:not(:last-child) {
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
  .log td {
    padding: 6px 10px;
    border-bottom: 1px solid var(--border);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .log td:not(:last-child) {
    border-right: 1px solid color-mix(in srgb, var(--border) 55%, transparent);
  }
  .time { color: var(--fg-dim); }
  .dim { color: var(--fg-dim); }
  .sent { font-size: 11px; }

  .chip {
    display: inline-block;
    background: hsl(var(--hue) 40% 22%);
    color: hsl(var(--hue) 70% 78%);
    border-radius: 10px;
    padding: 1px 8px;
    font-size: 11px;
  }
  @media (prefers-color-scheme: light) {
    .chip {
      background: hsl(var(--hue) 60% 90%);
      color: hsl(var(--hue) 70% 30%);
    }
  }

  .empty {
    text-align: center;
    color: var(--fg-dim);
    padding: 30px;
  }
</style>
