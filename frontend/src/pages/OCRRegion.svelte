<script lang="ts">
  import {capture, type main} from '../../wailsjs/go/models'
  import {t, type Key} from '../lib/i18n'

  // A capture region edited in percent of the game window, with a button
  // that captures it now and shows what OCR reads there.
  export let region: capture.Region
  export let defaults: {x: number; y: number; w: number; h: number} // percent
  export let title: Key
  export let testHint: Key
  export let test: (r: capture.Region) => Promise<main.OCRTest>
  // describe names what the test recognised, or '' for nothing.
  export let describe: (r: main.OCRTest) => string
  export let noMatch: Key

  const fields = ['x', 'y', 'w', 'h'] as const

  // Written back to region as fractions. The settings are loaded before
  // this component is created.
  let pct = {x: round(region.x * 100), y: round(region.y * 100), w: round(region.w * 100), h: round(region.h * 100)}
  $: region = capture.Region.createFrom({x: pct.x / 100, y: pct.y / 100, w: pct.w / 100, h: pct.h / 100})

  let result: main.OCRTest | null = null
  let testErr = ''
  let testing = false

  function round(v: number) {
    return Math.round(v * 10) / 10
  }

  async function runTest() {
    testing = true
    testErr = ''
    try {
      result = await test(region)
    } catch (e) {
      result = null
      testErr = String(e)
    }
    testing = false
  }

  $: found = result ? describe(result) : ''
</script>

<h3>{$t(title)}</h3>
<div class="region">
  {#each fields as f}
    <label>{$t(`ocr.${f}`)} (%)<input type="number" min="0" max="100" step="0.5" bind:value={pct[f]} /></label>
  {/each}
</div>
<div class="row">
  <button class="secondary" on:click={runTest} disabled={testing}>{$t('ocr.test')}</button>
  <button class="link" on:click={() => (pct = {...defaults})}>{$t('ocr.defaults')}</button>
</div>
<p class="muted hint">{$t(testHint)}</p>

{#if testErr}<p class="err">{testErr}</p>{/if}
{#if result}
  <img class="shot" src={result.image} alt="" />
  <p class="muted hint">{$t('ocr.window')} {result.window} · {$t('ocr.captured')} {result.region}</p>
  <pre>{result.lines.length ? result.lines.join('\n') : $t('ocr.noText')}</pre>
  <p>
    {#if found}
      ✔ {$t('ocr.found')} <b>{found}</b>
    {:else}
      {$t(noMatch)}
    {/if}
  </p>
{/if}

<style>
  .region {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 12px;
    margin-bottom: 12px;
  }
  .region label {
    display: grid;
    gap: 4px;
    color: var(--muted);
    font-size: 12px;
  }
  input[type='number'] {
    background: var(--hover);
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 6px 10px;
    font: inherit;
  }
  .row {
    display: flex;
    align-items: center;
    gap: 16px;
  }
  .hint {
    font-size: 12px;
  }
  h3 {
    font-size: 13px;
    font-weight: 600;
    margin: 14px 0 8px;
  }
  .shot {
    display: block;
    max-width: 100%;
    margin-top: 12px;
    border: 1px solid var(--border);
    border-radius: 6px;
  }
  pre {
    background: var(--hover);
    border-radius: 6px;
    padding: 8px 10px;
    font-size: 12px;
    white-space: pre-wrap;
  }
  .err {
    color: var(--danger);
  }
  button.secondary {
    background: none;
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 6px 14px;
    font: inherit;
    cursor: pointer;
  }
  button.secondary:disabled {
    opacity: 0.4;
  }
</style>
