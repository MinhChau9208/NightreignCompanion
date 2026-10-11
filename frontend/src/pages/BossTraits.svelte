<script lang="ts">
  import type {gamedata} from '../../wailsjs/go/models'
  import {t} from '../lib/i18n'
  import {damageKey, easyStatuses, signed, statusKey, weakAndResist} from '../lib/boss'

  export let form: gamedata.BossForm

  $: wr = weakAndResist(form)
  $: easy = easyStatuses(form)
</script>

<dl class="traits">
  <dt>{$t('boss.weak')}</dt>
  <dd>
    {#each wr.weak as d}<span class="trait weak">▼ {$t(damageKey(d.type))} {signed(d.value)}%</span>{/each}
    {#if wr.weak.length === 0}<span class="muted">—</span>{/if}
  </dd>
  <dt>{$t('boss.resist')}</dt>
  <dd>
    {#each wr.resist as d}<span class="trait resist">▲ {$t(damageKey(d.type))} {signed(d.value)}%</span>{/each}
    {#if wr.resist.length === 0}<span class="muted">—</span>{/if}
  </dd>
  <dt>{$t('boss.easyStatus')}</dt>
  <dd>
    {#each easy as s}<span class="trait weak">{$t(statusKey(s))}</span>{/each}
    {#if easy.length === 0}<span class="muted">—</span>{/if}
  </dd>
  <dt>{$t('boss.level.immune')}</dt>
  <dd>
    {#each form.immune ?? [] as s}<span class="trait">{$t(statusKey(s))}</span>{/each}
    {#if !form.immune?.length}<span class="muted">—</span>{/if}
  </dd>
</dl>

<style>
  .traits {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 6px 12px;
    margin: 0;
    align-items: baseline;
  }
  dt {
    color: var(--muted);
    font-size: 12px;
  }
  dd {
    margin: 0;
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }
  .trait {
    font-size: 12px;
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 1px 7px;
    color: var(--muted);
  }
  .trait.weak {
    color: var(--ok);
  }
  .trait.resist {
    color: var(--danger);
  }
</style>
