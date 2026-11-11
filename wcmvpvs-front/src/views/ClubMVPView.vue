<script setup>
import { computed, onMounted, onUnmounted, ref, shallowRef } from 'vue'
import { clubHomePath } from '../clubDomain'
import { clubCurrentSetLabel, clubMatchSets, clubScoreLabel, clubSetLabel, connectClubMatches } from '../clubHub'
import { clubTypography } from '../clubTypography'

const props = defineProps({ slug: { type: String, required: true }, matchId: String, customDomain: Boolean })
const homePath = computed(() => clubHomePath(props.slug, props.customDomain))
const api = `/api/v1/clubs/${encodeURIComponent(props.slug)}`
const data = shallowRef(null)
const selectedId = ref(null)
const confirmed = ref(false)
const sending = ref(false)
const connected = ref(false)
const error = ref('')
const loading = ref(true)
const redirecting = ref(false)
const controller = new AbortController()
let disconnect
let fallback
let refreshing = false
let refreshRequested = false
let retryRefresh = false

const match = computed(() => data.value?.match)
const selected = computed(() => data.value?.players.find(player => player.id === selectedId.value))
const partials = computed(() => match.value ? clubMatchSets(match.value).map(set => clubSetLabel(set, match.value.home)) : [])
const currentSet = computed(() => match.value && clubCurrentSetLabel(match.value))
const sides = computed(() => {
  const { club, team, match } = data.value
  const clubSide = { name: club.name, logo: team.logoUrl || club.logoUrl }
  const opponent = { name: match.opponent, logo: '' }
  return match.home ? [clubSide, opponent] : [opponent, clubSide]
})
const theme = computed(() => {
  const club = data.value?.club
  if (!club) return {}
  const { style } = clubTypography(club)
  return { '--primary': club.primaryColor, '--accent': club.secondaryColor, '--title-font': style['--club-font-title'], '--body-font': style['--club-font-body'] }
})
const playerName = player => [player.firstName, player.lastName].filter(Boolean).join(' ')
const matchAPI = id => `${api}/matches/${id}/mvp`

function goHome() {
  if (redirecting.value) return
  redirecting.value = true
  window.location.replace(homePath.value)
}

async function refresh() {
  if (redirecting.value || controller.signal.aborted) return
  if (refreshing) {
    refreshRequested = true
    return
  }
  refreshing = true
  if (!data.value) {
    loading.value = true
    error.value = ''
  }
  try {
    const id = match.value?.id || props.matchId
    const response = await fetch(id ? matchAPI(id) : `${api}/mvp`, {
      cache: 'no-store', credentials: 'same-origin', signal: controller.signal,
    })
    if (response.status === 404 || response.status === 409) return goHome()
    if (!response.ok) throw new Error('mvp_unavailable')
    const next = await response.json()
    if (redirecting.value) return
    retryRefresh = false
    if (match.value?.revision > next.match.revision) next.match = match.value
    next.voted ||= data.value?.voted
    data.value = next
    if (selectedId.value !== null && !selected.value) selectedId.value = null
    if (!disconnect) disconnect = connectClubMatches(props.slug, onScore, value => { connected.value = value })
  } catch (cause) {
    if (cause.name !== 'AbortError') {
      retryRefresh = true
      if (!data.value) error.value = 'Impossibile caricare la votazione. Riprova.'
    }
  } finally {
    loading.value = false
    refreshing = false
    if (refreshRequested) {
      refreshRequested = false
      void refresh()
    }
  }
}

function onScore(update) {
  const current = match.value
  if (!current || redirecting.value) return
  const next = update.match || update.matches?.find(item => item.id === current.id)
  if (next?.id === current.id && next.revision >= current.revision) {
    if (next.status !== 'live') return goHome()
    data.value = { ...data.value, match: next }
  }
  // A new calendar snapshot also follows changes to call-ups or voting status.
  if (update.matches) void refresh()
}

function checkOnReturn() {
  if (!document.hidden) void refresh()
}

async function vote() {
  if (!selected.value || sending.value || data.value.voted || redirecting.value) return
  sending.value = true
  error.value = ''
  try {
    const response = await fetch(`${matchAPI(match.value.id)}/vote`, {
      method: 'POST', credentials: 'same-origin', signal: controller.signal,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ playerId: selected.value.id }),
    })
    const result = await response.json().catch(() => ({}))
    if (redirecting.value) return
    if (response.status === 404 || result.error === 'mvp_closed') return goHome()
    if (result.voted && (response.ok || result.error === 'already_voted')) {
      confirmed.value = response.ok
      data.value = { ...data.value, voted: result.voted }
      return
    }
    if (result.error === 'invalid_player' || result.error === 'voter_cookie_required') void refresh()
    error.value = {
      invalid_player: 'I convocati sono cambiati. Controlla la lista e scegli un giocatore.',
      voter_cookie_required: 'Per votare occorre consentire i cookie per questo sito. Poi riprova.',
      rate_limited: 'Troppi tentativi ravvicinati. Attendi qualche secondo e riprova.',
    }[result.error] || 'Invio del voto non riuscito. Riprova.'
  } catch (cause) {
    if (cause.name !== 'AbortError') error.value = 'Invio del voto non riuscito. Controlla la connessione e riprova.'
  } finally {
    sending.value = false
  }
}

