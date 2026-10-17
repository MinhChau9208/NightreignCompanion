<script lang="ts">
  import {onDestroy, onMount} from 'svelte'
  import {GetSettings, GetTimerProfiles, HotkeyErrors, SaveSettings, TimerCommand, TimerSyncStatus} from '../../wailsjs/go/main/App'
  import {config, type gamedata, type main} from '../../wailsjs/go/models'
  import {EventsOn} from '../../wailsjs/runtime/runtime'
  import {lang, loc, t, type Key} from '../lib/i18n'
  import {clock, kindKey, mainClock, stateKey, timerState, urgency} from '../lib/timer'

  const state = timerState()
  let profiles: gamedata.TimerProfile[] = []
  let settings: config.Settings | null = null
  let keyErrs: string[] = []
  let sync: main.SyncStatus | null = null
  let error = ''

  const offs = [
    EventsOn('hotkeys', (errs: string[]) => (keyErrs = errs ?? [])),
    EventsOn('timer:sync', (st: main.SyncStatus) => (sync = st)),
  ]
  onDestroy(() => offs.forEach((off) => off()))

  onMount(async () => {
    try {
      ;[profiles, settings, keyErrs, sync] = await Promise.all([GetTimerProfiles(), GetSettings(), HotkeyErrors(), TimerSyncStatus()])
    } catch (e) {
      error = String(e)
    }
  })

  async function run(cmd: string) {
    error = ''
    try {
      await TimerCommand(cmd)
    } catch (e) {
      error = String(e)
    }
  }

  async function saveTimer() {
    if (!settings) return
    error = ''
    try {
      await SaveSettings(config.Settings.createFrom(settings))
    } catch (e) {
      error = String(e)
    }
  }

  $: s = $state
  // The profile the timer runs on; a newly picked one applies after reset.
  $: prof = profiles.find((p) => p.id === s?.profile)
  $: active = s && (s.state === 'running' || s.state === 'paused')
  $: boss = s?.kind === 'boss'
  $: keys = settings
    ? ([
        ['timer.key.start', settings.hotkeys.timerStart],
        ['timer.key.sync', settings.hotkeys.timerSync],
        ['timer.key.reset', settings.hotkeys.timerReset],
        ['timer.key.overlay', settings.hotkeys.overlay],
      ] as [Key, string][])
    : []

  const syncKey = (st: main.SyncStatus) => `sync.state.${st.state}` as Key
  // The label of the phase the last banner moved the timer to.
  $: lastPhase = sync?.last.at ? prof?.phases.find((p) => p.id === sync?.last.phase) : undefined
  const time = (ms: number) => new Date(ms).toLocaleTimeString()
</script>

<h1>{$t('nav.timer')}</h1>

<p class="muted intro">{$t('timer.intro')}</p>

