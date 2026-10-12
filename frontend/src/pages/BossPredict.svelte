<script lang="ts">
  import {createEventDispatcher, onDestroy, onMount} from 'svelte'
  import {GetExpedition, OCRStatus, PredictNightlord} from '../../wailsjs/go/main/App'
  import {EventsOn} from '../../wailsjs/runtime/runtime'
  import type {boss, gamedata, main} from '../../wailsjs/go/models'
  import {lang, loc, t, type Key} from '../lib/i18n'
  import {pct} from '../lib/boss'
  import type {Prefill} from '../lib/runs'
  import BossTraits from './BossTraits.svelte'
  import {portrait} from '../lib/icons'

  export let data: main.BossData

  const dispatch = createEventDispatcher<{stats: string; record: Prefill}>()

  let night1 = ''
  let night2 = ''
  let exclude: string[] = []
  let result: boss.Result | null = null
  let error = ''
  // The expedition lives in the main process (boss OCR fills it in), so
  // predicting waits until it is loaded instead of overwriting it.
  let loaded = false
  let detected: string[] = []
  let ocr: main.OCRStatus | null = null

  function apply(e: main.Expedition) {
    night1 = e.night1
    night2 = e.night2
    exclude = e.exclude ?? []
    detected = e.detected ?? []
  }

  onMount(async () => {
    try {
      apply(await GetExpedition())
      ocr = await OCRStatus()
    } catch (e) {
      error = String(e)
    }
    loaded = true
  })
  const offs = [EventsOn('boss:expedition', apply), EventsOn('ocr:status', (s: main.OCRStatus) => (ocr = s))]
  onDestroy(() => offs.forEach((off) => off()))

  $: byId = new Map<string, gamedata.Boss>(data.bosses.map((b) => [b.id, b]))
  $: byName = (a: gamedata.Boss, b: gamedata.Boss) => loc(a.name, $lang).localeCompare(loc(b.name, $lang))
  $: nights = [1, 2].map((n) => data.bosses.filter((b) => b.kind === 'night_boss' && b.night === n).sort(byName))

  // Requests can resolve out of order; only the latest one may land.
  let seq = 0
  async function predict(q: boss.Query) {
    const mine = ++seq
    try {
      const r = await PredictNightlord(q)
      if (mine === seq) {
        result = r
        error = ''
      }
    } catch (e) {
      if (mine === seq) error = String(e)
    }
  }
  $: if (loaded) predict({night1, night2, exclude})

  // chance of a boss on a night still open; undefined once that night is set.
  function chance(r: boss.Result | null, night: number, id: string): number | undefined {
    const list = night === 1 ? r?.night1 : r?.night2
    if (!list) return undefined
    return list.find((c) => c.boss === id)?.probability ?? 0
  }

  function pick(night: number, id: string) {
    if (night === 1) night1 = night1 === id ? '' : id
    else night2 = night2 === id ? '' : id
  }

  function toggleExclude(id: string) {
    exclude = exclude.includes(id) ? exclude.filter((x) => x !== id) : [...exclude, id]
  }

  function reset() {
    night1 = night2 = ''
    exclude = []
  }

  const ocrKey = (s: main.OCRStatus) => `ocr.state.${s.state}` as Key

  $: top = result && result.signal && !result.impossible ? result.ranking.filter((c) => c.probability > 0).slice(0, 2) : []
  $: selected = [night1, night2]
  // Only a certain Nightlord is worth filling into the run form.
  $: sure = result?.signal && !result.impossible && result.ranking[0]?.probability >= 0.995 ? result.ranking[0].nightlord : ''

  function record() {
    dispatch('record', {night1, night2, nightlord: sure})
  }
</script>

