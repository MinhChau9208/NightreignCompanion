<script lang="ts">
  import {onDestroy, onMount} from 'svelte'
  import {GetSettings} from '../wailsjs/go/main/App'
  import {EventsOn, Quit} from '../wailsjs/runtime/runtime'
  import {lang, t} from './lib/i18n'
  import {fmtMs, icon, netStats} from './lib/net'
  import {fpsLevel, fpsStatus} from './lib/fps'
  import {flowLevel, gameNet, hasPing, kind, mainFlow} from './lib/gamenet'

  const net = netStats()
  const fps = fpsStatus()
  const game = gameNet()
  let minFps = 50

  type Status = {uptimeSec: number; dataVersion: string; phase: string}

  let status: Status | null = null
  let connected = false
  let opacity = 0.85

  function applySettings(s: {language: string; overlay: {opacity: number}; network: {thresholds: {minFps: number}}}) {
    opacity = s.overlay.opacity
    minFps = s.network.thresholds.minFps
    lang.set(s.language === 'en' ? 'en' : 'vi')
  }

  const offs = [
    EventsOn('status', (s: Status) => {
      status = s
      connected = true
    }),
    EventsOn('settings', applySettings),
    EventsOn('ipc:disconnected', () => (connected = false)),
  ]
  onDestroy(() => offs.forEach((off) => off()))

  onMount(async () => {
    try {
      applySettings(await GetSettings())
    } catch {
      // Defaults are fine for the overlay.
    }
  })

  $: gf = mainFlow($game)

  function fmt(sec: number) {
    const m = Math.floor(sec / 60)
    const s = sec % 60
    return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  }
</script>

<div class="overlay" style="--o: {opacity}">
  <div class="bar">
    <span class="dot" class:ok={connected}></span>
    <span>{connected ? $t('overlay.connected') : $t('overlay.disconnected')}</span>
    {#if status}<span class="muted">· {fmt(status.uptimeSec)}</span>{/if}
    <button class="close" on:click={Quit} title="Close">×</button>
  </div>
  <ul class="net">
    {#if $fps?.state === 'running'}
      <li class="level-{fpsLevel($fps, minFps)}">
        <span class="icon">{icon(fpsLevel($fps, minFps))}</span>
        <span class="name">FPS</span>
        <span class="num">{Math.round($fps.fps)}</span>
        <span class="num loss">{Math.round($fps.low1Fps)}</span>
      </li>
    {/if}
    {#if gf}
      <li class="level-{flowLevel(gf)}">
        <span class="icon">{icon(flowLevel(gf))}</span>
        <span class="name">{$t(kind(gf.kind))}</span>
        {#if gf.ping && hasPing(gf)}
          <span class="num">{gf.ping.lastLost ? $t('net.lost') : `${fmtMs(gf.ping.lastMs)} ms`}</span>
          <span class="num loss">{gf.ping.lossPct.toFixed(0)}%</span>
        {:else}
          <span class="num">{gf.pktsInPerSec > 0 ? `${Math.round(gf.maxGapMs)} ms` : '—'}</span>
          <span class="num loss">gap</span>
        {/if}
      </li>
    {/if}
    {#each $net.slice(0, gf ? 2 : 3) as n (n.target)}
      <li class="level-{n.level}">
        <span class="icon">{icon(n.level)}</span>
        <span class="name">{n.target === 'gateway' ? 'Router' : n.target}</span>
        <span class="num">{n.sent === 0 ? '—' : n.lastLost ? $t('net.lost') : `${fmtMs(n.lastMs)} ms`}</span>
        <span class="num loss">{n.sent ? `${n.lossPct.toFixed(0)}%` : ''}</span>
      </li>
    {/each}
  </ul>
  <div class="timer">
    <span class="muted">{$t('overlay.timer')}</span>
    <strong>{status?.phase === 'idle' || !status ? $t('overlay.idle') : status.phase}</strong>
  </div>
</div>

<style>
  .overlay {
    height: 100vh;
    box-sizing: border-box;
    padding: 10px 14px;
    background: rgba(14, 16, 22, var(--o));
    border: 1px solid rgba(201, 168, 106, 0.45);
    border-radius: 10px;
    --wails-draggable: drag;
    user-select: none;
    font-size: 13px;
  }
  .bar {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--danger);
  }
  .dot.ok {
    background: var(--ok);
  }
  .close {
    margin-left: auto;
    --wails-draggable: no-drag;
    background: none;
    border: none;
    color: var(--muted);
    font-size: 18px;
    cursor: pointer;
    line-height: 1;
  }
  .close:hover {
    color: var(--text);
  }
  .net {
    list-style: none;
    margin: 8px 0 0;
    padding: 0;
    display: grid;
    gap: 2px;
  }
  .net li {
    display: grid;
    grid-template-columns: 14px 1fr auto 36px;
    gap: 6px;
    align-items: baseline;
  }
  .name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .num {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }
  .loss {
    color: var(--muted);
  }
  .level-good .icon {
    color: var(--ok);
  }
  .level-warn .icon {
    color: var(--warn);
  }
  .level-bad .icon {
    color: var(--danger);
  }
  .level-unknown .icon {
    color: var(--muted);
  }
  .timer {
    margin-top: 8px;
    display: flex;
    flex-direction: column;
  }
  .timer strong {
    font-size: 16px;
    color: var(--accent);
  }
</style>