onMounted(() => {
  void refresh()
  document.addEventListener('visibilitychange', checkOnReturn)
  fallback = window.setInterval(() => {
    if (data.value && (!connected.value || retryRefresh)) checkOnReturn()
  }, 10000)
})

onUnmounted(() => {
  controller.abort()
  disconnect?.()
  window.clearInterval(fallback)
  document.removeEventListener('visibilitychange', checkOnReturn)
})
</script>

<template>
  <main class="mvp-page" :style="theme">
    <div v-if="redirecting || !data" class="mvp-state">
      <p v-if="redirecting" role="status">Votazione chiusa. <a :href="homePath">Torna alla home del club</a></p>
      <p v-else-if="loading" role="status">Caricamento votazione…</p>
      <template v-else>
        <p role="alert">{{ error }}</p>
        <button type="button" class="mvp-confirm" @click="refresh">Riprova</button>
      </template>
    </div>
    <template v-else>
      <header class="mvp-score">
        <div class="mvp-score-inner">
          <div class="mvp-division"><span>{{ data.team.name }}</span><span class="mvp-live">LIVE</span></div>
          <div class="mvp-sides">
            <div v-for="(side, i) in sides" :key="i" class="mvp-team" :class="{ 'is-away': i === 1 }">
              <img v-if="side.logo" :src="side.logo" alt="" width="42" height="42" />
              <strong>{{ side.name }}</strong>
            </div>
            <div class="mvp-score-value" :aria-label="`Risultato ${clubScoreLabel(match)}`">{{ clubScoreLabel(match) }}</div>
          </div>
          <div class="mvp-partials">
            <p v-if="currentSet">{{ currentSet }}</p>
            <p v-if="partials.length">Parziali · {{ partials.join(' · ') }}</p>
          </div>
        </div>
      </header>

      <section v-if="data.voted" class="mvp-state mvp-receipt" role="status">
        <span class="mvp-check" aria-hidden="true">✓</span>
        <h1>{{ confirmed ? 'Voto registrato!' : 'Hai già votato' }}</h1>
        <p>Hai votato <strong>{{ playerName(data.voted) }}</strong> come MVP della partita.</p>
      </section>
      <form v-else class="mvp-ballot" :aria-busy="sending" @submit.prevent="vote">
        <h1>Vota il tuo MVP</h1>
        <p class="mvp-intro">Chi sta facendo la differenza? Scegli il tuo giocatore.</p>
        <fieldset class="mvp-grid" aria-label="Giocatori convocati" :disabled="sending">
          <label v-for="player in data.players" :key="player.id" class="mvp-card" :class="{ 'is-selected': selectedId === player.id }">
            <input v-model="selectedId" type="radio" name="mvp-player" :value="player.id" @change="error = ''" />
            <span class="mvp-card-top">
              <span class="mvp-jersey"><small>#</small>{{ player.jerseyNumber }}</span>
              <span class="mvp-card-check" aria-hidden="true">{{ selectedId === player.id ? '✓' : '' }}</span>
            </span>
            <strong>{{ playerName(player) }}</strong>
            <span v-if="player.role" class="mvp-role">{{ player.role }}</span>
          </label>
        </fieldset>
        <div class="mvp-actions">
          <p v-if="error" class="mvp-error" role="alert">{{ error }}</p>
          <button type="submit" class="mvp-confirm" :disabled="!selected || sending">
            {{ sending ? 'Invio in corso…' : 'Conferma voto' }}
          </button>
        </div>
      </form>
    </template>
  </main>
</template>

