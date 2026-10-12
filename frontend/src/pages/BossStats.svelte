<script lang="ts">
  import type {gamedata, main} from '../../wailsjs/go/models'
  import {lang, loc, t, type Key} from '../lib/i18n'
  import {damageKey, damageTypes, levelKey, signed, statusKey, statusLevel, statusTypes, type Version} from '../lib/boss'
  import BossTraits from './BossTraits.svelte'
  import {damageIcon, portrait, statusIcon} from '../lib/icons'

  export let data: main.BossData
  export let focus = ''

  let version: Version = 'normal'
  let party = 0 // index into health/poise: solo, duo, trio

  $: nightlords = data.bosses.filter((b) => b.kind === 'nightlord')
  $: if (!focus && nightlords.length) focus = nightlords[0].id
  $: nl = nightlords.find((b) => b.id === focus)
  $: hasEverdark = (nl?.stats?.everdark.length ?? 0) > 0
  $: if (!hasEverdark && version === 'everdark') version = 'normal'
  $: forms = (nl?.stats?.[version] ?? []) as gamedata.BossForm[]

  const parties: Key[] = ['boss.solo', 'boss.duo', 'boss.trio']
  // Largest |negation| in the data is 50; bars use a fixed ±50 scale so
  // bosses compare at a glance.
  const scale = 50
  const width = (v: number) => (Math.min(Math.abs(v), scale) / scale) * 50
  const fmt = (n: number) => n.toLocaleString($lang === 'en' ? 'en-US' : 'vi-VN', {maximumFractionDigits: 1})
</script>

<div class="picker">
  {#each nightlords as b (b.id)}
    <button class="chip" class:on={b.id === focus} on:click={() => (focus = b.id)}>
      {#if portrait(b.id)}<img class="portrait" src={portrait(b.id)} alt="" />{/if}
      {loc(b.name, $lang)}
      {#if b.dlc}<small class="tag">DLC</small>{/if}
    </button>
  {/each}
</div>

{#if nl}
  <div class="title">
    {#if portrait(nl.id)}<img class="portrait" src={portrait(nl.id)} alt="" />{/if}
    <div>
      <h2>{loc(nl.name, $lang)}</h2>
      {#if nl.expedition}<span class="muted">{$t('boss.expedition')}: {loc(nl.expedition, $lang)}</span>{/if}
    </div>
  </div>
  <div class="controls">
    <div class="seg">
      <button class:on={version === 'normal'} on:click={() => (version = 'normal')}>{$t('boss.normal')}</button>
      <button class:on={version === 'everdark'} disabled={!hasEverdark} on:click={() => (version = 'everdark')}
        title={hasEverdark ? '' : $t('boss.noEverdark')}>{$t('boss.everdark')}</button>
    </div>
    <div class="seg">
      {#each parties as p, i}
        <button class:on={party === i} on:click={() => (party = i)}>{$t(p)}</button>
      {/each}
    </div>
  </div>

  {#each forms as f}
    <section class="card form">
      <header>
        <h2>{f.name}</h2>
        <small class="muted">ID {f.gameIds.join(', ')}</small>
      </header>

      <div class="hero">
        <div>
          <span class="muted">{$t('boss.hp')}</span>
          <b>{f.health ? fmt(f.health[party]) : '?'}</b>
        </div>
        <div>
          <span class="muted">{$t('boss.poise')}</span>
          <b>{fmt(f.poise[party])}</b>
        </div>
      </div>

      <div class="cols">
        <div>
          <h3>{$t('boss.negation')}</h3>
          <div class="neg" role="table">
            {#each damageTypes as d}
              {@const v = f.negation[d] ?? 0}
              <div class="nrow" role="row" title={$t('boss.dmgTaken').replace('{n}', String(100 - v))}>
                <span role="cell"><img class="ico" src={damageIcon(d)} alt="" /> {$t(damageKey(d))}</span>
                <span class="track" role="cell">
                  <span class="axis"></span>
                  {#if v < 0}
                    <span class="fill weak" style="right: 50%; width: {width(v)}%"></span>
                  {:else if v > 0}
                    <span class="fill resist" style="left: 50%; width: {width(v)}%"></span>
                  {/if}
                </span>
                <span role="cell" class="num">{signed(v)}%</span>
              </div>
            {/each}
          </div>
          <p class="muted hint">{$t('boss.negationHint')}</p>
        </div>

        <div>
          <h3>{$t('boss.statusTitle')}</h3>
          <div class="status" role="table">
            {#each statusTypes as s}
              {@const lvl = statusLevel(f, s)}
              <div class="srow" role="row">
                <span role="cell"><img class="ico" src={statusIcon(s)} alt="" /> {$t(statusKey(s))}</span>
                <span role="cell" class="num">{lvl === 'immune' ? '—' : f.resist[s]}</span>
                <span role="cell" class="lvl {lvl}">{$t(levelKey(lvl))}</span>
              </div>
            {/each}
          </div>
          <p class="muted hint">{$t('boss.statusHint')}</p>
        </div>
      </div>

      <h3>{$t('boss.summary')}</h3>
      <BossTraits form={f} />
    </section>
  {/each}
{/if}

<style>
  .picker {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-bottom: 14px;
  }
  .picker .chip {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
  .picker .portrait {
    width: 20px;
    height: 20px;
    margin-left: -6px;
  }
  .title {
    display: flex;
    align-items: center;
    gap: 14px;
    margin-bottom: 14px;
  }
  .title h2 {
    margin: 0 0 2px;
    font-size: 18px;
  }
  .title .portrait {
    width: 72px;
    height: 72px;
  }
  .tag {
    margin-left: 4px;
    font-size: 10px;
    color: var(--muted);
  }
  .controls {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 12px;
    margin-bottom: 16px;
  }
  .form {
    margin-bottom: 16px;
  }
  header {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    gap: 8px;
  }
  .hero {
    display: flex;
    gap: 32px;
    margin: 4px 0 8px;
  }
  .hero div {
    display: flex;
    flex-direction: column;
  }
  .hero span {
    font-size: 12px;
  }
  .hero b {
    font-size: 26px;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
  }
  .cols {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 24px;
  }
  h3 {
    margin: 14px 0 8px;
    font-size: 13px;
    font-weight: 600;
  }
  .nrow,
  .srow {
    display: grid;
    gap: 10px;
    align-items: center;
    padding: 3px 0;
  }
  .nrow {
    grid-template-columns: 90px 1fr 48px;
  }
  .srow {
    grid-template-columns: 1fr 48px 90px;
    border-top: 1px solid var(--border);
  }
  .track {
    position: relative;
    height: 10px;
    background: var(--hover);
    border-radius: 4px;
  }
  .axis {
    position: absolute;
    left: 50%;
    top: -2px;
    bottom: -2px;
    width: 1px;
    background: var(--muted);
  }
  .fill {
    position: absolute;
    top: 0;
    bottom: 0;
  }
  .fill.weak {
    background: var(--ok);
    border-radius: 4px 0 0 4px;
  }
  .fill.resist {
    background: var(--danger);
    border-radius: 0 4px 4px 0;
  }
  .num {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }
  .lvl.easy {
    color: var(--ok);
  }
  .lvl.hard {
    color: var(--danger);
  }
  .lvl {
    font-size: 12px;
  }
  .lvl.mid {
    color: var(--warn);
  }
  .lvl.immune {
    color: var(--muted);
  }
  .hint {
    font-size: 12px;
    margin: 8px 0 0;
  }
</style>
