<script lang="ts">
  import {t} from '../lib/i18n'

  export let title: string
  export let phase: string
</script>

<h1>{title}</h1>
<section class="card">
  <h2>{$t('soon.title')}</h2>
  <p class="muted">{$t('soon.body')} <b>{phase}</b> — docs/SCOPE.md §5.</p>
</section>
