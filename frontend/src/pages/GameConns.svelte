<script lang="ts">
  import {StartFPS} from '../../wailsjs/go/main/App'
  import {t} from '../lib/i18n'
  import {fpsRunning} from '../lib/fps'
  import {flowLevel, gameNet, gapLevel, hasPing, kind, pingNoReply} from '../lib/gamenet'
  import {fmtMs, icon, label, worst} from '../lib/net'

  const status = gameNet()
  const running = fpsRunning()
  let busy = false
  let startError = ''

  async function start() {
    busy = true
    startError = ''
    try {
      await StartFPS()
    } catch (e) {
      startError = String(e)
    } finally {
      busy = false
    }
  }

  $: s = $status
  $: flows = s?.flows ?? []
  $: overall = worst(flows.map(flowLevel))
  const rate = (v: number) => (v < 10 ? v.toFixed(1) : Math.round(v).toString())
</script>

<section class="card">
  <header>
    <h2>{$t('gnet.title')}</h2>
    {#if flows.some((f) => f.proto === 'udp')}
      <span class="pill level-{overall}">{icon(overall)} {$t(label(overall))}</span>
    {/if}
  </header>

  {#if !$running}
    <p class="muted">{$t('gnet.intro')}</p>
    <button class="primary" disabled={busy} on:click={start}>{$t('helper.start')}</button>
    {#if startError}<p class="err">{startError}</p>{/if}
  {:else if !s || s.state === 'starting'}
    <p class="muted">{$t('fps.starting')}</p>
  {:else if s.state === 'waiting'}
    <p class="muted">{$t('gnet.waiting')}</p>
  {:else}
    {#if s.error}<p class="err">{s.error}</p>{/if}
    {#if flows.length === 0}
      <p class="muted">{$t('gnet.idle')}</p>
    {:else}
      <div class="table" role="table">
        <div class="row head" role="row">
          <span role="columnheader">{$t('gnet.remote')}</span>
          <span role="columnheader">Ping</span>
          <span role="columnheader">{$t('gnet.gap')}</span>
          <span role="columnheader">{$t('gnet.pkts')}</span>
          <span role="columnheader">{$t('gnet.kbps')}</span>
        </div>
        {#each flows as f (f.proto + f.remote)}
          <div class="row" role="row" class:stale={f.idleMs > 5000}>
            <span role="cell">
              <b>{$t(kind(f.kind))}</b>
              <small class="muted">{f.proto.toUpperCase()} · {f.remote}{f.via ? ` · ${$t('gnet.via')} ${f.via}` : ''}</small>
            </span>
            <span role="cell" class="num">
              {#if f.ping && hasPing(f)}
                <span class="level-{f.ping.level}">{icon(f.ping.level)}</span>
                <span title={f.pingAddr ? `${$t('gnet.pingApprox')} ${f.pingAddr}` : ''}>{f.pingAddr ? '≈' : ''}{fmtMs(f.ping.avgMs)} ms</span>
                {#if f.ping.lossPct > 0}<small class="muted">· {f.ping.lossPct.toFixed(0)}%</small>{/if}
              {:else if pingNoReply(f)}
                <small class="muted">{$t('gnet.noReply')}</small>
              {:else}
                <span class="muted">—</span>
              {/if}
            </span>
            <span role="cell" class="num">
              {#if f.sendOnly}
                <small class="muted" title={$t('gnet.sendOnlyHint')}>{$t('gnet.sendOnly')}</small>
              {:else if f.proto === 'udp' && f.pktsInPerSec > 0}
                <span class="level-{gapLevel(f)}">{icon(gapLevel(f))}</span>
                {Math.round(f.maxGapMs)} ms
              {:else}
                <span class="muted">—</span>
              {/if}
            </span>
            <span role="cell" class="num">{rate(f.pktsInPerSec)} · {rate(f.pktsOutPerSec)}</span>
            <span role="cell" class="num">{rate(f.kbpsIn)} · {rate(f.kbpsOut)}</span>
          </div>
        {/each}
      </div>
    {/if}
    <p class="muted foot">{$t('gnet.footnote')}</p>
  {/if}
</section>

<style>
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
  .table {
    margin-top: 12px;
    display: grid;
    gap: 2px;
  }
  .row {
    display: grid;
    grid-template-columns: minmax(180px, 2fr) 1.2fr 1fr 1fr 1fr;
    gap: 12px;
    align-items: center;
    padding: 6px 0;
    border-top: 1px solid var(--border);
  }
  .row.head {
    border-top: none;
    color: var(--muted);
    font-size: 12px;
  }
  .row.stale {
    opacity: 0.55;
  }
  .row span:first-child {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .row small {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .num {
    font-variant-numeric: tabular-nums;
  }
  .level-good {
    color: var(--ok);
  }
  .level-warn {
    color: var(--warn);
  }
  .level-bad {
    color: var(--danger);
  }
  .level-unknown {
    color: var(--muted);
  }
  .err {
    color: var(--danger);
    font-size: 12px;
    word-break: break-word;
  }
  .foot {
    font-size: 12px;
    margin: 12px 0 0;
  }
</style>
