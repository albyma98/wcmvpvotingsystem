<script setup>
import { computed, onMounted, onUnmounted, ref, shallowRef } from 'vue'
import { clubMVPPath } from '../clubDomain'

const props = defineProps({
  match: { type: Object, required: true },
  clubName: { type: String, required: true },
  slug: { type: String, required: true },
  customDomain: Boolean,
  request: { type: Function, required: true },
})
defineEmits(['close'])

const path = `/matches/${props.match.id}/mvp`
const publicPath = computed(() => clubMVPPath(props.slug, props.customDomain, props.match.id))
const state = shallowRef(null)
const heading = ref(null)
const showCallups = ref(false)
const showRanking = ref(false)
const selected = ref([])
const busy = ref(false)
const loading = ref(false)
const error = ref('')
const notice = ref('')
const controller = new AbortController()
let timer
let stream
let connected = false
let pendingRefresh = false

const eligible = computed(() => state.value?.callups.filter(player => player.eligible) ?? [])
const unavailable = computed(() => state.value?.callups.filter(player => !player.eligible) ?? [])
const locked = computed(() => state.value?.final || state.value?.totalVotes > 0)
const dirty = computed(() => selected.value.length !== eligible.value.length ||
  eligible.value.some(player => !selected.value.includes(player.id)))
const leaders = computed(() => state.value?.callups.filter(player => player.leader) ?? [])
const status = computed(() => state.value.final ? 'Conclusa' : state.value.open ? 'Aperta' : 'Chiusa')
const playerName = player => [player.firstName, player.lastName].filter(Boolean).join(' ')
const percent = new Intl.NumberFormat('it-IT', { maximumFractionDigits: 1 })
const refreshError = 'Impossibile aggiornare MVP Light. Riprova.'

function applyState(next, keepDraft) {
  state.value = next
  if (next.final) closeStream()
  if (keepDraft && !locked.value) {
    const roster = new Set(next.roster.map(player => player.id))
    selected.value = selected.value.filter(id => roster.has(id))
  } else {
    selected.value = eligible.value.map(player => player.id)
  }
}

async function refresh() {
  if (controller.signal.aborted) return
  if (loading.value || busy.value) {
    pendingRefresh = true
    return
  }
  pendingRefresh = false
  loading.value = true
  try {
    const next = await props.request(path, { signal: controller.signal })
    if (!controller.signal.aborted) {
      applyState(next, Boolean(state.value && dirty.value))
      if (error.value === refreshError) error.value = ''
    }
  } catch (cause) {
    if (cause.name !== 'AbortError') {
      pendingRefresh = true
      error.value = refreshError
    }
  } finally {
    loading.value = false
  }
}

async function update(change, success) {
  if (busy.value || loading.value) return
  busy.value = true
  error.value = ''
  notice.value = ''
  try {
    const next = await props.request(path, { method: 'PUT', body: JSON.stringify(change) })
    if (controller.signal.aborted) return
    applyState(next, !change.playerIds && dirty.value)
    notice.value = success
  } catch (cause) {
    error.value = {
      callups_locked: 'È già stato registrato un voto. I convocati sono bloccati.',
      no_callups: 'Salva almeno un convocato disponibile prima di aprire la votazione.',
      invalid_player: 'Il roster è cambiato. Controlla i convocati e riprova.',
      mvp_closed: 'La partita non è votabile in questo momento.',
    }[cause.message] || 'Impossibile salvare MVP Light. Riprova.'
  } finally {
    busy.value = false
  }
  if (error.value) void refresh()
}

function checkVisible() {
  if (document.hidden || state.value?.final) {
    closeStream()
    return
  }
  if (!stream && typeof EventSource === 'function') {
    stream = new EventSource(`/api/v1/club-admin/${encodeURIComponent(props.slug)}/mvp/stream`)
    stream.onopen = () => { connected = true }
    stream.onerror = () => { connected = false }
    stream.onmessage = () => { void refresh() }
  }
  void refresh()
}

function closeStream() {
  stream?.close()
  stream = undefined
  connected = false
}

onMounted(() => {
  heading.value?.focus()
  checkVisible()
  document.addEventListener('visibilitychange', checkVisible)
  timer = window.setInterval(() => {
    if (!document.hidden && !state.value?.final && (!connected || pendingRefresh)) void refresh()
  }, 4000)
})

onUnmounted(() => {
  controller.abort()
  closeStream()
  window.clearInterval(timer)
  document.removeEventListener('visibilitychange', checkVisible)
})
</script>

