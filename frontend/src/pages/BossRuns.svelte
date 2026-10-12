<script lang="ts">
  import {onMount} from 'svelte'
  import {DeleteRun, GetExpedition, ListRuns, SaveRun, TimerState} from '../../wailsjs/go/main/App'
  import {store, type gamedata, type main} from '../../wailsjs/go/models'
  import {lang, loc, t} from '../lib/i18n'
  import {clock} from '../lib/timer'
  import {portrait} from '../lib/icons'
  import {groupBy, maxDepth, parseClock, resultKey, runResults, summary, type Prefill, type Run} from '../lib/runs'

  export let data: main.BossData
  // Bosses picked on the predictor; consumed once.
  export let prefill: Prefill | null = null

  let runs: Run[] = []
  let error = ''
  let status = ''

  // The form.
  let mode = 'deep'
  let depth = 3
  let character = ''
  let night1 = ''
  let night2 = ''
  let nightlord = ''
  let result = 'win'
  let clearText = ''
  let confirmDelete = ''

  $: names = new Map<string, string>([
    ...data.bosses.map((b): [string, string] => [b.id, loc(b.name, $lang)]),
    ...data.characters.map((c): [string, string] => [c.id, loc(c.name, $lang)]),
  ])
  $: name = (id: string) => names.get(id) ?? id
  $: byName = (a: gamedata.Boss, b: gamedata.Boss) => loc(a.name, $lang).localeCompare(loc(b.name, $lang))
  $: lords = data.bosses.filter((b) => b.kind === 'nightlord').sort(byName)
  $: nights = [1, 2].map((n) => data.bosses.filter((b) => b.kind === 'night_boss' && b.night === n).sort(byName))

  $: clearSeconds = parseClock(clearText)
  $: total = summary(runs)
  $: perLord = groupBy(runs, (r) => r.nightlord)
  $: perChar = groupBy(runs, (r) => r.character)
  $: perNight = [groupBy(runs, (r) => r.night1Boss), groupBy(runs, (r) => r.night2Boss)]

  $: if (prefill) {
    ;({night1, night2, nightlord} = prefill)
    mode = 'deep'
    prefill = null
  }

  onMount(async () => {
    await load()
    // Bosses picked on the predictor or read by boss OCR.
    if (!night1 && !night2 && !nightlord) {
      try {
        const e = await GetExpedition()
        ;({night1, night2, nightlord} = e)
        if (e.night1 || e.night2) mode = 'deep'
      } catch {}
    }
    // A finished timer run is most likely the one being recorded.
    try {
      const s = await TimerState()
      if (s.state === 'done' && !clearText) clearText = clock(s.elapsed)
    } catch {}
  })

  async function load() {
    try {
      runs = await ListRuns()
    } catch (e) {
      error = String(e)
    }
  }

  async function fromTimer() {
    try {
      const s = await TimerState()
      if (s.state !== 'idle') clearText = clock(s.elapsed)
    } catch (e) {
      status = String(e)
    }
  }

  async function save() {
    if (clearSeconds === null) return
    try {
      await SaveRun(
        store.Run.createFrom({
          mode,
          depth: mode === 'deep' ? depth : 0,
          character,
          night1Boss: night1,
          night2Boss: night2,
          shiftingEarth: '',
          nightlord,
          result,
          clearSeconds,
          buildJson: '',
        }),
      )
      night1 = night2 = nightlord = clearText = ''
      status = $t('runs.saved')
      await load()
    } catch (e) {
      status = String(e)
    }
  }

  async function remove(id: string) {
    if (confirmDelete !== id) {
      confirmDelete = id
      return
    }
    confirmDelete = ''
    try {
      await DeleteRun(id)
      await load()
    } catch (e) {
      error = String(e)
    }
  }

  const rate = (wins: number, n: number) => (n ? `${Math.round((wins / n) * 100)}%` : '—')
  const when = (r: Run) =>
    new Date(r.createdAt).toLocaleString($lang === 'vi' ? 'vi-VN' : 'en-GB', {dateStyle: 'short', timeStyle: 'short'})
</script>

