<script lang="ts">
  import {onMount} from 'svelte'
  import {GetRelicData} from '../../wailsjs/go/main/App'
  import type {gamedata, main} from '../../wailsjs/go/models'
  import {lang, loc, t} from '../lib/i18n'
  import {categoryKey, fold, haystack, kindHintKey, kindKey, relicCategories, relicKinds, stackHintKey, stackKey} from '../lib/relic'

  let data: main.RelicData | null = null
  let error = ''

  let query = ''
  let kind = '' // '' = all
  let category = ''
  let character = '' // '' = all, '-' = generic only
  let stackOnly = false
  let open = new Set<string>()

  onMount(async () => {
    try {
      data = await GetRelicData()
    } catch (e) {
      error = String(e)
    }
  })

  $: index = (data?.effects ?? []).map((e) => ({e, text: haystack(e)}))
  $: chars = new Map<string, gamedata.Character>((data?.characters ?? []).map((c) => [c.id, c]))
  $: words = fold(query).split(/\s+/).filter(Boolean)
  $: shown = index
    .filter(
      ({e, text}) =>
        (!kind || e.kind === kind) &&
        (!category || e.categories.includes(category)) &&
        (!character || (character === '-' ? !e.character : e.character === character)) &&
        (!stackOnly || e.stacking === 'yes') &&
        words.every((w) => text.includes(w)),
    )
    .map(({e}) => e)
  $: filtered = !!(query || kind || category || character || stackOnly)

  function toggle(id: string) {
    open.has(id) ? open.delete(id) : open.add(id)
    open = open
  }

  function clear() {
    query = kind = category = character = ''
    stackOnly = false
  }
</script>

<h1>{$t('nav.relics')}</h1>

{#if error}
  <p class="err">{error}</p>
{:else if data}
  <section class="card filters">
    <input type="text" bind:value={query} placeholder={$t('relic.search')} aria-label={$t('relic.search')} />
    <div class="row">
      <div class="seg">
        <button class:on={kind === ''} on:click={() => (kind = '')}>{$t('relic.all')}</button>
        {#each relicKinds as k}
          <button class:on={kind === k} on:click={() => (kind = k)}>{$t(kindKey(k))}</button>
        {/each}
      </div>
      <select bind:value={character} aria-label={$t('relic.character')}>
        <option value="">{$t('relic.anyCharacter')}</option>
        <option value="-">{$t('relic.generic')}</option>
        {#each data.characters as c (c.id)}
          <option value={c.id}>{loc(c.name, $lang)}{c.dlc ? ' (DLC)' : ''}</option>
        {/each}
      </select>
      <label class="check"><input type="checkbox" bind:checked={stackOnly} /> {$t('relic.stackOnly')}</label>
    </div>
    <div class="cats">
      {#each relicCategories as c}
        <button class="chip" class:on={category === c} on:click={() => (category = category === c ? '' : c)}>
          {$t(categoryKey(c))}
        </button>
      {/each}
    </div>
    <p class="muted hint">{$t(kindHintKey(kind))}</p>
  </section>

  <div class="summary">
    <span class="muted">{$t('relic.count').replace('{n}', String(shown.length))}</span>
    {#if filtered}<button class="link" on:click={clear}>{$t('relic.clear')}</button>{/if}
  </div>

  <section class="card list">
    {#each shown as e (e.id)}
      {@const more = !!(e.notes || e.inGame)}
      <div class="item">
        <button class="head" on:click={() => more && toggle(e.id)} disabled={!more} aria-expanded={more ? open.has(e.id) : undefined}>
          <span class="name">
            <b>{e.name.en}</b>
            <span class="tags">
              <span class="tag">{$t(kindKey(e.kind))}</span>
              {#each e.categories as c}<span class="tag" class:bad={c === 'curse'}>{$t(categoryKey(c))}</span>{/each}
              {#if e.character}<span class="tag">{loc(chars.get(e.character)?.name, $lang)}</span>{/if}
            </span>
          </span>
          <span class="effect">
            {#if e.effect}{loc(e.effect, $lang)}{:else}<span class="muted">{$t('relic.asNamed')}</span>{/if}
          </span>
          <span class="stack stack-{e.stacking}" title={$t(stackHintKey(e.stacking))}>{$t(stackKey(e.stacking))}</span>
          <span class="more muted">{more ? (open.has(e.id) ? '▴' : '▾') : ''}</span>
        </button>
        {#if open.has(e.id)}
          <div class="details">
            {#if e.inGame}<p><span class="muted">{$t('relic.inGame')}:</span> {e.inGame}</p>{/if}
            {#if e.notes}<p><span class="muted">{$t('relic.notes')}:</span> {loc(e.notes, $lang)}</p>{/if}
          </div>
        {/if}
      </div>
    {:else}
      <p class="muted">{$t('relic.empty')}</p>
    {/each}
  </section>
  <p class="muted foot">{$t('relic.source')}</p>
{/if}

<style>
  .filters {
    display: grid;
    gap: 12px;
    margin-bottom: 12px;
  }
  input[type='text'] {
    background: var(--hover);
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 8px 12px;
    font: inherit;
  }
  .row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 12px;
  }
  .check {
    display: flex;
    align-items: center;
    gap: 6px;
    accent-color: var(--accent);
  }
  .cats {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
  .hint {
    font-size: 12px;
    margin: 0;
  }
  .summary {
    display: flex;
    gap: 12px;
    align-items: baseline;
    margin: 0 0 8px;
    font-size: 13px;
  }
  .list {
    padding: 4px 18px;
  }
  .item + .item {
    border-top: 1px solid var(--border);
  }
  .head {
    display: grid;
    grid-template-columns: minmax(220px, 1.2fr) 2fr 104px 14px;
    gap: 14px;
    align-items: start;
    width: 100%;
    padding: 10px 0;
    background: none;
    border: none;
    color: var(--text);
    font: inherit;
    text-align: left;
    cursor: pointer;
  }
  .head:disabled {
    cursor: default;
  }
  .name {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .tags {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }
  .tag {
    font-size: 11px;
    color: var(--muted);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 0 5px;
  }
  .tag.bad {
    color: var(--danger);
  }
  .effect {
    white-space: pre-line;
  }
  .stack {
    font-size: 12px;
    text-align: right;
  }
  .stack-yes {
    color: var(--ok);
  }
  .stack-no {
    color: var(--warn);
  }
  .stack-na,
  .stack-unknown,
  .stack-varies {
    color: var(--muted);
  }
  .details {
    padding: 0 0 12px;
    font-size: 13px;
    white-space: pre-line;
  }
  .details p {
    margin: 0 0 6px;
  }
  .err {
    color: var(--danger);
  }
  .foot {
    font-size: 12px;
    margin-top: 16px;
  }
</style>
