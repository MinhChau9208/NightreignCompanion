<script lang="ts">
  import {onMount} from 'svelte'
  import {GetBossData} from '../../wailsjs/go/main/App'
  import type {main} from '../../wailsjs/go/models'
  import {t, type Key} from '../lib/i18n'
  import BossPredict from './BossPredict.svelte'
  import BossStats from './BossStats.svelte'
  import BossPools from './BossPools.svelte'

  type Tab = 'predict' | 'stats' | 'pools'
  const tabs: {id: Tab; label: Key}[] = [
    {id: 'predict', label: 'boss.tabPredict'},
    {id: 'stats', label: 'boss.tabStats'},
    {id: 'pools', label: 'boss.tabPools'},
  ]
  let tab: Tab = 'predict'
  let data: main.BossData | null = null
  let error = ''
  // Nightlord shown on the stats tab; the predictor can jump to it.
  let focus = ''

  onMount(async () => {
    try {
      data = await GetBossData()
    } catch (e) {
      error = String(e)
    }
  })

  function showStats(id: string) {
    focus = id
    tab = 'stats'
  }
</script>

<h1>{$t('nav.bosses')}</h1>

<div class="tabs" role="tablist">
  {#each tabs as x}
    <button role="tab" aria-selected={tab === x.id} class:active={tab === x.id} on:click={() => (tab = x.id)}>
      {$t(x.label)}
    </button>
  {/each}
</div>

{#if error}
  <p class="err">{error}</p>
{:else if data}
  {#if tab === 'predict'}
    <BossPredict {data} on:stats={(e) => showStats(e.detail)} />
  {:else if tab === 'stats'}
    <BossStats {data} bind:focus />
  {:else}
    <BossPools {data} on:stats={(e) => showStats(e.detail)} />
  {/if}
  <p class="muted foot">{$t('boss.source')}</p>
{/if}

<style>
  .tabs {
    display: flex;
    gap: 4px;
    margin-bottom: 16px;
    border-bottom: 1px solid var(--border);
  }
  .tabs button {
    background: none;
    border: none;
    border-bottom: 2px solid transparent;
    color: var(--muted);
    font: inherit;
    padding: 8px 14px;
    cursor: pointer;
  }
  .tabs button:hover {
    color: var(--text);
  }
  .tabs button.active {
    color: var(--accent);
    border-bottom-color: var(--accent);
  }
  .err {
    color: var(--danger);
  }
  .foot {
    font-size: 12px;
    margin-top: 16px;
  }
</style>
