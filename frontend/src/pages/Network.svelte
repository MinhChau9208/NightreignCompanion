<script lang="ts">
  import {onMount} from 'svelte'
  import {GetSettings} from '../../wailsjs/go/main/App'
  import {t} from '../lib/i18n'
  import {diagnose, fmtMs, icon, label, netStats, worst} from '../lib/net'
  import Sparkline from '../lib/Sparkline.svelte'
  import FPSCard from './FPSCard.svelte'

  const stats = netStats()
  let pingThreshold = 0
  let minFps = 50

  onMount(async () => {
    try {
      const th = (await GetSettings()).network.thresholds
      pingThreshold = th.pingMs
      minFps = th.minFps
    } catch {
      // No threshold line then.
    }
  })

  $: overall = worst($stats.map((s) => s.level))
  $: diag = diagnose($stats)
</script>

<h1>{$t('nav.network')}</h1>

<section class="card summary level-{overall}">
  <span class="pill level-{overall}">{icon(overall)} {$t(label(overall))}</span>
  <span>{diag ? $t(diag) : $t('net.summaryHint')}</span>
</section>

{#if $stats.length === 0}
  <p class="muted">{$t('net.starting')}</p>
{/if}

<div class="grid">
  {#each $stats as s (s.target)}
    <section class="card">
      <header>
        <div>
          <h2>{s.target === 'gateway' ? $t('net.gateway') : s.target}</h2>
          <small class="muted">
            {s.probe.method ? s.probe.method.toUpperCase() : '—'}
            {s.probe.address && s.probe.address !== s.target ? `· ${s.probe.address}` : ''}
          </small>
        </div>
        <span class="pill level-{s.level}">{icon(s.level)} {$t(label(s.level))}</span>
      </header>

      {#if s.error}<p class="err">{s.error}</p>{/if}

      <div class="hero">
        {#if s.sent === 0}
          <span class="muted">—</span>
        {:else if s.lastLost}
          <span class="lost">{$t('net.lost')}</span>
        {:else}
          <b>{fmtMs(s.lastMs)}</b><span class="unit">ms</span>
        {/if}
      </div>

      <Sparkline history={s.history} threshold={pingThreshold} lostLabel={$t('net.lost')} />

      <dl>
        <div><dt>{$t('net.avg')}</dt><dd>{s.received ? `${fmtMs(s.avgMs)} ms` : '—'}</dd></div>
        <div><dt>Min / Max</dt><dd>{s.received ? `${fmtMs(s.minMs)} / ${fmtMs(s.maxMs)}` : '—'}</dd></div>
        <div><dt>Jitter</dt><dd>{s.received > 1 ? `${fmtMs(s.jitterMs)} ms` : '—'}</dd></div>
        <div><dt>{$t('net.loss60')}</dt><dd>{s.sent ? `${s.lossPct.toFixed(1)}%` : '—'}</dd></div>
        <div>
          <dt>{$t('net.lossSession')}</dt>
          <dd>{s.totalLost}/{s.totalSent}</dd>
        </div>
      </dl>
    </section>
  {/each}
</div>

<p class="muted foot">{$t('net.footnote')}</p>

<div class="fps">
  <FPSCard {minFps} />
</div>

<style>
  .summary {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 16px;
  }
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 16px;
  }
  header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 8px;
  }
  header h2 {
    margin: 0;
  }
  .pill {
    font-size: 12px;
    border: 1px solid var(--border);
    border-radius: 999px;
    padding: 2px 10px;
    white-space: nowrap;
  }
  .pill.level-good {
    color: var(--ok);
  }
  .pill.level-warn {
    color: var(--warn);
  }
  .pill.level-bad {
    color: var(--danger);
  }
  .pill.level-unknown {
    color: var(--muted);
  }
  .hero {
    margin: 14px 0 6px;
    font-variant-numeric: tabular-nums;
  }
  .hero b {
    font-size: 34px;
    font-weight: 600;
  }
  .unit {
    margin-left: 4px;
    color: var(--muted);
  }
  .lost {
    font-size: 20px;
    color: var(--danger);
  }
  dl {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px 16px;
    margin: 12px 0 0;
  }
  dt {
    color: var(--muted);
    font-size: 12px;
  }
  dd {
    margin: 0;
    font-variant-numeric: tabular-nums;
  }
  .err {
    color: var(--danger);
    font-size: 12px;
    word-break: break-word;
  }
  .fps {
    max-width: 460px;
  }
  .foot {
    font-size: 12px;
    margin: 16px 0;
  }
</style>
