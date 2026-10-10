<script lang="ts">
  import {onMount} from 'svelte'
  import {GetSettings} from '../wailsjs/go/main/App'
  import {lang, t, type Key} from './lib/i18n'
  import Dashboard from './pages/Dashboard.svelte'
  import Settings from './pages/Settings.svelte'
  import Network from './pages/Network.svelte'
  import ComingSoon from './pages/ComingSoon.svelte'

  type Page = {id: string; label: Key; phase?: string}
  const pages: Page[] = [
    {id: 'dashboard', label: 'nav.dashboard'},
    {id: 'network', label: 'nav.network'},
    {id: 'bosses', label: 'nav.bosses', phase: 'Phase 2'},
    {id: 'relics', label: 'nav.relics', phase: 'Phase 3'},
    {id: 'timer', label: 'nav.timer', phase: 'Phase 4'},
    {id: 'settings', label: 'nav.settings'},
  ]
  let current = pages[0]

  onMount(async () => {
    try {
      const s = await GetSettings()
      lang.set(s.language === 'en' ? 'en' : 'vi')
    } catch {
      // Startup error; the dashboard shows it.
    }
  })
</script>

<div class="shell">
  <nav>
    <div class="brand">
      <span class="moon">☾</span>
      <div>
        <strong>Nightreign</strong>
        <small>Companion</small>
      </div>
    </div>
    {#each pages as p}
      <button class:active={current.id === p.id} on:click={() => (current = p)}>
        {$t(p.label)}
        {#if p.phase}<span class="tag">{p.phase}</span>{/if}
      </button>
    {/each}
  </nav>

  <main>
    {#if current.id === 'dashboard'}
      <Dashboard />
    {:else if current.id === 'network'}
      <Network />
    {:else if current.id === 'settings'}
      <Settings />
    {:else}
      <ComingSoon title={$t(current.label)} phase={current.phase ?? ''} />
    {/if}
  </main>
</div>

<style>
  .shell {
    display: grid;
    grid-template-columns: 220px 1fr;
    height: 100vh;
  }
  nav {
    background: var(--surface);
    border-right: 1px solid var(--border);
    padding: 16px 10px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 4px 10px 18px;
  }
  .brand strong {
    display: block;
    color: var(--accent);
    letter-spacing: 0.04em;
  }
  .brand small {
    color: var(--muted);
  }
  .moon {
    font-size: 28px;
    color: var(--accent);
  }
  nav button {
    display: flex;
    justify-content: space-between;
    align-items: center;
    background: none;
    border: none;
    color: var(--text);
    text-align: left;
    padding: 9px 12px;
    border-radius: 6px;
    cursor: pointer;
    font: inherit;
  }
  nav button:hover {
    background: var(--hover);
  }
  nav button.active {
    background: var(--hover);
    color: var(--accent);
  }
  .tag {
    font-size: 10px;
    color: var(--muted);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 1px 5px;
  }
  main {
    overflow-y: auto;
    padding: 28px 32px;
  }
</style>
