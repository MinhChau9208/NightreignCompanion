<script lang="ts">
  import {TestBannerOCR, TestOCR} from '../../wailsjs/go/main/App'
  import type {config, main} from '../../wailsjs/go/models'
  import {t} from '../lib/i18n'
  import OCRRegion from './OCRRegion.svelte'

  export let ocr: config.OCRSettings

  // config.DefaultOCRRegion, DefaultDayRegion and DefaultMessageRegion, in percent.
  const bossDefaults = {x: 22, y: 68, w: 42, h: 20}
  const dayDefaults = {x: 25, y: 45, w: 50, h: 20}
  const messageDefaults = {x: 60, y: 14, w: 40, h: 8}

  const bossFound = (r: main.OCRTest) => r.match
  function bannerFound(r: main.OCRTest) {
    if (r.match === 'day') return $t('sync.match.day').replaceAll('{n}', String(r.day))
    if (r.match === 'shrink') return $t('sync.match.shrink')
    return ''
  }
</script>

<section class="card">
  <h2>{$t('ocr.title')}</h2>
  <label class="check"><input type="checkbox" bind:checked={ocr.enabled} /> {$t('ocr.enabled')}</label>
  <p class="muted hint">{$t('ocr.hint')}</p>
  <OCRRegion
    bind:region={ocr.region}
    defaults={bossDefaults}
    title="ocr.region"
    testHint="ocr.testHint"
    test={TestOCR}
    describe={bossFound}
    noMatch="ocr.noMatch"
  />
</section>

<section class="card">
  <h2>{$t('sync.title')}</h2>
  <label class="check"><input type="checkbox" bind:checked={ocr.timerSync} /> {$t('sync.enabled')}</label>
  <p class="muted hint">{$t('sync.hint')}</p>
  <OCRRegion
    bind:region={ocr.dayRegion}
    defaults={dayDefaults}
    title="sync.dayRegion"
    testHint="sync.dayHint"
    test={TestBannerOCR}
    describe={bannerFound}
    noMatch="sync.noMatch"
  />
  <OCRRegion
    bind:region={ocr.messageRegion}
    defaults={messageDefaults}
    title="sync.messageRegion"
    testHint="sync.messageHint"
    test={TestBannerOCR}
    describe={bannerFound}
    noMatch="sync.noMatch"
  />
</section>

<style>
  .check {
    display: flex;
    gap: 8px;
    align-items: center;
    padding: 4px 0;
  }
  .hint {
    font-size: 12px;
  }
</style>
