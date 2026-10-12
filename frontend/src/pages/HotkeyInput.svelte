<script lang="ts">
  import {onDestroy} from 'svelte'
  import {SuspendHotkeys} from '../../wailsjs/go/main/App'
  import {t} from '../lib/i18n'

  export let value: string
  let recording = false

  // KeyboardEvent.code → the key names internal/hotkey accepts.
  const named: Record<string, string> = {
    Space: 'Space',
    PageUp: 'PageUp',
    PageDown: 'PageDown',
    End: 'End',
    Home: 'Home',
    Insert: 'Insert',
    Delete: 'Delete',
    Pause: 'Pause',
    ScrollLock: 'ScrollLock',
    ArrowLeft: 'Left',
    ArrowUp: 'Up',
    ArrowRight: 'Right',
    ArrowDown: 'Down',
  }
  function keyName(code: string) {
    let m: RegExpExecArray | null
    if ((m = /^Key([A-Z])$/.exec(code))) return m[1]
    if ((m = /^Digit(\d)$/.exec(code))) return m[1]
    if ((m = /^Numpad(\d)$/.exec(code))) return `Num${m[1]}`
    if (/^F\d{1,2}$/.test(code)) return code
    return named[code] ?? ''
  }

  // The registered hotkeys would swallow the very keys being recorded, so
  // they are off while recording.
  function begin() {
    recording = true
    SuspendHotkeys(true).catch(() => {})
  }
  function end() {
    if (!recording) return
    recording = false
    SuspendHotkeys(false).catch(() => {})
  }
  onDestroy(end)

  function keydown(e: KeyboardEvent) {
    if (!recording) return
    e.preventDefault()
    if (e.key === 'Escape') return end()
    const k = keyName(e.code)
    if (!k) return // a modifier on its own, or an unsupported key
    const mods = [e.ctrlKey && 'Ctrl', e.altKey && 'Alt', e.shiftKey && 'Shift', e.metaKey && 'Win'].filter(Boolean)
    value = [...mods, k].join('+')
    end()
  }
</script>

<span class="hk">
  <button class="key" class:rec={recording} on:click={begin} on:keydown={keydown} on:blur={end}>
    {recording ? $t('settings.hotkeyPress') : value || $t('timer.keyOff')}
  </button>
  {#if value && !recording}
    <button class="link" title={$t('settings.hotkeyClear')} on:click={() => (value = '')}>×</button>
  {/if}
</span>

<style>
  .hk {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }
  .key {
    min-width: 150px;
    background: var(--hover);
    border: 1px solid var(--border);
    border-radius: 4px;
    color: var(--text);
    font: inherit;
    font-size: 12px;
    padding: 3px 8px;
    cursor: pointer;
    text-align: left;
  }
  .key.rec {
    border-color: var(--accent);
    color: var(--accent);
  }
  .link {
    font-size: 15px;
  }
</style>