<template>
  <section class="card content-editor mvp-admin" aria-labelledby="mvp-admin-title" :aria-busy="busy || (!state && loading)">
    <header class="mvp-admin-header">
      <div>
        <h2 id="mvp-admin-title" ref="heading" tabindex="-1">Votazione MVP Light</h2>
        <p>{{ clubName }} – {{ match.opponent }}<span v-if="state"> · {{ state.team.name }}</span></p>
      </div>
      <button type="button" :disabled="busy" @click="$emit('close')">Torna alla partita</button>
    </header>
    <p v-if="error" class="mvp-admin-error" role="alert">{{ error }} <button type="button" :disabled="loading || busy" @click="error = ''; refresh()">Aggiorna</button></p>
    <p v-if="notice" role="status">{{ notice }}</p>
    <p v-if="!state && loading" role="status">Caricamento…</p>
    <template v-if="state">
      <dl class="mvp-admin-summary">
        <div><dt>Funzionalità</dt><dd>{{ state.enabled ? 'Attiva' : 'Non attiva' }}</dd></div>
        <div><dt>Votazione</dt><dd>{{ status }}</dd></div>
        <div><dt>Convocati</dt><dd>{{ state.callups.length }}</dd></div>
        <div><dt>Voti ricevuti</dt><dd>{{ state.totalVotes }}</dd></div>
        <div><dt>Giocatori votati</dt><dd>{{ state.playersVoted }}</dd></div>
      </dl>
      <div class="mvp-admin-actions">
        <button type="button" :aria-expanded="showCallups" @click="showCallups = !showCallups">{{ locked ? 'Mostra convocati' : 'Configura convocati' }}</button>
        <button type="button" :aria-expanded="showRanking" @click="showRanking = !showRanking">{{ state.final ? 'Classifica finale' : 'Classifica live' }}</button>
        <button v-if="!state.final" type="button" class="mvp-admin-primary" :disabled="busy || loading || (!state.enabled && (!state.canOpen || dirty))"
          @click="update({ enabled: !state.enabled }, state.enabled ? 'Votazione disattivata.' : 'Votazione attivata.')">
          {{ state.enabled ? 'Disattiva votazione' : 'Attiva votazione' }}
        </button>
      </div>
      <p v-if="!state.final && state.match.status !== 'live'" class="mvp-admin-hint">Puoi salvare i convocati ora e attivare la votazione dopo aver avviato la diretta.</p>
      <p v-else-if="!state.final && !state.enabled && !eligible.length" class="mvp-admin-hint">Salva almeno un convocato per attivare la votazione.</p>
      <div v-if="state.open || state.qrTarget" class="mvp-admin-actions">
        <a v-if="state.open" :href="publicPath" target="_blank" rel="noopener">Apri la pagina di voto</a>
        <button type="button" :disabled="busy || loading || (!state.qrTarget && !state.open)" @click="update({ qrTarget: !state.qrTarget }, state.qrTarget ? 'Priorità QR rimossa.' : 'Partita associata al QR ClubHub.')">
          {{ state.qrTarget ? 'Rimuovi priorità QR' : 'Associa al QR ClubHub' }}
        </button>
        <p class="mvp-admin-hint">Il QR apre l’unica votazione disponibile. Con più partite votabili, apre quella associata qui.</p>
      </div>

      <section v-if="showCallups" class="mvp-admin-section" aria-labelledby="mvp-callups-title">
        <h3 id="mvp-callups-title">MVP Light — Giocatori convocati</h3>
        <template v-if="locked">
          <p v-if="state.totalVotes" class="mvp-admin-hint">I convocati sono bloccati dal primo voto.</p>
          <ul v-if="state.callups.length" class="mvp-admin-list">
            <li v-for="player in state.callups" :key="player.id">
              <strong>#{{ player.jerseyNumber }} {{ playerName(player) }}</strong><small>{{ player.role }}</small>
              <small v-if="!player.eligible">Rimosso dal roster</small>
            </li>
          </ul>
          <p v-else>Nessun convocato salvato.</p>
        </template>
        <template v-else>
          <p>{{ selected.length }} selezionati su {{ state.roster.length }}</p>
          <p v-if="unavailable.length" class="mvp-admin-hint">{{ unavailable.length }} convocati salvati sono stati rimossi dal roster.</p>
          <fieldset v-if="state.roster.length" class="mvp-admin-list" :disabled="busy" aria-label="Convocati della divisione">
            <label v-for="player in state.roster" :key="player.id">
              <input v-model="selected" type="checkbox" :value="player.id" />
              <span><strong>#{{ player.jerseyNumber }} {{ playerName(player) }}</strong><small>{{ player.role }}</small></span>
            </label>
          </fieldset>
          <p v-else class="mvp-admin-hint">Il roster della divisione è vuoto.</p>
          <div class="mvp-admin-actions">
            <button type="button" :disabled="busy || selected.length === state.roster.length" @click="selected = state.roster.map(player => player.id)">Seleziona tutti</button>
            <button type="button" :disabled="busy || !selected.length" @click="selected = []">Deseleziona tutti</button>
            <button type="button" class="mvp-admin-primary" :disabled="busy || loading || (!dirty && !unavailable.length)" @click="update({ playerIds: selected }, 'Convocati salvati.')">Salva convocati</button>
          </div>
        </template>
      </section>

      <section v-if="showRanking" class="mvp-admin-section" aria-labelledby="mvp-ranking-title">
        <h3 id="mvp-ranking-title">{{ state.final ? 'Classifica MVP finale' : 'Classifica MVP live' }}</h3>
        <p v-if="leaders.length"><strong>{{ state.final ? 'MVP' : 'Leader provvisorio' }}{{ leaders.length > 1 ? ' ex aequo' : '' }}</strong> · {{ leaders.map(playerName).join(', ') }}</p>
        <p v-else class="mvp-admin-hint">Nessun voto ricevuto.</p>
        <ol v-if="state.callups.length" class="mvp-ranking">
          <li v-for="player in state.callups" :key="player.id" :class="{ 'mvp-leader': player.leader }">
            <div><strong>#{{ player.jerseyNumber }} {{ playerName(player) }}</strong><span>{{ player.votes }} {{ player.votes === 1 ? 'voto' : 'voti' }} · {{ percent.format(player.percentage) }}%</span></div>
            <progress :value="player.percentage" max="100" :aria-label="`Percentuale per ${playerName(player)}`"></progress>
          </li>
        </ol>
      </section>
    </template>
  </section>
