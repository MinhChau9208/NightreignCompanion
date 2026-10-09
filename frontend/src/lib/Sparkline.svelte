<script lang="ts">
  // A per-second history as a single 2px line (RTT or FPS). Negative values
  // mean "lost": they break the line and get a tick on the baseline. A dashed
  // line marks the threshold (max ping or min FPS).
  import {fmtMs} from './net'

  export let history: number[] = []
  export let windowSize = 60
  export let threshold = 0
  export let format = (v: number) => `${fmtMs(v)} ms`
  export let lostLabel = 'lost'

  const W = 300
  const H = 64
  const PAD = 4

  let hover: number | null = null
  let svg: SVGSVGElement

  $: received = history.filter((v) => v >= 0)
  $: top = Math.max(20, ...received) * 1.15
  // Show the threshold only when it is near the data, so it never squashes the line.
  $: showThreshold = threshold > 0 && threshold <= top * 1.5
  $: yMax = showThreshold ? Math.max(top, threshold * 1.1) : top
  $: step = (W - PAD * 2) / Math.max(1, windowSize - 1)
  // Right-align so the newest sample is always at the right edge.
  $: offset = windowSize - history.length
  $: x = (i: number) => PAD + (i + offset) * step
  $: y = (v: number) => H - PAD - (v / yMax) * (H - PAD * 2)

  $: path = history
    .map((v, i) => {
      if (v < 0) return ''
      const cmd = i > 0 && history[i - 1] >= 0 ? 'L' : 'M'
      return `${cmd}${x(i).toFixed(1)},${y(v).toFixed(1)}`
    })
    .join('')

  function onMove(e: MouseEvent) {
    const r = svg.getBoundingClientRect()
    const px = ((e.clientX - r.left) / r.width) * W
    const i = Math.round((px - PAD) / step) - offset
    hover = i >= 0 && i < history.length ? i : null
  }
</script>

<div class="wrap">
  <svg
    bind:this={svg}
    viewBox="0 0 {W} {H}"
    preserveAspectRatio="none"
    role="img"
    aria-label="RTT history"
    on:mousemove={onMove}
    on:mouseleave={() => (hover = null)}
  >
    <line class="base" x1={PAD} x2={W - PAD} y1={H - PAD} y2={H - PAD} />
    {#if showThreshold}
      <line class="threshold" x1={PAD} x2={W - PAD} y1={y(threshold)} y2={y(threshold)} />
    {/if}
    <path d={path} />
    {#each history as v, i}
      {#if v < 0}<rect class="lost" x={x(i) - 1.5} y={H - PAD - 6} width="3" height="6" rx="1" />{/if}
    {/each}
    {#if hover !== null}
      <line class="cross" x1={x(hover)} x2={x(hover)} y1={PAD} y2={H - PAD} />
      {#if history[hover] >= 0}<circle cx={x(hover)} cy={y(history[hover])} r="3.5" />{/if}
    {/if}
  </svg>
  {#if hover !== null}
    <div class="tip" style="left: {(x(hover) / W) * 100}%">
      {history[hover] >= 0 ? format(history[hover]) : lostLabel}
      <span>−{history.length - 1 - hover}s</span>
    </div>
  {/if}
</div>

<style>
  .wrap {
    position: relative;
  }
  svg {
    display: block;
    width: 100%;
    height: 64px;
    cursor: crosshair;
  }
  path {
    fill: none;
    stroke: var(--accent);
    stroke-width: 2;
    stroke-linejoin: round;
    stroke-linecap: round;
    vector-effect: non-scaling-stroke;
  }
  .base {
    stroke: var(--border);
    stroke-width: 1;
    vector-effect: non-scaling-stroke;
  }
  .threshold {
    stroke: var(--muted);
    stroke-width: 1;
    stroke-dasharray: 4 4;
    vector-effect: non-scaling-stroke;
  }
  .cross {
    stroke: var(--muted);
    stroke-width: 1;
    vector-effect: non-scaling-stroke;
  }
  circle {
    fill: var(--accent);
    stroke: var(--card);
    stroke-width: 2;
    vector-effect: non-scaling-stroke;
  }
  .lost {
    fill: var(--danger);
  }
  .tip {
    position: absolute;
    top: -30px;
    transform: translateX(-50%);
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 3px 8px;
    font-size: 12px;
    white-space: nowrap;
    pointer-events: none;
    font-variant-numeric: tabular-nums;
  }
  .tip span {
    color: var(--muted);
    margin-left: 4px;
  }
</style>
