<script setup>
import { computed, reactive, ref } from 'vue'
import { useMasterWorkspace } from '../workspace'

const props = defineProps({
  tournament: Boolean,
  busy: Boolean,
  error: Object,
  listPath: { type: String, required: true },
})
const emit = defineEmits(['submit', 'clear-error'])
const { follow, useDirty } = useMasterWorkspace()

const form = reactive(props.tournament
  ? { name: '', format: 'BEACH VOLLEY 4X4', dateLabel: '', location: '' }
  : {
      name: '', slug: '', city: '', tagline: '', logoUrl: '', heroImageUrl: '',
      primaryColor: '#172554', secondaryColor: '#f59e0b', seasonLabel: 'Stagione 2026/27',
      categoriesText: 'Prima Divisione Maschile\nPrima Divisione Femminile\nGiovanile 1\nGiovanile 2',
    })
const initial = JSON.stringify(form)
const validation = ref('')
const nameError = computed(() => validation.value || (props.error?.field === 'name' ? props.error.message : ''))
const slugError = computed(() => props.error?.field === 'slug' ? props.error.message : '')
useDirty(() => JSON.stringify(form) !== initial)

function submit() {
  if (!form.name.trim()) {
    validation.value = 'Inserisci il nome.'
    return
  }
  validation.value = ''
  if (props.tournament) {
    emit('submit', { ...form })
    return
  }
  const { categoriesText, ...details } = form
  emit('submit', { ...details, categories: categoriesText.split('\n').map(line => line.trim()).filter(Boolean) })
}

function clearError(field) {
  if (field === 'name') validation.value = ''
  if (props.error?.field === field) emit('clear-error')
}
</script>

<template>
  <form class="m-stack" @submit.prevent="submit">
    <section class="m-panel" aria-labelledby="portal-profile-heading">
      <header class="m-panel-head">
        <div>
          <h2 id="portal-profile-heading">{{ tournament ? 'Dati del torneo' : 'Profilo societario' }}</h2>
          <p v-if="tournament">Il pannello dell’organizzatore consente di gestire squadre, calendario e sponsor dopo la creazione.</p>
          <p v-else>Il portale include il sito pubblico, le categorie iniziali e un accesso dedicato al CMS.</p>
        </div>
      </header>
      <fieldset class="m-fields portal-fields" :disabled="busy">
        <label class="m-field">
          <span>{{ tournament ? 'Nome del torneo' : 'Nome della società' }} (obbligatorio)</span>
          <input v-model="form.name" class="m-input" :placeholder="tournament ? 'Sunset Beach Cup' : 'Volley Club Aurora'" required
            :aria-invalid="nameError ? 'true' : undefined" :aria-describedby="nameError ? 'portal-name-error' : undefined"
            @input="clearError('name')">
          <small v-if="nameError" id="portal-name-error" class="m-error" role="alert">{{ nameError }}</small>
        </label>
        <template v-if="tournament">
          <label class="m-field"><span>Formato</span><input v-model="form.format" class="m-input" placeholder="BEACH VOLLEY 4X4"></label>
          <label class="m-field"><span>Date</span><input v-model="form.dateLabel" class="m-input" placeholder="8–11 giugno 2027"></label>
          <label class="m-field"><span>Luogo</span><input v-model="form.location" class="m-input" placeholder="Lido di Classe, RA"></label>
        </template>
        <template v-else>
          <label class="m-field">
            <span>Slug facoltativo</span>
            <input v-model="form.slug" class="m-input" placeholder="volley-club-aurora" spellcheck="false"
              :aria-invalid="slugError ? 'true' : undefined" :aria-describedby="slugError ? 'portal-slug-error' : 'portal-slug-hint'" @input="clearError('slug')">
            <small v-if="slugError" id="portal-slug-error" class="m-error" role="alert">{{ slugError }}</small>
            <small v-else id="portal-slug-hint" class="m-muted-text">Se lasciato vuoto, viene ricavato dal nome.</small>
          </label>
          <label class="m-field"><span>Città</span><input v-model="form.city" class="m-input" placeholder="Bari"></label>
          <label class="m-field"><span>Stagione</span><input v-model="form.seasonLabel" class="m-input" placeholder="Stagione 2026/27"></label>
          <label class="m-field portal-wide"><span>Claim</span><input v-model="form.tagline" class="m-input" placeholder="Una società, quattro squadre, una sola passione."></label>
        </template>
      </fieldset>
    </section>

    <template v-if="!tournament">
      <section class="m-panel" aria-labelledby="portal-brand-heading">
        <header class="m-panel-head"><div><h2 id="portal-brand-heading">Identità del sito</h2><p>Immagini e colori della pagina pubblica.</p></div></header>
        <fieldset class="m-fields portal-fields" :disabled="busy">
          <label class="m-field"><span>URL del logo</span><input v-model="form.logoUrl" class="m-input" inputmode="url" placeholder="https://societa.it/logo.png"></label>
          <label class="m-field"><span>URL dell’immagine di copertina</span><input v-model="form.heroImageUrl" class="m-input" inputmode="url" placeholder="https://societa.it/copertina.jpg"></label>
          <label class="m-field"><span>Colore primario</span><span class="portal-color"><input v-model="form.primaryColor" class="m-input" type="color"><code>{{ form.primaryColor }}</code></span></label>
          <label class="m-field"><span>Colore accento</span><span class="portal-color"><input v-model="form.secondaryColor" class="m-input" type="color"><code>{{ form.secondaryColor }}</code></span></label>
        </fieldset>
      </section>
      <section class="m-panel" aria-labelledby="portal-categories-heading">
        <header class="m-panel-head"><div><h2 id="portal-categories-heading">Categorie iniziali</h2><p>Potrai gestire le squadre associate dal CMS ClubHub.</p></div></header>
        <label class="m-field"><span>Categorie, una per riga</span><textarea v-model="form.categoriesText" class="m-input" rows="5" placeholder="Prima Divisione Maschile" :disabled="busy" /></label>
      </section>
    </template>

    <div v-if="error?.message && !error.field" class="m-error" role="alert">{{ error.message }}</div>
    <div class="m-actions portal-form-actions">
      <a v-if="!busy" class="m-btn" :href="listPath" @click="follow">Annulla</a>
      <button class="m-btn m-primary" type="submit" :disabled="busy">
        {{ busy ? 'Creazione…' : tournament ? 'Crea torneo' : 'Crea ClubHub' }}
      </button>
    </div>
  </form>
</template>

<style scoped>
.portal-fields { padding: 0; margin: 0; min-inline-size: 0; border: 0; }
.portal-wide { grid-column: 1 / -1; }
.portal-color { display: flex; align-items: center; gap: 10px; }
.portal-color input { flex: 0 0 54px; max-width: 54px; cursor: pointer; }
.portal-form-actions { justify-content: flex-end; }
</style>