<section class="card intro">
  <div>
    <p class="muted">{$t('boss.predictIntro')}</p>
    {#if ocr}
      <p class="ocr {ocr.state}">
        {$t(ocrKey(ocr))}{#if ocr.state === 'scanning'} — {$t(ocr.slot === 3 ? 'ocr.nightlord' : ocr.slot === 2 ? 'boss.night2' : 'boss.night1')}{/if}
        {#if ocr.error}<span class="muted">({ocr.error})</span>{/if}
      </p>
    {/if}
  </div>
  <span class="buttons">
    <button class="secondary" on:click={record} disabled={!night1 && !night2}>{$t('boss.recordRun')}</button>
    <button class="secondary" on:click={reset} disabled={!night1 && !night2 && exclude.length === 0}>{$t('boss.reset')}</button>
  </span>
</section>

<div class="pickers">
  {#each nights as list, i}
    <section class="card">
      <h2>
        {$t(i === 0 ? 'boss.night1' : 'boss.night2')}
        {#if detected.includes(i === 0 ? 'night1' : 'night2')}<span class="auto" title={$t('ocr.detectedHint')}>{$t('ocr.detected')}</span>{/if}
      </h2>
      <div class="bosses">
        {#each list as b (b.id)}
          {@const c = chance(result, i + 1, b.id)}
          <button
            class="chip"
            class:on={selected[i] === b.id}
            disabled={c === 0 && selected[i] !== b.id}
            title={c === 0 ? $t('boss.notInPool') : ''}
            on:click={() => pick(i + 1, b.id)}
          >
            {loc(b.name, $lang)}
            {#if c !== undefined && c > 0}<small>{pct(c)}</small>{/if}
          </button>
        {/each}
      </div>
    </section>
  {/each}
</div>

{#if error}<p class="err">{error}</p>{/if}

{#if result}
  <section class="card">
    <h2>{$t('boss.ranking')}</h2>
    {#if result.impossible}
      <p class="err">{$t('boss.impossible')}</p>
    {:else if !result.signal}
      <p class="muted">{$t('boss.noSignal')}</p>
    {/if}
    <div class="ranking">
      {#each result.ranking as c (c.nightlord)}
        {@const nl = byId.get(c.nightlord)}
        <div class="row" class:out={c.excluded || c.probability === 0}>
          <span class="who">
            {#if portrait(c.nightlord)}<img class="portrait" src={portrait(c.nightlord)} alt="" />{:else}<span class="portrait"></span>{/if}
            <span>
              <b>{loc(nl?.name, $lang)}</b>
              <small class="muted">{loc(nl?.expedition, $lang)}</small>
            </span>
          </span>
          <span class="bar" title={pct(c.probability)}>
            <span style="width: {c.probability * 100}%"></span>
          </span>
          <span class="num">{c.excluded ? '—' : pct(c.probability)}</span>
          <span class="actions">
            <button class="link" on:click={() => toggleExclude(c.nightlord)}>
              {$t(c.excluded ? 'boss.include' : 'boss.exclude')}
            </button>
            <button class="link" on:click={() => dispatch('stats', c.nightlord)}>{$t('boss.viewStats')}</button>
          </span>
        </div>
      {/each}
    </div>
    <p class="muted foot">
      {$t('boss.assumption')}
      {result.runs ? $t('boss.learned').replace('{n}', String(result.runs)) : $t('boss.learnHint')}
    </p>
  </section>

  {#if top.length}
    <div class="prep">
      {#each top as c (c.nightlord)}
        {@const nl = byId.get(c.nightlord)}
        <section class="card">
          <h2 class="prepTitle">
            {#if portrait(c.nightlord)}<img class="portrait" src={portrait(c.nightlord)} alt="" />{/if}
            <span>{$t('boss.prepare')} {loc(nl?.name, $lang)} <span class="muted">· {pct(c.probability)}</span></span>
          </h2>
          {#each nl?.stats?.normal ?? [] as f}
            {#if (nl?.stats?.normal.length ?? 0) > 1}<h3>{f.name}</h3>{/if}
            <BossTraits form={f} />
          {/each}
        </section>
      {/each}
    </div>
  {/if}
{/if}

<style>
  .intro {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 16px;
    margin-bottom: 16px;
  }
  .intro p {
    margin: 0;
  }
  .ocr {
    margin-top: 6px !important;
    font-size: 12px;
    color: var(--muted);
  }
  .ocr.scanning {
    color: var(--accent);
  }
  .ocr.error,
  .ocr.noGame {
    color: var(--warn);
  }
  .auto {
    margin-left: 8px;
    font-size: 11px;
    font-weight: 500;
    color: var(--accent);
    border: 1px solid var(--accent);
    border-radius: 4px;
    padding: 0 5px;
  }
  .buttons {
    display: flex;
    gap: 8px;
  }
  .pickers,
  .prep {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
    gap: 16px;
    margin-bottom: 16px;
  }
  .prep {
    margin: 16px 0 0;
  }
  .bosses {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
  .chip small {
    margin-left: 4px;
    color: var(--muted);
    font-variant-numeric: tabular-nums;
  }
  .ranking {
    display: grid;
    gap: 2px;
  }
  .row {
    display: grid;
    grid-template-columns: minmax(200px, 1.4fr) 2fr 52px max-content;
    gap: 12px;
    align-items: center;
    padding: 6px 0;
    border-top: 1px solid var(--border);
  }
  .row.out {
    opacity: 0.45;
  }
  .who {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  .who > span:not(.portrait) {
    display: flex;
    flex-direction: column;
  }
  .who .portrait {
    width: 36px;
    height: 36px;
  }
  .prepTitle {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  .prepTitle .portrait {
    width: 44px;
    height: 44px;
  }
  .bar {
    height: 8px;
    background: var(--hover);
    border-radius: 4px;
    overflow: hidden;
  }
  .bar span {
    display: block;
    height: 100%;
    background: var(--accent);
    border-radius: 0 4px 4px 0;
    transition: width 0.2s;
  }
  .num {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }
  .actions {
    display: flex;
    gap: 12px;
  }
  h3 {
    margin: 12px 0 6px;
    font-size: 13px;
    font-weight: 600;
  }
  .err {
    color: var(--danger);
  }
  .foot {
    font-size: 12px;
    margin: 12px 0 0;
  }
  button.secondary {
    background: none;
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 6px 14px;
    font: inherit;
    cursor: pointer;
    white-space: nowrap;
  }
  button.secondary:disabled {
    opacity: 0.4;
    cursor: default;
  }
</style>
