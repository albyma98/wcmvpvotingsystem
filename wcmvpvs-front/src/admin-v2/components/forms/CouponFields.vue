<script setup>
import { computed, inject, ref } from 'vue';
const props = defineProps({ draft: { type: Object, required: true }, identity: { type: String, required: true } });
const c = inject('admin-v2');
const search = ref('');
const events = computed(() => c.events.filter(event => `${c.couponMatchLabel(event)} ${event.start_datetime}`.toLowerCase().includes(search.value.toLowerCase())));
</script>

<template>
  <div class="v2-grid v2-grid-2">
    <label class="v2-field">Titolo<input v-model.trim="draft.title" required placeholder="Es. Sconto per i tifosi" /></label>
    <label class="v2-field">Partner<select v-model.number="draft.sponsorId" required><option :value="0" disabled>Seleziona un partner</option><option v-for="partner in c.partners" :key="partner.id" :value="partner.id">{{ partner.displayName || partner.username }}</option></select></label>
    <label class="v2-field">Stato<select v-model="draft.status"><option v-for="status in c.couponStatusOptions" :key="status" :value="status">{{ c.getCouponStatusLabel(status) }}</option></select></label>
    <label class="v2-field">Limite utilizzi<input v-model.number="draft.maxUses" type="number" min="0" /><small>0 significa illimitato.</small></label>
    <label class="v2-field">Disponibile dal<input v-model="draft.startDateInput" type="datetime-local" /></label>
    <label class="v2-field">Disponibile fino al<input v-model="draft.endDateInput" type="datetime-local" /></label>
  </div>
  <label class="v2-field">Descrizione breve<textarea v-model.trim="draft.shortDesc" rows="3" placeholder="Descrivi il vantaggio e le condizioni" /></label>
  <div class="v2-grid v2-grid-2">
    <label class="v2-field">URL immagine<input v-model.trim="draft.imageUrl" type="text" inputmode="url" placeholder="https://…" @input="c.syncCouponImageSource(draft)" /></label>
    <label class="v2-field">Oppure carica un’immagine<input type="file" accept="image/*" @change="c.run(`coupon-image-${identity}`, () => c.handleCouponImageFileChange($event, draft))" /></label>
  </div>
  <div v-if="draft.imagePreview" class="v2-media-preview"><img :src="draft.imagePreview" alt="Anteprima del coupon" /><button type="button" class="v2-button" @click="c.clearCouponImage(draft)">Rimuovi immagine</button></div>
  <fieldset class="v2-fieldset"><legend>Eventi associati</legend><label class="v2-field">Cerca evento<input v-model="search" type="search" placeholder="Squadra, data o partita" /></label><div class="v2-check-list"><label v-for="event in events" :key="event.id" class="v2-check"><input v-model="draft.matchIds" type="checkbox" :value="event.id" /><span>{{ c.couponMatchLabel(event) }}<small>{{ c.formatEventDate(event.start_datetime) }}</small></span></label></div><p v-if="!c.events.length" class="v2-muted">Crea un evento per associarlo al coupon.</p><p v-else-if="!events.length" class="v2-muted">Nessun evento corrisponde alla ricerca.</p></fieldset>
</template>

<style scoped>
.v2-fieldset{min-width:0;margin:0;padding:18px;border:1px solid var(--v2-line,#dce2e9);border-radius:12px}.v2-fieldset legend{padding:0 8px;font-weight:650}.v2-check-list{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:8px;max-height:270px;overflow:auto;margin-top:12px}.v2-check{display:flex;align-items:center;gap:10px;padding:12px;border:1px solid var(--v2-line,#dce2e9);border-radius:8px;cursor:pointer}.v2-check small{display:block;margin-top:5px;font-size:12px;color:var(--v2-muted,#526078)}.v2-media-preview{display:flex;align-items:center;flex-wrap:wrap;gap:16px}.v2-media-preview img{height:100px;width:160px;object-fit:contain;border:1px solid var(--v2-line,#dce2e9);border-radius:10px}@media(max-width:600px){.v2-check-list{grid-template-columns:1fr}}
</style>
