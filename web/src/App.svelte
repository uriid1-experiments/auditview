<script>
  import EventsView from './lib/EventsView.svelte'
  import RulesView from './lib/RulesView.svelte'
  import AlertsView from './lib/AlertsView.svelte'
  import { i18n, LANGS, t } from './lib/i18n.svelte.js'

  let tab = $state('events')
</script>

<header>
  <h1>auditview</h1>
  <nav>
    <button class:active={tab === 'events'} onclick={() => (tab = 'events')}>{t('nav.events')}</button>
    <button class:active={tab === 'rules'} onclick={() => (tab = 'rules')}>{t('nav.rules')}</button>
    <button class:active={tab === 'alerts'} onclick={() => (tab = 'alerts')}>{t('nav.alerts')}</button>
  </nav>
  <span class="spacer"></span>
  <details class="settings">
    <summary title={t('settings.title')} aria-label={t('settings.title')}>⚙</summary>
    <div class="panel">
      <label>
        <span>{t('settings.language')}</span>
        <select bind:value={i18n.locale}>
          {#each LANGS as [code, name]}
            <option value={code}>{name}</option>
          {/each}
        </select>
      </label>
    </div>
  </details>
</header>

<!-- Tabs stay mounted so edits in the rules editor survive switching -->
<div style:display={tab === 'events' ? '' : 'none'}><EventsView /></div>
<div style:display={tab === 'rules' ? '' : 'none'}><RulesView /></div>
<div style:display={tab === 'alerts' ? '' : 'none'}><AlertsView /></div>

<style>
  header {
    display: flex;
    align-items: center;
    gap: 20px;
    padding: 12px 16px 4px;
  }
  h1 {
    margin: 0;
    font-size: 16px;
    color: var(--accent);
  }
  nav {
    display: flex;
    gap: 4px;
  }
  nav button {
    border: none;
    background: none;
    color: var(--fg-dim);
    padding: 5px 12px;
    border-radius: 6px;
  }
  nav button.active {
    background: var(--chip-bg);
    color: var(--fg);
  }
  .spacer { flex: 1; }

  .settings { position: relative; }
  .settings summary {
    list-style: none;
    cursor: pointer;
    color: var(--fg-dim);
    padding: 4px 8px;
    border-radius: 6px;
    font-size: 15px;
    user-select: none;
  }
  .settings summary::-webkit-details-marker { display: none; }
  .settings summary:hover, .settings[open] summary {
    background: var(--chip-bg);
    color: var(--fg);
  }
  .panel {
    position: absolute;
    right: 0;
    top: calc(100% + 6px);
    z-index: 20;
    min-width: 200px;
    padding: 10px 12px;
    border: 1px solid var(--border);
    border-radius: 8px;
    background: var(--bg-alt);
    box-shadow: 0 8px 20px -10px rgb(0 0 0 / 0.6);
  }
  .panel label {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    color: var(--fg-dim);
    font-size: 12px;
  }
</style>
