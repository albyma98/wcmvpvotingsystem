<script setup>
import { computed } from 'vue'
import PortalCopy from './PortalCopy.vue'

const props = defineProps({
  credentials: { type: Object, required: true },
  tournament: Boolean,
})
defineEmits(['dismiss'])

const rows = computed(() => {
  const data = props.credentials
  const links = [
    [props.tournament ? 'Portale tifosi' : 'Sito pubblico', data.publicPath],
    [props.tournament ? 'Pannello admin' : 'CMS ClubHub', data.adminPath],
  ].filter(([, path]) => path)
    .map(([label, path]) => [label, new URL(path, window.location.origin).href])
  return [...links, ['Username', data.adminUsername], ['Password', data.adminPassword]].filter(([, value]) => value)
})
</script>

<template>
  <section class="m-panel portal-credentials" aria-label="Nuove credenziali">
    <header class="m-panel-head">
      <div>
        <h2>{{ credentials.adminPath ? 'Portale creato' : 'Password aggiornata' }}</h2>
        <p>La password è visibile solo ora. Copiala prima di chiudere questa scheda.</p>
        <p v-if="tournament && !credentials.adminPath">Le sessioni attive del pannello torneo sono state disconnesse.</p>
      </div>
      <button class="m-btn" type="button" @click="$emit('dismiss')">Chiudi credenziali</button>
    </header>
    <dl class="portal-credentials-list">
      <div v-for="[label, value] in rows" :key="label">
        <dt>{{ label }}</dt>
        <dd><code>{{ value }}</code><PortalCopy :value="value" :label="`Copia ${label.toLowerCase()}`" /></dd>
      </div>
    </dl>
  </section>
</template>

<style scoped>
.portal-credentials-list { margin: 0; }
.portal-credentials-list > div { display: grid; grid-template-columns: minmax(110px, .22fr) minmax(0, 1fr); gap: 6px 16px; padding: 12px 0; }
.portal-credentials-list dt { font-weight: 600; }
.portal-credentials-list dd { display: flex; flex-wrap: wrap; gap: 8px 12px; align-items: center; min-width: 0; margin: 0; }
.portal-credentials-list code { overflow-wrap: anywhere; }
@media (max-width: 520px) { .portal-credentials-list > div { grid-template-columns: 1fr; } }
</style>
