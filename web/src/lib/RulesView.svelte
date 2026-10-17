<script>
  import { keyHue } from './api.js'
  import { t } from './i18n.svelte.js'
  import { describe, flagLegend } from './rulesLang.js'
  import RuleCard from './RuleCard.svelte'

  let files = $state([])
  let onDisk = $state({})     // name -> content as last read/written
  let warnings = $state({})   // name -> [strings]
  let status = $state(null)
  let applyResult = $state(null)
  let applyArmed = $state(false)
  let applying = $state(false)
  let error = $state(null)
  let newFileName = $state('')
  let showLoaded = $state(false)
  let expert = $state(false)

  async function load() {
    try {
      const res = await fetch('/api/rules')
      if (!res.ok) throw new Error(await res.text())
      files = (await res.json()).files ?? []
      onDisk = Object.fromEntries(files.map((f) => [f.name, contentOf(f)]))
      error = null
    } catch (e) {
      error = String(e)
    }
    refreshStatus()
  }

  async function refreshStatus() {
    try {
      const res = await fetch('/api/audit/status')
      if (res.ok) status = await res.json()
    } catch { /* status is best-effort */ }
  }

  $effect(() => { load() })

  const inSync = $derived(status?.check_output?.includes('No change'))
  const enabled = $derived(status?.status?.match(/enabled (\d)/)?.[1])
  const lost = $derived(status?.status?.match(/lost (\d+)/)?.[1])

  function contentOf(file) {
    return file.entries.map((e) => e.raw).join('\n')
  }

  function addLine(file, raw = '') {
    file.entries.push({ line: file.entries.length + 1, raw, kind: 'other' })
  }

  function removeLine(file, idx) {
    file.entries.splice(idx, 1)
  }

  async function save(file) {
    try {
      const res = await fetch(`/api/rules/${encodeURIComponent(file.name)}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: contentOf(file) }),
      })
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      const i = files.findIndex((f) => f.name === file.name)
      if (i >= 0) files[i] = data.file
      warnings = { ...warnings, [file.name]: data.warnings ?? [] }
      onDisk = { ...onDisk, [file.name]: contentOf(data.file) }
      error = null
      refreshStatus()
    } catch (e) {
      error = String(e)
    }
  }

  async function createFile() {
    let name = newFileName.trim()
    if (!name) return
    if (!name.endsWith('.rules')) name += '.rules'
    try {
      const res = await fetch(`/api/rules/${encodeURIComponent(name)}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: `# ${name}\n` }),
      })
      if (!res.ok) throw new Error(await res.text())
      newFileName = ''
      await load()
    } catch (e) {
      error = String(e)
    }
  }

  async function apply() {
    if (!applyArmed) {
      applyArmed = true
      setTimeout(() => (applyArmed = false), 4000)
      return
    }
    applyArmed = false
    applying = true
    applyResult = null
    try {
      const res = await fetch('/api/audit/apply', { method: 'POST' })
      applyResult = await res.json()
    } catch (e) {
      applyResult = { ok: false, output: String(e) }
    }
    applying = false
    refreshStatus()
  }

  // Dirtiness is derived by comparing with disk, not kept as a flag:
  // cancelling an add or restoring a line clears it by itself.
  const dirtyNames = $derived(
    files.filter((f) => contentOf(f) !== onDisk[f.name]).map((f) => f.name)
  )
  const hasDirty = $derived(dirtyNames.length > 0)

  let savingAll = $state(false)

  async function saveAll() {
    savingAll = true
    for (const name of [...dirtyNames]) {
      const f = files.find((x) => x.name === name)
      if (f) await save(f)
    }
    savingAll = false
  }

  // Unsaved edits: warn before the page is closed or reloaded
  $effect(() => {
    if (!hasDirty) return
    const warn = (e) => {
      e.preventDefault()
      e.returnValue = ''
    }
    window.addEventListener('beforeunload', warn)
    return () => window.removeEventListener('beforeunload', warn)
  })
</script>

