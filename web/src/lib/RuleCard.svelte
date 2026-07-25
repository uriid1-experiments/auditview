<script>
  import { keyHue } from './api.js'
  import { PERMS, RECIPES, describe, editable, formFrom, generate } from './rulesLang.js'

  let { entry, onchange, ondelete } = $props()

  // Локальное представление: после правки формой карточка показывает
  // новое состояние до сохранения файла (сервер перепарсит при save).
  let view = $state({ ...entry })
  // Свежедобавленная пустая строка сразу открывается формой
  const isNew = entry.raw === '' && entry.kind !== 'blank' && entry.kind !== 'comment'
  let editing = $state(isNew)
  let form = $state(formFrom(entry))
  let recipe = $state(-1)

  function pickRecipe() {
    if (recipe >= 0) form = { ...RECIPES[recipe].f }
  }

  const desc = $derived(describe(view))
  const canEdit = $derived(editable(view))
  const commentText = $derived(
    view.kind === 'comment' ? view.raw.replace(/^\s*#+\s?/, '') : ''
  )

  function startEdit() {
    if (!canEdit) return
    form = formFrom(view)
    editing = true
  }

  function apply() {
    const raw = generate(form)
    if (!raw) return
    view = {
      ...view,
      kind: form.scope === 'exact' && form.action === 'always' ? 'watch' : 'syscall',
      raw,
      action: form.action,
      path: form.path.trim(),
      path_kind: form.scope === 'dir' ? 'dir' : form.action === 'never' ? 'path' : 'watch',
      perms: form.perms,
      key: form.key.trim(),
      fields: form.exe.trim() ? { exe: form.exe.trim() } : undefined,
      err: '',
    }
    editing = false
    onchange?.(raw)
  }

  // Отмена у только что добавленного правила убирает и саму строку —
  // иначе в файле остаётся пустышка, и он числится изменённым.
  function cancel() {
    if (isNew) ondelete?.()
    else editing = false
  }

  function togglePerm(c) {
    form.perms = form.perms.includes(c)
      ? form.perms.replace(c, '')
      : PERMS.map(([p]) => p).filter((p) => form.perms.includes(p) || p === c).join('')
  }
</script>

{#if view.kind === 'blank'}
  <div class="gap"></div>
{:else if view.kind === 'comment'}
  <div class="comment">{commentText}</div>
{:else}
  <div class="card" class:never={view.action === 'never'} class:editing>
    {#if !editing}
      <button class="face" class:clickable={canEdit} onclick={startEdit}
        title={canEdit ? 'нажмите, чтобы изменить' : 'правится только в режиме эксперта'}>
        {#if desc}
          <span class="icon">{desc.icon}</span>
          <span class="text">{desc.text}</span>
          {#if desc.key}
            <span class="chip" style="--hue: {keyHue(desc.key)}">{desc.key}</span>
          {/if}
        {:else}
          <span class="icon">⚙</span>
          <code class="text rawline">{view.raw}</code>
        {/if}
      </button>
      <button class="del" title="удалить правило" onclick={() => ondelete?.()}>✕</button>
    {:else}
      <div class="form">
        {#if isNew}
          <div class="row">
            <select bind:value={recipe} onchange={pickRecipe}>
              <option value={-1}>— рецепт (не обязательно) —</option>
              {#each RECIPES as r, i}
                <option value={i}>{r.label}</option>
              {/each}
            </select>
          </div>
        {/if}
        <div class="row">
          <select bind:value={form.action}>
            <option value="always">👁 Следить</option>
            <option value="never">🔇 Исключение (без событий)</option>
          </select>
          <input class="grow" placeholder="путь" bind:value={form.path} spellcheck="false" />
          <select bind:value={form.scope}>
            <option value="exact">точный путь</option>
            <option value="dir">каталог + содержимое</option>
          </select>
        </div>
        <div class="row">
          <span class="perms">
            ловить:
            {#each PERMS as [c, label]}
              <label><input type="checkbox" checked={form.perms.includes(c)}
                onchange={() => togglePerm(c)} /> {label}</label>
            {/each}
          </span>
          {#if form.action === 'always'}
            <input class="key-in" placeholder="метка, напр. cred_npm" bind:value={form.key} spellcheck="false" />
          {:else}
            <input class="grow" placeholder="программа (пусто = любая), напр. /usr/bin/konsole" bind:value={form.exe} spellcheck="false" />
          {/if}
        </div>
        <div class="row">
          <code class="preview">{generate(form) || '…'}</code>
          <button class="ok-btn" onclick={apply} disabled={!generate(form)}>Готово</button>
          <button onclick={cancel}>Отмена</button>
        </div>
      </div>
    {/if}
  </div>
{/if}

<style>
  .gap { height: 8px; }
  .comment {
    color: var(--fg-dim);
    font-size: 12px;
    padding: 8px 10px 2px;
  }

  .card {
    display: flex;
    align-items: center;
    border: 1px solid transparent;
    border-radius: 6px;
    margin: 2px 0;
  }
  .card:hover { background: var(--bg-hover); }
  .card.editing {
    border-color: var(--accent);
    background: var(--bg);
    padding: 8px;
  }

  .face {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 8px;
    border: none;
    background: none;
    padding: 5px 8px;
    text-align: left;
    cursor: default;
    min-width: 0;
  }
  .face.clickable { cursor: pointer; }
  .face:hover { background: none; }
  .icon { flex-shrink: 0; }
  .text {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .card.never .text { color: var(--fg-dim); }
  .rawline { font-size: 11px; background: none; }

  .chip {
    flex-shrink: 0;
    background: hsl(var(--hue) 40% 22%);
    color: hsl(var(--hue) 70% 78%);
    border-radius: 10px;
    padding: 1px 8px;
    font-size: 11px;
  }
  @media (prefers-color-scheme: light) {
    .chip { background: hsl(var(--hue) 60% 90%); color: hsl(var(--hue) 70% 30%); }
  }

  .del {
    border: none;
    background: none;
    color: var(--fg-dim);
    opacity: 0;
    padding: 2px 8px;
  }
  .card:hover .del { opacity: 1; }
  .del:hover { color: var(--fail); background: none; }

  .form {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .row { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
  .grow { flex: 1; min-width: 200px; }
  .key-in { width: 240px; }
  .perms { display: flex; gap: 10px; color: var(--fg-dim); align-items: center; }
  .perms label { display: flex; gap: 4px; align-items: center; cursor: pointer; white-space: nowrap; }
  .preview { flex: 1; min-width: 200px; font-size: 11px; padding: 4px 8px; }
  .ok-btn { border-color: var(--ok); color: var(--ok); }
</style>
