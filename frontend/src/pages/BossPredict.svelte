<script lang="ts">
  import {createEventDispatcher} from 'svelte'
  import {PredictNightlord} from '../../wailsjs/go/main/App'
  import type {boss, gamedata, main} from '../../wailsjs/go/models'
  import {lang, t} from '../lib/i18n'
  import {loc, pct} from '../lib/boss'
  import BossTraits from './BossTraits.svelte'

  export let data: main.BossData

  const dispatch = createEventDispatcher<{stats: string}>()

  let night1 = ''
  let night2 = ''
  let exclude: string[] = []
  let result: boss.Result | null = null
  let error = ''

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
  $: predict({night1, night2, exclude})

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

  $: top = result && result.signal && !result.impossible ? result.ranking.filter((c) => c.probability > 0).slice(0, 2) : []
  $: selected = [night1, night2]
</script>

<section class="card intro">
  <p class="muted">{$t('boss.predictIntro')}</p>
  <button class="secondary" on:click={reset} disabled={!night1 && !night2 && exclude.length === 0}>{$t('boss.reset')}</button>
</section>

<div class="pickers">
  {#each nights as list, i}
    <section class="card">
      <h2>{$t(i === 0 ? 'boss.night1' : 'boss.night2')}</h2>
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
            <b>{loc(nl?.name, $lang)}</b>
            <small class="muted">{loc(nl?.expedition, $lang)}</small>
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
    <p class="muted foot">{$t('boss.assumption')}</p>
  </section>

  {#if top.length}
    <div class="prep">
      {#each top as c (c.nightlord)}
        {@const nl = byId.get(c.nightlord)}
        <section class="card">
          <h2>{$t('boss.prepare')} {loc(nl?.name, $lang)} <span class="muted">· {pct(c.probability)}</span></h2>
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
    flex-direction: column;
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