</template>

<style scoped>
.mvp-admin { min-width: 0; color: #1e293b; overflow-wrap: anywhere; }
.mvp-admin-header, .mvp-admin-actions { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; }
.mvp-admin-header { justify-content: space-between; }
.mvp-admin h2, .mvp-admin h3 { margin: 0; font-weight: 750; }
.mvp-admin h2 { font-size: 1.4rem; }
.mvp-admin h3 { font-size: 1.1rem; }
.mvp-admin p { margin: 6px 0; }
.mvp-admin-summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(96px, 1fr)); gap: 8px; margin: 16px 0; }
.mvp-admin-summary div { border-radius: 8px; padding: 8px; background: #eef2f7; }
.mvp-admin-summary dt, .mvp-admin-hint { color: #475569; font-size: .86rem; }
.mvp-admin-summary dd { margin: 2px 0 0; font-weight: 750; }
.mvp-admin button { min-height: 40px; padding: 6px 10px; border: 1px solid #bcc8d9; border-radius: 8px; background: #fff; color: #1e293b; font-weight: 650; }
.mvp-admin button.mvp-admin-primary { border-color: #243c70; background: #243c70; color: #fff; }
.mvp-admin button:disabled { opacity: .57; }
.mvp-admin a { color: #243c70; text-decoration: underline; font-weight: 650; }
.mvp-admin-actions + .mvp-admin-actions { margin-top: 12px; }
.mvp-admin-section { border-top: 1px solid #d9e1ed; margin-top: 18px; padding-top: 14px; }
.mvp-admin-error { color: #9f1239; }
.mvp-admin-list { display: grid; min-inline-size: 0; grid-template-columns: repeat(auto-fit, minmax(min(100%, 175px), 1fr)); gap: 6px; border: 0; padding: 0; margin: 10px 0; }
.mvp-admin-list label, .mvp-admin-list li { display: flex; flex-wrap: wrap; align-items: center; gap: 6px 10px; min-height: 44px; padding: 6px 8px; border: 1px solid #d9e1ed; border-radius: 8px; }
.mvp-admin-list label span { flex: 1; min-width: 0; }
.mvp-admin-list input { width: 18px; height: 18px; accent-color: #243c70; }
.mvp-admin-list small { display: block; color: #475569; }
.mvp-ranking { display: grid; gap: 6px; margin: 12px 0 0; padding: 0; list-style: none; }
.mvp-ranking li { border: 1px solid #d9e1ed; border-radius: 8px; padding: 8px 10px; }
.mvp-ranking li.mvp-leader { border-color: #6783b7; background: #eef4ff; }
.mvp-ranking li div { display: flex; justify-content: space-between; flex-wrap: wrap; gap: 3px 12px; }
.mvp-ranking li span { font-variant-numeric: tabular-nums; }
.mvp-ranking progress { display: block; width: 100%; height: 10px; margin-top: 7px; accent-color: #345695; }
</style>
