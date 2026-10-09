<script lang="ts">
  import {onDestroy, onMount} from 'svelte'
  import {GetSettings} from '../wailsjs/go/main/App'
  import {EventsOn, Quit} from '../wailsjs/runtime/runtime'
  import {lang, t} from './lib/i18n'

  type Status = {uptimeSec: number; dataVersion: string; phase: string}

  let status: Status | null = null
  let connected = false
  let opacity = 0.85

  function applySettings(s: {language: string; overlay: {opacity: number}}) {
    opacity = s.overlay.opacity
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
  .timer {
    margin-top: 10px;
    display: flex;
    flex-direction: column;
  }
  .timer strong {
    font-size: 20px;
    color: var(--accent);
  }
</style>
