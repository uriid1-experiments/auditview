<script>
  import EventsView from './lib/EventsView.svelte'
  import RulesView from './lib/RulesView.svelte'
  import AlertsView from './lib/AlertsView.svelte'

  let tab = $state('events')
</script>

<header>
  <h1>auditview</h1>
  <nav>
    <button class:active={tab === 'events'} onclick={() => (tab = 'events')}>События</button>
    <button class:active={tab === 'rules'} onclick={() => (tab = 'rules')}>Правила</button>
    <button class:active={tab === 'alerts'} onclick={() => (tab = 'alerts')}>Алерты</button>
  </nav>
</header>

<!-- Вкладки не размонтируются, чтобы правки в редакторе правил
     не терялись при переключении -->
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
</style>