{#if prof && !prof.verified}
  <p class="warn">⚠ {$t('timer.unverified')}</p>
{/if}

{#if s}
  <div class="grid">
    <section class="card now">
      <div class="state">
        <span class="badge {s.state}">{$t(stateKey(s.state))}</span>
        {#if s.kind}<span class="kind {s.kind}">{$t(kindKey(s.kind))}</span>{/if}
      </div>

      <div class="big {urgency(s)}">{mainClock(s)}</div>
      <div class="label">
        {#if s.state === 'done'}
          {$t('timer.finished')}
        {:else if s.state === 'idle'}
          {$t('timer.idleHint')}
        {:else}
          {loc(s.label, $lang)}{#if boss}<span class="muted"> · {$t('timer.fight')}</span>{/if}
        {/if}
      </div>
      {#if s.nextKind && s.state !== 'done'}
        <div class="muted next">{$t('timer.then')}: {loc(s.nextLabel, $lang)}</div>
      {/if}
      {#if s.state !== 'idle'}
        <div class="muted next">{$t('timer.total')}: {clock(s.elapsed)}</div>
      {/if}

      <div class="actions">
        {#if !active}
          <button class="primary" on:click={() => run('start')}>{$t(s.state === 'done' ? 'timer.restart' : 'timer.start')}</button>
        {:else}
          <button class="primary" on:click={() => run('next')} title={$t(boss ? 'timer.bossDoneHint' : 'timer.nextHint')}>
            {$t(boss ? 'timer.bossDone' : 'timer.next')}
          </button>
          <button on:click={() => run(s?.state === 'paused' ? 'resume' : 'pause')}>
            {$t(s.state === 'paused' ? 'timer.resume' : 'timer.pause')}
          </button>
          <button on:click={() => run('prev')} disabled={s.phase <= 0}>{$t('timer.prev')}</button>
        {/if}
        {#if s.state !== 'idle'}
          <button on:click={() => run('reset')}>{$t('timer.reset')}</button>
        {/if}
      </div>
      {#if error}<p class="err">{error}</p>{/if}
      {#if sync}
        <p class="sync small {sync.state}">
          {$t(syncKey(sync))}{#if sync.error} <span class="muted">({sync.error})</span>{/if}
          {#if sync.last.at}
            <br /><span class="muted"
              >{$t('sync.last')}: “{sync.last.text}” → {lastPhase ? loc(lastPhase.label, $lang) : sync.last.phase} · {time(sync.last.at)}</span
            >
          {/if}
        </p>
      {/if}
    </section>

    <section class="card">
      <h2>{$t('timer.hotkeys')}</h2>
      <ul class="keys">
        {#each keys as [label, key]}
          <li><span class="muted">{$t(label)}</span><kbd class:off={!key}>{key || $t('timer.keyOff')}</kbd></li>
        {/each}
      </ul>
      {#if keyErrs.length}
        <p class="err small">{$t('timer.hotkeyErr')}</p>
        <ul class="err small errs">
          {#each keyErrs as e}<li>{e}</li>{/each}
        </ul>
      {/if}
      <p class="muted small">{$t('timer.syncHint')}</p>

      {#if settings}
        <h2 class="opts">{$t('timer.options')}</h2>
        <label class="row">
          {$t('timer.profile')}
          <select bind:value={settings.timer.profile} on:change={saveTimer} disabled={profiles.length < 2}>
            {#each profiles as p}<option value={p.id}>{loc(p.name, $lang)}</option>{/each}
          </select>
        </label>
        {#if active && settings.timer.profile !== s.profile}
          <p class="muted small">{$t('timer.profileAfterReset')}</p>
        {/if}
        <label class="check">
          <input type="checkbox" bind:checked={settings.timer.sound} on:change={saveTimer} />
          {$t('timer.sound')}
        </label>
      {/if}
    </section>
  </div>

  {#if prof}
    <section class="card phases">
      <h2>{$t('timer.phases')} — {loc(prof.name, $lang)}</h2>
      <ol>
        {#each prof.phases as p, i}
          <li class:past={s.phase > i} class:current={s.phase === i}>
            <span class="kind {p.kind}">{$t(kindKey(p.kind))}</span>
            <span>{loc(p.label, $lang)}</span>
            <span class="num muted">{p.seconds ? clock(p.seconds) : $t('timer.untimed')}</span>
          </li>
        {/each}
      </ol>
      {#if prof.note}<p class="muted small">{prof.note}</p>{/if}
    </section>
  {/if}
{:else if error}
  <p class="err">{error}</p>
{/if}

<style>
  .intro {
    max-width: 760px;
    margin-top: -8px;
  }
  .warn {
    color: var(--warn);
    font-size: 13px;
  }
  .grid {
    display: grid;
    grid-template-columns: minmax(320px, 1.3fr) 1fr;
    gap: 16px;
    margin-bottom: 16px;
  }
  .now {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .state {
    display: flex;
    gap: 8px;
    align-items: center;
  }
  .badge,
  .kind {
    font-size: 11px;
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 1px 6px;
    color: var(--muted);
  }
  .badge.running {
    color: var(--ok);
    border-color: var(--ok);
  }
  .badge.paused {
    color: var(--warn);
    border-color: var(--warn);
  }
  .kind.shrink {
    color: var(--warn);
  }
  .kind.boss {
    color: var(--danger);
  }
  .big {
    font-size: 64px;
    font-weight: 600;
    line-height: 1.1;
    color: var(--accent);
    font-variant-numeric: tabular-nums;
  }
  .big.warn {
    color: var(--warn);
  }
  .big.bad {
    color: var(--danger);
  }
  .label {
    font-size: 16px;
  }
  .next {
    font-size: 13px;
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-top: 12px;
  }
  .actions button:not(.primary) {
    background: var(--hover);
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 8px 14px;
    font: inherit;
    cursor: pointer;
  }
  .actions button:disabled {
    opacity: 0.4;
    cursor: default;
  }
  .keys {
    list-style: none;
    padding: 0;
    margin: 0 0 10px;
    display: grid;
    gap: 6px;
  }
  .keys li {
    display: flex;
    justify-content: space-between;
    gap: 12px;
  }
  kbd {
    background: var(--hover);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 1px 6px;
    font-size: 12px;
  }
  kbd.off {
    color: var(--muted);
  }
  .opts {
    margin-top: 18px;
  }
  .row {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .check {
    display: flex;
    gap: 8px;
    align-items: center;
    padding: 8px 0 0;
  }
  .small {
    font-size: 12px;
  }
  .sync {
    margin: 10px 0 0;
    color: var(--muted);
  }
  .sync.watching {
    color: var(--ok);
  }
  .sync.error {
    color: var(--danger);
  }
  .errs {
    padding-left: 18px;
    margin: 4px 0 10px;
  }
  .err {
    color: var(--danger);
  }
  .phases ol {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: 2px;
  }
  .phases li {
    display: grid;
    grid-template-columns: 90px 1fr auto;
    gap: 12px;
    align-items: baseline;
    padding: 5px 8px;
    border-radius: 6px;
  }
  .phases li.past {
    opacity: 0.45;
  }
  .phases li.current {
    background: var(--hover);
    color: var(--accent);
  }
  .phases .kind {
    justify-self: start;
  }
  .num {
    font-variant-numeric: tabular-nums;
  }
</style>
