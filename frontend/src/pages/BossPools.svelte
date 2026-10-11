<script lang="ts">
  import {createEventDispatcher} from 'svelte'
  import type {gamedata, main} from '../../wailsjs/go/models'
  import {lang, loc, t} from '../lib/i18n'

  export let data: main.BossData

  const dispatch = createEventDispatcher<{stats: string}>()

  $: byId = new Map<string, gamedata.Boss>(data.bosses.map((b) => [b.id, b]))
  $: byName = (a: gamedata.Boss, b: gamedata.Boss) => loc(a.name, $lang).localeCompare(loc(b.name, $lang))
  $: nights = [1, 2].map((n) => data.bosses.filter((b) => b.kind === 'night_boss' && b.night === n).sort(byName))
  $: sets = data.pools.map((p) => ({id: p.nightlord, n: [new Set(p.night1), new Set(p.night2)]}))
  $: unpooled = data.bosses.filter((b) => b.kind === 'nightlord' && !data.pools.some((p) => p.nightlord === b.id))
</script>

<section class="card">
  <p class="muted intro">{$t('boss.poolsIntro')}</p>
  {#if !data.poolsVerified}<p class="warn">{$t('boss.poolsUnverified')}</p>{/if}
  <div class="scroll">
    <table>
      <thead>
        <tr>
          <th></th>
          {#each sets as s}
            {@const nl = byId.get(s.id)}
            <th>
              <button class="link" title={loc(nl?.name, $lang)} on:click={() => dispatch('stats', s.id)}>
                {loc(nl?.expedition, $lang) || loc(nl?.name, $lang)}
              </button>
            </th>
          {/each}
          <th class="muted">{$t('boss.poolCount')}</th>
        </tr>
      </thead>
      {#each nights as list, i}
        <tbody>
          <tr class="section"><th colspan={sets.length + 2}>{$t(i === 0 ? 'boss.night1' : 'boss.night2')}</th></tr>
          {#each list as b (b.id)}
            {@const count = sets.filter((s) => s.n[i].has(b.id)).length}
            <tr>
              <th>{loc(b.name, $lang)}</th>
              {#each sets as s}
                <td class:yes={s.n[i].has(b.id)}>{s.n[i].has(b.id) ? '●' : ''}</td>
              {/each}
              <td class="muted">{count}</td>
            </tr>
          {/each}
        </tbody>
      {/each}
    </table>
  </div>
  {#if unpooled.length}
    <p class="muted foot">
      {$t('boss.noPool')}: {unpooled.map((b) => loc(b.name, $lang)).join(', ')}
    </p>
  {/if}
</section>

<style>
  .intro {
    margin-top: 0;
  }
  .warn {
    color: var(--warn);
    font-size: 12px;
  }
  .scroll {
    overflow-x: auto;
  }
  table {
    border-collapse: collapse;
    width: 100%;
    font-size: 13px;
  }
  th,
  td {
    padding: 5px 8px;
    border-top: 1px solid var(--border);
    text-align: center;
    white-space: nowrap;
  }
  tbody th {
    text-align: left;
    font-weight: normal;
  }
  thead th {
    border-top: none;
    vertical-align: bottom;
  }
  tr.section th {
    text-align: left;
    color: var(--accent);
    font-weight: 600;
    padding-top: 14px;
  }
  tbody tr:not(.section):hover {
    background: var(--hover);
  }
  td.yes {
    color: var(--accent);
  }
  .foot {
    font-size: 12px;
    margin: 12px 0 0;
  }
</style>
