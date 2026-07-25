<script>
  import { keyHue } from './api.js'
  import { describe, FLAG_LEGEND } from './rulesLang.js'
  import RuleCard from './RuleCard.svelte'

  let files = $state([])
  let dirty = $state({})      // name -> true
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

  function markDirty(file) {
    dirty = { ...dirty, [file.name]: true }
    // re-parse locally? keep it simple: chips refresh after save
  }

  function addLine(file, raw = '') {
    file.entries.push({ line: file.entries.length + 1, raw, kind: 'other' })
    markDirty(file)
  }

  function removeLine(file, idx) {
    file.entries.splice(idx, 1)
    markDirty(file)
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
      const d = { ...dirty }
      delete d[file.name]
      dirty = d
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

  const hasDirty = $derived(Object.keys(dirty).length > 0)

  // Несохранённые правки: предупреждаем при закрытии/обновлении страницы
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
        аудит: {enabled === '0' ? 'выключен' : 'включён'}
      </span>
      <span class="badge" class:ok={lost === '0'} class:bad={lost !== '0'}>lost: {lost ?? '?'}</span>
      <span class="badge" class:ok={inSync} class:warn={!inSync}>
        {inSync ? 'файлы = ядро' : 'файлы ≠ ядро (нужно применить)'}
      </span>
    {/if}
    <span class="spacer"></span>
    <label class="expert-toggle">
      <input type="checkbox" bind:checked={expert} />
      режим эксперта
    </label>
    <button
      class="apply"
      class:armed={applyArmed}
      disabled={applying || hasDirty}
      onclick={apply}
      title={hasDirty
        ? 'Сначала сохраните изменённые файлы'
        : 'augenrules --load: перезагрузит ВСЕ правила из rules.d в ядро'}
    >
      {applying ? 'применяется…'
        : hasDirty ? 'Сначала сохраните'
        : applyArmed ? 'Точно применить? (клик ещё раз)'
        : 'Применить правила'}
    </button>
  </div>

  {#if hasDirty}
    <div class="hint dirty-hint">
      ⚠ Не сохранено: <b>{Object.keys(dirty).join(', ')}</b> — нажмите
      «Сохранить» у файла, иначе изменения не попадут на диск и
      «Применить» загрузит старую версию.
    </div>
  {/if}

  {#if error}
    <div class="error">{error}</div>
  {/if}

  {#if expert}
    <details class="legend">
      <summary>Шпаргалка по флагам auditd</summary>
      <table>
        <tbody>
          {#each FLAG_LEGEND as [flag, meaning]}
            <tr><td><code>{flag}</code></td><td>{meaning}</td></tr>
          {/each}
        </tbody>
      </table>
    </details>
  {/if}

  {#if applyResult}
    <div class="result" class:ok={applyResult.ok} class:bad={!applyResult.ok}>
      <b>{applyResult.ok ? 'Применено' : 'Ошибка применения'}</b>
      {#if applyResult.output}<pre>{applyResult.output}</pre>{/if}
      {#if applyResult.loaded?.length}
        <button class="linkish" onclick={() => (showLoaded = !showLoaded)}>
          загружено правил: {applyResult.loaded.length} {showLoaded ? '▴' : '▾'}
        </button>
        {#if showLoaded}<pre>{applyResult.loaded.join('\n')}</pre>{/if}
      {/if}
    </div>
  {/if}

  {#each files as file (file.name)}
    <section class="file">
      <div class="filehead" class:unsaved={dirty[file.name]}>
        <h3>{file.name}</h3>
        {#if dirty[file.name]}
          <span class="unsaved-badge">не сохранено</span>
          <button class="save" onclick={() => save(file)}>💾 Сохранить</button>
        {/if}
        <button class="linkish" onclick={() => addLine(file)}>
          {expert ? '+ строка' : '+ правило'}
        </button>
      </div>
      {#if warnings[file.name]?.length}
        <div class="warn-box">
          {#each warnings[file.name] as wmsg}<div>⚠ {wmsg}</div>{/each}
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
                <input
                  class="raw"
                  bind:value={entry.raw}
                  oninput={() => markDirty(file)}
                  spellcheck="false"
                />
                {#if describe(entry)}
                  <div class="translate">{describe(entry).icon} {describe(entry).text}</div>
                {/if}
              </div>
              <button class="del" title="удалить строку" onclick={() => removeLine(file, i)}>✕</button>
            </div>
          {:else}
            <RuleCard
              {entry}
              onchange={(raw) => { entry.raw = raw; markDirty(file) }}
              ondelete={() => removeLine(file, i)}
            />
          {/if}
        {/each}
      </div>
    </section>
  {/each}

  <div class="newfile">
    <input placeholder="новый файл, например 10-docker.rules" bind:value={newFileName} />
    <button onclick={createFile}>Создать файл</button>
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
  .badge.warn { color: #d9a53f; }

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

  .hint, .error, .warn-box {
    margin: 8px 0;
    padding: 8px 12px;
    border-radius: 6px;
    font-size: 12px;
  }
  .hint { border: 1px solid var(--border); color: var(--fg-dim); }
  .dirty-hint {
    border-color: #d9a53f;
    color: #d9a53f;
    font-size: 13px;
    position: sticky;
    top: 0;
    background: var(--bg);
    z-index: 5;
  }
  .error { border: 1px solid var(--fail); color: var(--fail); }
  .warn-box { border: 1px solid #d9a53f; color: #d9a53f; }

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
    border-bottom-color: #d9a53f;
    background: color-mix(in srgb, #d9a53f 7%, transparent);
  }
  .unsaved-badge {
    color: #d9a53f;
    font-size: 11px;
    animation: pulse 1.2s ease-in-out infinite;
  }
  @keyframes pulse {
    50% { opacity: 0.4; }
  }
  .save {
    background: var(--ok);
    border-color: var(--ok);
    color: #fff;
    font-weight: bold;
  }
  .save:hover { filter: brightness(1.15); background: var(--ok); }

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
