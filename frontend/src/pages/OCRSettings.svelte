<script lang="ts">
  import {TestOCR} from '../../wailsjs/go/main/App'
  import {capture, type config, type main} from '../../wailsjs/go/models'
  import {t} from '../lib/i18n'

  export let ocr: config.OCRSettings

  // The default region (config.DefaultOCRRegion), in percent.
  const defaults = {x: 22, y: 68, w: 42, h: 20}
  const fields = ['x', 'y', 'w', 'h'] as const

  // Edited in percent and written back to ocr.region as fractions. The
  // settings are loaded before this component is created.
  let pct = {x: round(ocr.region.x * 100), y: round(ocr.region.y * 100), w: round(ocr.region.w * 100), h: round(ocr.region.h * 100)}
  $: ocr.region = capture.Region.createFrom({x: pct.x / 100, y: pct.y / 100, w: pct.w / 100, h: pct.h / 100})

  let test: main.OCRTest | null = null
  let testErr = ''
  let testing = false

  function round(v: number) {
    return Math.round(v * 10) / 10
  }

  async function runTest() {
    testing = true
    testErr = ''
    try {
      test = await TestOCR(ocr.region)
    } catch (e) {
      test = null
      testErr = String(e)
    }
    testing = false
  }
</script>

<section class="card">
  <h2>{$t('ocr.title')}</h2>
  <label class="check"><input type="checkbox" bind:checked={ocr.enabled} /> {$t('ocr.enabled')}</label>
  <p class="muted hint">{$t('ocr.hint')}</p>

  <h3>{$t('ocr.region')}</h3>
  <div class="region">
    {#each fields as f}
      <label>{$t(`ocr.${f}`)} (%)<input type="number" min="0" max="100" step="0.5" bind:value={pct[f]} /></label>
    {/each}
  </div>
  <div class="row">
    <button class="secondary" on:click={runTest} disabled={testing}>{$t('ocr.test')}</button>
    <button class="link" on:click={() => (pct = {...defaults})}>{$t('ocr.defaults')}</button>
  </div>
  <p class="muted hint">{$t('ocr.testHint')}</p>

  {#if testErr}<p class="err">{testErr}</p>{/if}
  {#if test}
    <img class="shot" src={test.image} alt="" />
    <p class="muted hint">{$t('ocr.window')} {test.window} · {$t('ocr.captured')} {test.region}</p>
    <pre>{test.lines.length ? test.lines.join('\n') : $t('ocr.noText')}</pre>
    <p>
      {#if test.match}
        ✔ {$t('ocr.found')} <b>{test.match}</b>
      {:else}
        {$t('ocr.noMatch')}
      {/if}
    </p>
  {/if}
</section>

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
  .check {
    display: flex;
    gap: 8px;
    align-items: center;
    padding: 4px 0;
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