<div class="rules">
  <div class="statusbar">
    {#if status}
      <span class="badge" class:ok={enabled === '1' || enabled === '2'} class:bad={enabled === '0'}>
        {enabled === '0' ? t('rules.auditOff') : t('rules.auditOn')}
      </span>
      <span class="badge" class:ok={lost === '0'} class:bad={lost !== '0'}>{t('rules.lost', { n: lost ?? '?' })}</span>
      <span class="badge" class:ok={inSync} class:warn={!inSync}>
        {inSync ? t('rules.inSync') : t('rules.outOfSync')}
      </span>
    {/if}
    <span class="spacer"></span>
    <label class="expert-toggle">
      <input type="checkbox" bind:checked={expert} />
      {t('rules.expert')}
    </label>
    <button
      class="apply"
      class:armed={applyArmed}
      disabled={applying || hasDirty}
      onclick={apply}
      title={hasDirty
        ? t('rules.saveFirstTitle')
        : t('rules.applyTitle')}
    >
      {applying ? t('rules.applying')
        : hasDirty ? t('rules.saveFirst')
        : applyArmed ? t('rules.applyConfirm')
        : t('rules.apply')}
    </button>
  </div>

  {#if hasDirty}
    <div class="dirty-wrap">
      <div class="dirty-bar" role="status">
        <span class="dot" aria-hidden="true"></span>
        <div class="dirty-text">
          <div class="dirty-title">
            {t('rules.unsaved')}
            {#each dirtyNames as name}<code>{name}</code>{/each}
          </div>
          <div class="dirty-note">
            {t('rules.unsavedNote')}
          </div>
        </div>
        <button class="save" disabled={savingAll} onclick={saveAll}>
          {savingAll ? t('rules.saving') : t('rules.saveAll')}
        </button>
      </div>
    </div>
  {/if}

  {#if error}
    <div class="error"><b>{t('common.error')}</b> {error}</div>
  {/if}

  {#if expert}
    <details class="legend">
      <summary>{t('rules.legend')}</summary>
      <table>
        <tbody>
          {#each flagLegend() as [flag, meaning]}
            <tr><td><code>{flag}</code></td><td>{meaning}</td></tr>
          {/each}
        </tbody>
      </table>
    </details>
  {/if}

  {#if applyResult}
    <div class="result" class:ok={applyResult.ok} class:bad={!applyResult.ok}>
      <b>{applyResult.ok ? t('rules.applied') : t('rules.applyFailed')}</b>
      {#if applyResult.output}<pre>{applyResult.output}</pre>{/if}
      {#if applyResult.loaded?.length}
        <button class="linkish" onclick={() => (showLoaded = !showLoaded)}>
          {t('rules.loadedCount', { n: applyResult.loaded.length })} {showLoaded ? '▴' : '▾'}
        </button>
        {#if showLoaded}<pre>{applyResult.loaded.join('\n')}</pre>{/if}
      {/if}
    </div>
  {/if}

  {#each files as file (file.name)}
    <section class="file">
      <div class="filehead" class:unsaved={dirtyNames.includes(file.name)}>
        <h3>{file.name}</h3>
        {#if dirtyNames.includes(file.name)}
          <span class="unsaved-badge"><span class="dot" aria-hidden="true"></span>{t('rules.unsavedBadge')}</span>
          <button class="save" onclick={() => save(file)}>{t('rules.save')}</button>
        {/if}
        <button class="add" onclick={() => addLine(file)}>
          {expert ? t('rules.addLine') : t('rules.addRule')}
        </button>
      </div>
      {#if warnings[file.name]?.length}
        <div class="warn-box">
          <b>{t('rules.syntaxProblems')}</b>
          {#each warnings[file.name] as wmsg}<div class="warn-item">{wmsg}</div>{/each}
        </div>
      {/if}
      <div class="lines">
        {#each file.entries as entry, i (file.name + ':' + i)}
          {#if expert}
            <div class="entry" class:comment={entry.kind === 'comment' || entry.kind === 'blank'}>
              <div class="chips">
                {#if entry.kind === 'watch' || entry.kind === 'syscall'}
                  {#if entry.action === 'never'}<span class="chip never">never</span>{/if}
                  {#if entry.perms}<span class="chip perm">{entry.perms}</span>{/if}
                  {#if entry.key}<span class="chip key" style="--hue: {keyHue(entry.key)}" title={entry.key}>{entry.key}</span>{/if}
                  {#if entry.err}<span class="chip bad" title={entry.err}>⚠</span>{/if}
                {/if}
              </div>
              <div class="rawwrap">
                <input class="raw" bind:value={entry.raw} spellcheck="false" />
                {#if describe(entry)}
                  <div class="translate">{describe(entry).icon} {describe(entry).text}</div>
                {/if}
              </div>
              <button class="del" title={t('rules.deleteLine')} onclick={() => removeLine(file, i)}>✕</button>
            </div>
          {:else}
            <RuleCard
              {entry}
              onchange={(raw) => (entry.raw = raw)}
              ondelete={() => removeLine(file, i)}
            />
          {/if}
        {/each}
      </div>
    </section>
  {/each}

  <div class="newfile">
    <input placeholder={t('rules.newFile')} bind:value={newFileName} />
    <button onclick={createFile}>{t('rules.createFile')}</button>
  </div>
</div>

<style>
  .rules { padding: 8px 16px 40px; max-width: 1100px; }

  .statusbar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 0;
  }
  .spacer { flex: 1; }
  .badge {
    padding: 2px 10px;
    border-radius: 10px;
    background: var(--chip-bg);
    color: var(--fg-dim);
    font-size: 12px;
  }
  .badge.ok { color: var(--ok); }
  .badge.bad { color: var(--fail); }
  .badge.warn { color: var(--warn); }

  .apply { border-color: var(--accent); color: var(--accent); }
  .expert-toggle {
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--fg-dim);
    font-size: 12px;
    cursor: pointer;
    white-space: nowrap;
  }
  .legend {
    margin: 8px 0;
    color: var(--fg-dim);
    font-size: 12px;
  }
  .legend summary { cursor: pointer; }
  .legend td { padding: 2px 12px 2px 0; vertical-align: top; }
  .rawwrap { flex: 1; min-width: 0; }
  .translate {
    color: var(--fg-dim);
    font-size: 11px;
    padding: 0 6px 3px;
  }
  .apply.armed { border-color: var(--fail); color: var(--fail); }

  /* Notices: tinted surface + coloured left rail, body text in the normal
     foreground. Colour marks severity; it never carries the sentence. */
  .error, .warn-box {
    padding: 9px 12px;
    border-radius: 6px;
    font-size: 12px;
    line-height: 1.5;
    color: var(--fg);
  }
  .error {
    margin: 8px 0;
    border: 1px solid color-mix(in srgb, var(--fail) 45%, transparent);
    border-left: 3px solid var(--fail);
    background: color-mix(in srgb, var(--fail) 10%, var(--bg-alt));
  }
  .error b { color: var(--fail); margin-right: 6px; }
  .warn-box {
    margin: 8px; /* sits inside the file card */
    border: 1px solid color-mix(in srgb, var(--warn) 45%, transparent);
    border-left: 3px solid var(--warn);
    background: color-mix(in srgb, var(--warn) 10%, var(--bg-alt));
  }
  .warn-box b { color: var(--warn); }
  .warn-item { padding-left: 14px; text-indent: -14px; }
  .warn-item::before { content: '⚠ '; color: var(--warn); }

  /* The sticky wrapper carries the page background so scrolled content
     cannot show through the bar's own margin. */
  .dirty-wrap {
    position: sticky;
    top: 0;
    z-index: 5;
    padding: 8px 0;
    background: var(--bg);
  }
  .dirty-bar {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 12px;
    border-radius: 8px;
    border: 1px solid color-mix(in srgb, var(--warn) 45%, transparent);
    border-left: 3px solid var(--warn);
    background: color-mix(in srgb, var(--warn) 12%, var(--bg-alt));
    box-shadow: 0 6px 14px -8px rgb(0 0 0 / 0.5);
  }
  .dirty-text { flex: 1; min-width: 0; }
  .dirty-title {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 6px;
    font-size: 13px;
    font-weight: bold;
    color: var(--fg);
  }
  .dirty-title code {
    font-weight: normal;
    font-size: 12px;
    background: color-mix(in srgb, var(--warn) 18%, transparent);
    color: var(--warn);
  }
  .dirty-note {
    margin-top: 3px;
    font-size: 12px;
    color: var(--fg-dim);
  }
  .dirty-bar .save { white-space: nowrap; }

  .dot {
    width: 8px;
    height: 8px;
    flex: none;
    border-radius: 50%;
    background: var(--warn);
    animation: pulse 1.6s ease-in-out infinite;
  }
  @keyframes pulse {
    50% { opacity: 0.25; }
  }
  @media (prefers-reduced-motion: reduce) {
    .dot { animation: none; }
  }

  .result {
    margin: 8px 0;
    padding: 10px 14px;
    border-radius: 6px;
    border: 1px solid var(--border);
  }
  .result.ok { border-color: var(--ok); }
  .result.bad { border-color: var(--fail); }
  .result pre {
    background: var(--bg-alt);
    padding: 8px;
    border-radius: 4px;
    overflow-x: auto;
    font-size: 11px;
  }

  .file {
    margin: 14px 0;
    border: 1px solid var(--border);
    border-radius: 8px;
    background: var(--bg-alt);
  }
  .filehead {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 12px;
    border-bottom: 1px solid var(--border);
  }
  .filehead h3 { margin: 0; font-size: 13px; flex: 1; }
  .filehead.unsaved {
    border-bottom-color: color-mix(in srgb, var(--warn) 55%, transparent);
    background: color-mix(in srgb, var(--warn) 10%, transparent);
  }
  .unsaved-badge {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 2px 9px;
    border-radius: 10px;
    font-size: 11px;
    white-space: nowrap;
    color: var(--warn);
    background: color-mix(in srgb, var(--warn) 15%, transparent);
    border: 1px solid color-mix(in srgb, var(--warn) 40%, transparent);
  }
  .unsaved-badge .dot { width: 6px; height: 6px; }
  .save {
    background: var(--ok);
    border-color: var(--ok);
    color: #fff;
    font-weight: bold;
  }
  .save:hover { filter: brightness(1.15); background: var(--ok); }

  .add {
    background: var(--accent);
    border-color: var(--accent);
    color: #fff;
    font-weight: bold;
    white-space: nowrap;
  }
  .add:hover { filter: brightness(1.15); background: var(--accent); }

  .lines { padding: 6px 8px; }
  .entry {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 1px 0;
  }
  .entry.comment .raw { color: var(--fg-dim); }
  .chips {
    width: 170px;
    min-width: 170px;
    display: flex;
    flex-wrap: wrap;
    gap: 3px 4px;
    justify-content: flex-end;
    overflow: hidden;
  }
  .chip {
    font-size: 10px;
    padding: 1px 7px;
    border-radius: 8px;
    background: var(--chip-bg);
    color: var(--fg-dim);
    white-space: nowrap;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .chip.key { background: hsl(var(--hue) 40% 22%); color: hsl(var(--hue) 70% 78%); }
  @media (prefers-color-scheme: light) {
    .chip.key { background: hsl(var(--hue) 60% 90%); color: hsl(var(--hue) 70% 30%); }
  }
  .chip.never { background: color-mix(in srgb, var(--fail) 20%, transparent); color: var(--fail); }
  .chip.bad { color: var(--fail); }

  .raw {
    flex: 1;
    border: 1px solid transparent;
    background: transparent;
    padding: 3px 6px;
    font-size: 12px;
  }
  .raw:hover { border-color: var(--border); }
  .raw:focus { border-color: var(--accent); background: var(--bg); }

  .del {
    border: none;
    background: none;
    color: var(--fg-dim);
    opacity: 0;
    padding: 2px 6px;
  }
  .entry:hover .del { opacity: 1; }
  .del:hover { color: var(--fail); background: none; }

  .linkish {
    border: none;
    background: none;
    color: var(--accent);
    padding: 2px 6px;
  }
  .linkish:hover { background: none; text-decoration: underline; }

  .newfile { display: flex; gap: 8px; margin-top: 16px; }
  .newfile input { width: 320px; }
</style>
