<script lang="ts">
  import {onDestroy, onMount} from 'svelte'
  import {Info, OverlayRunning, ToggleOverlay} from '../../wailsjs/go/main/App'
  import {EventsOn} from '../../wailsjs/runtime/runtime'
  import type {main} from '../../wailsjs/go/models'
  import {t} from '../lib/i18n'

  let info: main.AppInfo | null = null
  let overlayOn = false
  let busy = false
  let error = ''

  const off = EventsOn('overlay:state', (on: boolean) => (overlayOn = on))
  onDestroy(off)

  onMount(async () => {
    info = await Info()
    overlayOn = await OverlayRunning()
  })

  async function toggle() {
    busy = true
    error = ''
    try {
      overlayOn = await ToggleOverlay()
    } catch (e) {
      error = String(e)
    } finally {
      busy = false
    }
  }

  $: d = info?.data
  $: allVerified =
    !!d &&
    d.relicsVerified === d.relics &&
    d.bossesVerified === d.bosses &&
    d.timersVerified === d.timerProfiles &&
    d.priorsVerified
</script>

<h1>{$t('dash.title')}</h1>

{#if info?.error}
  <div class="card danger">
    <h2>{$t('dash.error')}</h2>
    <code>{info.error}</code>
  </div>
{/if}

<div class="grid">
  <section class="card">
    <h2>{$t('dash.overlay')}</h2>
    <button class="primary" disabled={busy || !!info?.error} on:click={toggle}>
      {overlayOn ? $t('dash.overlayOn') : $t('dash.overlayOff')}
    </button>
    <p class="muted">{$t('dash.overlayHint')}</p>
    {#if error}<p class="err">{error}</p>{/if}
  </section>

  {#if d}
    <section class="card">
      <h2>{$t('dash.data')} <span class="muted">v{d.dataVersion}</span></h2>
      <ul class="stats">
        <li><span>Relic</span><b>{d.relicsVerified}/{d.relics}</b> {$t('dash.verified')}</li>
        <li><span>Boss</span><b>{d.bossesVerified}/{d.bosses}</b> {$t('dash.verified')}</li>
        <li><span>Timer</span><b>{d.timersVerified}/{d.timerProfiles}</b> {$t('dash.verified')}</li>
      </ul>
      {#if !allVerified}<p class="warn">⚠ {$t('dash.unverified')}</p>{/if}
    </section>
  {/if}

  {#if info}
    <section class="card">
      <h2>{$t('dash.app')}</h2>
      <ul class="stats">
        <li><span>Version</span><b>{info.version}</b></li>
        <li><span>{$t('dash.configDir')}</span><code>{info.configDir}</code></li>
      </ul>
    </section>
  {/if}
</div>

<style>
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 16px;
  }
  .stats {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: 6px;
  }
  .stats span {
    display: inline-block;
    min-width: 90px;
    color: var(--muted);
  }
  .danger {
    border-color: var(--danger);
    margin-bottom: 16px;
  }
  .warn {
    color: var(--warn);
    font-size: 13px;
  }
  .err {
    color: var(--danger);
  }
  code {
    word-break: break-all;
    font-size: 12px;
  }
</style>