<style scoped>
.mvp-page { --ink: #10182b; --dark: color-mix(in srgb, #0c1223 78%, var(--primary, #172554)); min-height: 100dvh; background: #f1f4f8; color: var(--ink); font-family: var(--body-font, Inter, sans-serif); }
.mvp-score { display: grid; align-items: center; min-height: clamp(162px, 31svh, 270px); padding: 18px max(14px, env(safe-area-inset-left)) 22px max(14px, env(safe-area-inset-right)); background: radial-gradient(ellipse at top, color-mix(in srgb, var(--dark) 72%, #52689c), var(--dark) 72%); color: #fff; }
.mvp-score-inner { width: min(100%, 680px); margin-inline: auto; }
.mvp-division { display: flex; justify-content: center; align-items: center; gap: 10px; font-size: .875rem; font-weight: 700; text-align: center; overflow-wrap: anywhere; }
.mvp-live { flex: none; border: 1px solid #ffadb7; border-radius: 999px; padding: 3px 7px; background: #562030; color: #ffe8eb; font-size: .68rem; letter-spacing: .06em; }
.mvp-sides { display: grid; grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr); align-items: center; gap: clamp(6px, 3vw, 22px); margin-top: 14px; text-align: center; }
.mvp-team { grid-area: 1 / 1; display: grid; justify-items: center; gap: 6px; font-size: clamp(.8rem, 3.25vw, 1.05rem); line-height: 1.2; overflow-wrap: anywhere; }
.mvp-team.is-away { grid-column: 3; }
.mvp-team img { max-width: 100%; object-fit: contain; }
.mvp-score-value { grid-area: 1 / 2; border-block: 1px solid #ffffff33; padding: 4px; font: 800 clamp(1.7rem, 7vw, 3.25rem)/1.1 var(--title-font, Inter, sans-serif); font-variant-numeric: tabular-nums; white-space: nowrap; }
.mvp-partials { margin-top: 10px; color: #e1e9fa; text-align: center; font-size: .82rem; overflow-wrap: anywhere; }
.mvp-partials p { margin: 2px 0; }
.mvp-ballot { width: min(100%, 920px); margin-inline: auto; padding: 22px clamp(12px, 3vw, 26px) 0; }
.mvp-page h1 { margin: 0; font: 850 clamp(1.7rem, 5vw, 2.5rem)/1.1 var(--title-font, Inter, sans-serif); letter-spacing: -.035em; }
.mvp-intro { margin: 7px 0 18px; color: #46526a; font-size: .94rem; }
.mvp-grid { min-inline-size: 0; display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 136px), 1fr)); gap: 10px; margin: 0; padding: 0; border: 0; }
.mvp-card { position: relative; display: flex; flex-direction: column; justify-content: start; gap: 4px; min-width: 0; min-height: 110px; padding: 10px; border: 2px solid #d7dfeb; border-radius: 16px; background: #fff; cursor: pointer; overflow-wrap: anywhere; transition: border-color 150ms, background-color 150ms, transform 150ms; }
.mvp-card.is-selected { border-color: var(--dark); background: color-mix(in srgb, #fff 86%, var(--accent, #f59e0b)); }
.mvp-card:active { transform: scale(.98); }
.mvp-card:focus-within, .mvp-confirm:focus-visible { outline: 3px solid #245cd1; outline-offset: 2px; }
.mvp-card input { position: absolute; width: 1px; height: 1px; opacity: 0; }
.mvp-card-top { display: flex; justify-content: space-between; align-items: start; gap: 4px; }
.mvp-jersey { font: 900 clamp(1.85rem, 7vw, 2.7rem)/1 var(--title-font, Inter, sans-serif); font-variant-numeric: tabular-nums; letter-spacing: -.06em; }
.mvp-jersey small { font-size: .46em; margin-right: 2px; }
.mvp-card-check { display: grid; place-items: center; width: 23px; height: 23px; flex: none; border: 1px solid #8491a7; border-radius: 50%; }
.mvp-card.is-selected .mvp-card-check { background: var(--dark); color: #fff; font-weight: 800; }
.mvp-card strong { line-height: 1.2; }
.mvp-role { color: #46526a; font-size: .8rem; line-height: 1.2; }
.mvp-actions { position: sticky; bottom: 0; display: grid; justify-items: center; gap: 6px; padding: 12px 0 max(12px, env(safe-area-inset-bottom)); background: #f1f4f8; }
.mvp-error { margin: 0; color: #8c1832; font-size: .875rem; text-align: center; }
.mvp-confirm { width: min(100%, 420px); min-height: 48px; padding: 8px 16px; border: 0; border-radius: 12px; background: var(--dark); color: #fff; font: inherit; font-weight: 800; cursor: pointer; }
.mvp-confirm:disabled { background: #d7dfeb; color: #43516a; cursor: default; }
.mvp-state { display: grid; justify-items: center; gap: 12px; padding: 38px 18px; text-align: center; }
.mvp-state p { margin: 0; overflow-wrap: anywhere; }
.mvp-check { display: grid; place-items: center; width: 52px; height: 52px; border-radius: 50%; background: #d7f2e5; color: #155735; font-size: 1.7rem; animation: mvp-appear 200ms ease-out; }
@keyframes mvp-appear { from { transform: scale(.88); opacity: .4; } }
@media (min-width: 680px) { .mvp-grid { grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); } }
@media (prefers-reduced-motion: reduce) { .mvp-card, .mvp-check { transition: none; animation: none; } }
</style>