<section class="card form">
  <h2>{$t('runs.record')}</h2>
  <p class="muted">{$t('runs.intro')}</p>

  <div class="grid">
    <span>{$t('runs.mode')}</span>
    <div class="line">
      <div class="seg">
        <button class:on={mode === 'normal'} on:click={() => (mode = 'normal')}>{$t('runs.normal')}</button>
        <button class:on={mode === 'deep'} on:click={() => (mode = 'deep')}>Deep of Night</button>
      </div>
      {#if mode === 'deep'}
        <label class="line">
          {$t('runs.depth')}
          <select bind:value={depth}>
            {#each Array.from({length: maxDepth}, (_, i) => i + 1) as d}<option value={d}>{d}</option>{/each}
          </select>
        </label>
      {/if}
    </div>

    <span>{$t('runs.character')}</span>
    <select bind:value={character}>
      <option value="">—</option>
      {#each data.characters as c (c.id)}<option value={c.id}>{loc(c.name, $lang)}</option>{/each}
    </select>

    {#each nights as list, i}
      <span>{$t(i === 0 ? 'boss.night1' : 'boss.night2')}</span>
      {#if i === 0}
        <select bind:value={night1}>
          <option value="">—</option>
          {#each list as b (b.id)}<option value={b.id}>{loc(b.name, $lang)}</option>{/each}
        </select>
      {:else}
        <select bind:value={night2}>
          <option value="">—</option>
          {#each list as b (b.id)}<option value={b.id}>{loc(b.name, $lang)}</option>{/each}
        </select>
      {/if}
    {/each}

    <span>Nightlord</span>
    <select bind:value={nightlord}>
      <option value="">—</option>
      {#each lords as b (b.id)}<option value={b.id}>{loc(b.name, $lang)}</option>{/each}
    </select>

    <span>{$t('runs.result')}</span>
    <div class="seg">
      {#each runResults as r}
        <button class:on={result === r} on:click={() => (result = r)}>{$t(resultKey(r))}</button>
      {/each}
    </div>

    <span>{$t('runs.clear')}</span>
    <div class="line">
      <input type="text" placeholder="mm:ss" bind:value={clearText} class:bad={clearSeconds === null} />
      <button class="link" on:click={fromTimer}>{$t('runs.fromTimer')}</button>
    </div>
  </div>

  <div class="line actions">
    <button class="primary" on:click={save} disabled={clearSeconds === null}>{$t('runs.save')}</button>
    <span class="muted">{clearSeconds === null ? $t('runs.badClear') : status}</span>
  </div>
</section>

{#if error}<p class="err">{error}</p>{/if}

{#if runs.length}
  <section class="card">
    <h2>{$t('runs.stats')}</h2>
    <p class="totals">
      <b>{total.runs}</b> {$t('runs.runs')} · <b>{total.wins}</b> {$t('runs.result.win').toLowerCase()} ·
      <b>{total.losses}</b> {$t('runs.result.loss').toLowerCase()} · {$t('runs.winRate')} <b>{rate(total.wins, total.runs)}</b>
      {#if total.avgClear}· {$t('runs.avgClear')} <b>{clock(total.avgClear)}</b>{/if}
    </p>

    <div class="tables">
      {#each [{title: 'Nightlord', rows: perLord}, {title: $t('runs.character'), rows: perChar}] as tbl}
        {#if tbl.rows.length}
          <table>
            <thead>
              <tr><th>{tbl.title}</th><th>{$t('runs.runs')}</th><th>{$t('runs.winRate')}</th><th>{$t('runs.avgClear')}</th></tr>
            </thead>
            <tbody>
              {#each tbl.rows as g (g.id)}
                <tr>
                  <td>
                    {#if portrait(g.id)}<img class="portrait" src={portrait(g.id)} alt="" />{/if}
                    {name(g.id)}
                  </td>
                  <td class="num">{g.runs}</td>
                  <td class="num">{rate(g.wins, g.runs)}</td>
                  <td class="num">{g.avgClear ? clock(g.avgClear) : '—'}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}
      {/each}
      {#each perNight as rows, i}
        {#if rows.length}
          <table>
            <thead>
              <tr><th>{$t(i === 0 ? 'boss.night1' : 'boss.night2')}</th><th>{$t('runs.seen')}</th><th>{$t('runs.winRate')}</th></tr>
            </thead>
            <tbody>
              {#each rows as g (g.id)}
                <tr><td>{name(g.id)}</td><td class="num">{g.runs}</td><td class="num">{rate(g.wins, g.runs)}</td></tr>
              {/each}
            </tbody>
          </table>
        {/if}
      {/each}
    </div>
  </section>

  <section class="card">
    <h2>{$t('runs.history')}</h2>
    <table class="history">
      <tbody>
        {#each runs as r (r.id)}
          <tr>
            <td class="muted">{when(r)}</td>
            <td>{r.mode === 'deep' ? `Deep ${r.depth}` : $t('runs.normal')}</td>
            <td>{r.character ? name(r.character) : ''}</td>
            <td>
              {[r.night1Boss, r.night2Boss, r.nightlord]
                .filter(Boolean)
                .map(name)
                .join(' → ')}
            </td>
            <td class={r.result}>{$t(resultKey(r.result))}</td>
            <td class="num">{r.clearSeconds ? clock(r.clearSeconds) : ''}</td>
            <td>
              <button class="link" on:click={() => remove(r.id)}>
                {$t(confirmDelete === r.id ? 'runs.confirmDelete' : 'runs.delete')}
              </button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </section>
{:else if !error}
  <p class="muted">{$t('runs.empty')}</p>
{/if}

<style>
  section {
    margin-bottom: 16px;
  }
  .form p {
    margin: 0 0 14px;
  }
  .grid {
    display: grid;
    grid-template-columns: max-content minmax(0, 360px);
    gap: 10px 16px;
    align-items: center;
  }
  .line {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .actions {
    margin-top: 16px;
  }
  input[type='text'] {
    width: 90px;
    background: var(--hover);
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 6px 10px;
    font: inherit;
  }
  input.bad {
    border-color: var(--danger);
  }
  .totals {
    margin: 0 0 16px;
  }
  .tables {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 16px 24px;
    align-items: start;
  }
  table {
    width: 100%;
    border-collapse: collapse;
  }
  th {
    text-align: left;
    font-weight: 600;
    color: var(--muted);
    font-size: 12px;
  }
  th,
  td {
    padding: 5px 8px 5px 0;
    border-bottom: 1px solid var(--border);
  }
  td .portrait {
    width: 22px;
    height: 22px;
    vertical-align: middle;
    margin-right: 6px;
  }
  .num {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }
  .history td {
    white-space: nowrap;
  }
  .history td:nth-child(4) {
    white-space: normal;
    width: 100%;
  }
  .win {
    color: var(--ok);
  }
  .loss {
    color: var(--danger);
  }
  .abandoned {
    color: var(--muted);
  }
  .err {
    color: var(--danger);
  }
</style>
