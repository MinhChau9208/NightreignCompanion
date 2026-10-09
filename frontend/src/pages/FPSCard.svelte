<script lang="ts">
  import {StartFPS, StopFPS} from '../../wailsjs/go/main/App'
  import {t} from '../lib/i18n'
  import {fpsLevel, fpsRunning, fpsStatus} from '../lib/fps'
  import {icon, label} from '../lib/net'
  import Sparkline from '../lib/Sparkline.svelte'

  export let minFps = 50

  const status = fpsStatus()
  const running = fpsRunning()
  let busy = false
  let startError = ''

  async function start() {
    busy = true
    startError = ''
    try {
      await StartFPS()
    } catch (e) {
      startError = String(e)
    } finally {
      busy = false
    }
  }

  async function stop() {
    busy = true
    try {
      await StopFPS()
    } finally {
      busy = false
    }
  }

  $: s = $status
  $: level = fpsLevel(s, minFps)
  const fmtFps = (v: number) => `${Math.round(v)} FPS`
</script>

<section class="card">
  <header>
    <div>
      <h2>FPS</h2>
      <small class="muted">{s?.process ?? ''}{s?.pid ? ` · PID ${s.pid}` : ''}</small>
    </div>
    {#if $running && s?.state === 'running'}
      <span class="pill level-{level}">{icon(level)} {$t(label(level))}</span>
    {/if}
  </header>

  {#if !$running}
    <p class="muted">{$t('fps.intro')}</p>
    <button class="primary" disabled={busy} on:click={start}>{$t('fps.start')}</button>
    {#if startError}<p class="err">{startError}</p>{/if}
    {#if s?.state === 'error'}<p class="err">{s.error}</p>{/if}
  {:else}
    {#if !s || s.state === 'starting'}
      <p class="muted">{$t('fps.starting')}</p>
    {:else if s.state === 'waiting'}
      <p class="muted">{$t('fps.waiting')} <code>{s.process}</code>…</p>
    {:else if s.state === 'error'}
      <p class="err">{s.error}</p>
    {:else}
      <div class="hero">
        <b>{Math.round(s.fps)}</b><span class="unit">FPS</span>
      </div>
      <Sparkline history={s.history ?? []} threshold={minFps} format={fmtFps} />
      <dl>
        <div><dt>{$t('fps.avg')}</dt><dd>{s.avgFps.toFixed(1)}</dd></div>
        <div><dt>1% low</dt><dd>{s.low1Fps.toFixed(1)}</dd></div>
        <div><dt>Frametime</dt><dd>{s.frametimeMs ? `${s.frametimeMs.toFixed(2)} ms` : '—'}</dd></div>
        <div><dt>{$t('fps.maxFrametime')}</dt><dd>{s.maxFrametimeMs.toFixed(1)} ms</dd></div>
      </dl>
    {/if}
    <button class="secondary" disabled={busy} on:click={stop}>{$t('fps.stop')}</button>
  {/if}
</section>

<style>
  header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 8px;
  }
  header h2 {
    margin: 0;
  }
  .pill {
    font-size: 12px;
    border: 1px solid var(--border);
    border-radius: 999px;
    padding: 2px 10px;
    white-space: nowrap;
  }
  .pill.level-good {
    color: var(--ok);
  }
  .pill.level-warn {
    color: var(--warn);
  }
  .pill.level-bad {
    color: var(--danger);
  }
  .pill.level-unknown {
    color: var(--muted);
  }
  .hero {
    margin: 14px 0 6px;
    font-variant-numeric: tabular-nums;
  }
  .hero b {
    font-size: 34px;
    font-weight: 600;
  }
  .unit {
    margin-left: 4px;
    color: var(--muted);
  }
  dl {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px 16px;
    margin: 12px 0 14px;
  }
  dt {
    color: var(--muted);
    font-size: 12px;
  }
  dd {
    margin: 0;
    font-variant-numeric: tabular-nums;
  }
  .err {
    color: var(--danger);
    font-size: 12px;
  }
  button.secondary {
    background: none;
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 6px 14px;
    font: inherit;
    cursor: pointer;
  }
  button.secondary:hover {
    background: var(--hover);
  }
</style>
