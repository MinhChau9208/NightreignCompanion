<script lang="ts">
  import {onMount} from 'svelte'
  import {GetSettings, SaveSettings} from '../../wailsjs/go/main/App'
  import {config} from '../../wailsjs/go/models'
  import {lang, t} from '../lib/i18n'

  let s: config.Settings | null = null
  let status = ''
  let targetsText = ''

  onMount(async () => {
    s = await GetSettings()
    targetsText = s.network.pingTargets.join('\n')
  })

  async function save() {
    if (!s) return
    status = ''
    try {
      s.network.pingTargets = targetsText
        .split('\n')
        .map((l) => l.trim())
        .filter((l) => l !== '')
      // Opening the settings page counts as having seen the consent text.
      s.sharing.asked = true
      await SaveSettings(config.Settings.createFrom(s))
      lang.set(s.language === 'en' ? 'en' : 'vi')
      status = $t('settings.saved')
    } catch (e) {
      status = String(e)
    }
  }
</script>

<h1>{$t('settings.title')}</h1>

{#if s}
  <div class="stack">
    <section class="card">
      <h2>{$t('settings.language')}</h2>
      <select bind:value={s.language}>
        <option value="vi">Tiếng Việt</option>
        <option value="en">English</option>
      </select>
    </section>

    <section class="card">
      <h2>{$t('settings.overlay')}</h2>
      <label class="row">
        {$t('settings.opacity')}
        <input type="range" min="0.2" max="1" step="0.05" bind:value={s.overlay.opacity} />
        <span>{Math.round(s.overlay.opacity * 100)}%</span>
      </label>
    </section>

    <section class="card">
      <h2>{$t('settings.network')}</h2>
      <label class="field">
        {$t('settings.targets')}
        <textarea rows="4" spellcheck="false" bind:value={targetsText} />
      </label>
      <p class="muted hint">{$t('settings.targetsHint')}</p>
      <h3>{$t('settings.thresholds')}</h3>
      <div class="thresholds">
        <label>Ping (ms)<input type="number" min="1" bind:value={s.network.thresholds.pingMs} /></label>
        <label>Jitter (ms)<input type="number" min="1" bind:value={s.network.thresholds.jitterMs} /></label>
        <label>{$t('net.lost')} (%)<input type="number" min="0" step="0.5" bind:value={s.network.thresholds.lossPct} /></label>
        <label>{$t('settings.minFps')}<input type="number" min="1" bind:value={s.network.thresholds.minFps} /></label>
      </div>
      <label class="field process">
        {$t('settings.fpsProcess')}
        <input type="text" spellcheck="false" bind:value={s.fps.process} />
      </label>
    </section>

    <section class="card">
      <h2>{$t('settings.hotkeys')}</h2>
      <ul class="keys">
        <li><span>Timer start</span><kbd>{s.hotkeys.timerStart}</kbd></li>
        <li><span>Timer sync</span><kbd>{s.hotkeys.timerSync}</kbd></li>
        <li><span>Timer reset</span><kbd>{s.hotkeys.timerReset}</kbd></li>
        <li><span>Overlay</span><kbd>{s.hotkeys.overlay}</kbd></li>
      </ul>
      <p class="muted">{$t('settings.hotkeysNote')}</p>
    </section>

    <section class="card">
      <h2>{$t('settings.sharing')}</h2>
      <p>{$t('settings.sharingBody')}</p>
      <label class="check"><input type="checkbox" bind:checked={s.sharing.bosses} /> {$t('settings.shareBosses')}</label>
      <label class="check"><input type="checkbox" bind:checked={s.sharing.builds} /> {$t('settings.shareBuilds')}</label>
      <label class="check"><input type="checkbox" bind:checked={s.sharing.clearTimes} /> {$t('settings.shareClear')}</label>
      <p class="muted">{$t('settings.sharingPending')}</p>
    </section>

    <div class="row">
      <button class="primary" on:click={save}>{$t('settings.save')}</button>
      <span class="muted">{status}</span>
    </div>
  </div>
{/if}

<style>
  .stack {
    display: grid;
    gap: 16px;
    max-width: 640px;
  }
  .row {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .field {
    display: grid;
    gap: 6px;
  }
  textarea,
  input[type='text'],
  input[type='number'] {
    background: var(--hover);
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 6px 10px;
    font: inherit;
  }
  textarea {
    font-family: Consolas, monospace;
    resize: vertical;
  }
  .process {
    margin-top: 14px;
  }
  .hint {
    font-size: 12px;
  }
  h3 {
    font-size: 13px;
    font-weight: 600;
    margin: 14px 0 8px;
  }
  .thresholds {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 12px;
  }
  .thresholds label {
    display: grid;
    gap: 4px;
    color: var(--muted);
    font-size: 12px;
  }
  .check {
    display: flex;
    gap: 8px;
    align-items: center;
    padding: 4px 0;
  }
  .keys {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: 6px;
  }
  .keys span {
    display: inline-block;
    min-width: 110px;
    color: var(--muted);
  }
  kbd {
    background: var(--hover);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 1px 6px;
    font-size: 12px;
  }
</style>
