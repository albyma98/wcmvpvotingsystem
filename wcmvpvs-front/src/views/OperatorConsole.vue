<script setup>
// Console operatore campo (/op/:token) — il volontario del Campo 2.
// Zero account: magic link + PIN a 6 cifre consegnati via WhatsApp dall'admin.
// Vede e tocca SOLO le partite del suo campo (enforcement lato server).
import { computed, onMounted, ref } from 'vue'
import { useTournamentStream } from '@/composables/useTournamentStream'

const props = defineProps({ token: { type: String, required: true } })

const API = path => `/api/v1/op/${props.token}${path}`
const j = (method, path, body) =>
  fetch(API(path), {
    method,
    credentials: 'include',
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined
  })

const authed = ref(false)
const checked = ref(false)
const pin = ref('')
const pinError = ref('')
const court = ref('')
const tournament = ref('')
const slug = ref('')
const matches = ref([])
const notice = ref('')

function flash (m) { notice.value = m; setTimeout(() => { if (notice.value === m) notice.value = '' }, 2500) }

async function loadState () {
  const r = await j('GET', '/state')
  if (!r.ok) { authed.value = false; checked.value = true; return }
  const data = await r.json()
  court.value = data.court
  tournament.value = data.tournament
  slug.value = data.slug ?? ''
  matches.value = data.matches
  authed.value = true
  checked.value = true
}

async function doLogin () {
  pinError.value = ''
  const r = await j('POST', '/login', { pin: pin.value.trim() })
  if (!r.ok) { pinError.value = 'PIN errato.'; return }
  pin.value = ''
  await loadState()
}

async function score (matchId, action) {
  const r = await j('POST', `/matches/${matchId}/score`, { action })
  if (r.ok) {
    matches.value = (await r.json()).matches
  } else {
    const err = (await r.json().catch(() => ({}))).error
    if (err === 'set_tied') flash('Il set non può chiudersi in parità.')
    else if (err === 'not_your_court') flash('Questa partita non è sul tuo campo.')
    else if (err === 'no_closed_set') flash('Non ci sono set chiusi da annullare.')
    else if (err === 'current_set_started') flash('Annulla prima tutti i punti del set in corso.')
  }
}

async function undoLastSet (match) {
  if (!window.confirm(`Annullare l'ultimo set chiuso (${match.sets.at(-1)})? Il punteggio tornerà modificabile.`)) return
  if (!window.confirm('Seconda conferma: vuoi davvero annullare l’ultimo set chiuso?')) return
  await score(match.id, 'undo_last_set')
}

const live = computed(() => matches.value.filter(m => m.status === 'live'))
const scheduled = computed(() => matches.value.filter(m => m.status === 'scheduled'))
const finished = computed(() => matches.value.filter(m => m.status === 'finished'))

// Link del tabellone da proiettare per il pubblico (view pubblica full-screen).
const projectionUrl = computed(() =>
  slug.value ? `${window.location.origin}/proietta/${slug.value}/${encodeURIComponent(court.value)}` : '')
function copyProjection () {
  if (!projectionUrl.value || !navigator.clipboard) return
  navigator.clipboard.writeText(projectionUrl.value).then(() => flash('Link tabellone copiato.'), () => {})
}

// Aggiornamenti live via SSE (push): lo slug arriva dopo il login, il composable
// si aggancia appena disponibile. Le partite di un altro campo o create
// dall'admin compaiono senza refresh manuale.
onMounted(loadState)
useTournamentStream(slug, () => { if (authed.value) loadState() })
</script>

<template>
  <div class="op-page">
    <!-- PIN -->
    <div v-if="checked && !authed" class="op-pin">
      <h1>Console Campo</h1>
      <p>Inserisci il PIN che ti ha dato l'organizzatore.</p>
      <input v-model="pin" inputmode="numeric" maxlength="6" placeholder="······" @keyup.enter="doLogin" />
      <button @click="doLogin">Entra</button>
      <p v-if="pinError" class="err">{{ pinError }}</p>
    </div>

    <template v-else-if="authed">
      <header class="op-head">
        <h1>{{ court }}</h1>
        <p>{{ tournament }}</p>
      </header>
      <p v-if="notice" class="op-notice">{{ notice }}</p>

      <!-- Tabellone da proiettare per il pubblico -->
      <div v-if="projectionUrl" class="proj-link">
        <div class="pl-text">
          <b>📺 Tabellone pubblico</b>
          <small>Aprilo sul monitor/proiettore rivolto agli spettatori: mostra il punteggio live e la classifica.</small>
        </div>
        <div class="pl-actions">
          <a class="pl-open" :href="projectionUrl" target="_blank" rel="noopener">Apri</a>
          <button class="pl-copy" @click="copyProjection">Copia link</button>
        </div>
      </div>

      <article v-for="m in live" :key="m.id" class="score-card">
        <div class="sc-head"><span class="dot"></span>{{ m.setLabel }}</div>
        <div class="sc-grid">
          <div class="sc-team">
            <div class="name">{{ m.teamAName }}</div>
            <div class="cur">{{ m.curA }}</div>
            <button class="big" @click="score(m.id, 'point_a')">+1</button>
            <button class="undo" @click="score(m.id, 'undo_a')">annulla</button>
          </div>
          <div class="sc-mid">
            <div class="sets">{{ m.scoreA }} : {{ m.scoreB }}</div>
            <div class="setlist">{{ m.sets.join(' | ') || '—' }}</div>
            <button class="close-set" @click="score(m.id, 'close_set')">Chiudi set</button>
            <button v-if="m.sets.length" class="undo-set" @click="undoLastSet(m)">Annulla ultimo set</button>
            <button class="finish" @click="score(m.id, 'finish')">Fine partita</button>
          </div>
          <div class="sc-team">
            <div class="name">{{ m.teamBName }}</div>
            <div class="cur">{{ m.curB }}</div>
            <button class="big" @click="score(m.id, 'point_b')">+1</button>
            <button class="undo" @click="score(m.id, 'undo_b')">annulla</button>
          </div>
        </div>
      </article>
      <p v-if="!live.length" class="hint">Nessuna partita live sul tuo campo.</p>

      <h3 v-if="scheduled.length">Prossime sul {{ court }}</h3>
      <div v-for="m in scheduled" :key="m.id" class="row-match">
        <span>{{ m.time || '—' }} · <b>{{ m.teamAName }}</b> vs <b>{{ m.teamBName }}</b><small v-if="m.stage"> · {{ m.stage }}</small></span>
        <button class="start" @click="score(m.id, 'start')">▶ Avvia</button>
      </div>

      <h3 v-if="finished.length">Concluse</h3>
      <div v-for="m in finished" :key="m.id" class="row-match done">
        <span>{{ m.teamAName }} <b>{{ m.scoreA }}:{{ m.scoreB }}</b> {{ m.teamBName }}</span>
        <button class="ghost" @click="score(m.id, 'reopen')">Riapri</button>
      </div>
    </template>

    <p v-else class="hint center">Caricamento…</p>
  </div>
</template>

<style scoped>
.op-page { min-height: 100dvh; background: #0a0a0e; color: #f1f5f9; padding: 16px; max-width: 560px; margin: 0 auto; display: flex; flex-direction: column; gap: 10px; }
.op-pin { max-width: 300px; margin: 16vh auto 0; display: flex; flex-direction: column; gap: 10px; text-align: center; }
.op-pin h1 { font-size: 22px; margin: 0; }
.op-pin p { color: #94a3b8; font-size: 13.5px; margin: 0; }
.op-pin input { background: #15151b; border: 1px solid rgba(255,255,255,.14); border-radius: 12px; padding: 14px; color: #fff; font-size: 26px; text-align: center; letter-spacing: 8px; }
.op-pin button { background: #f2b928; color: #111; border: none; border-radius: 10px; padding: 13px; font-weight: 800; font-size: 15px; cursor: pointer; }
.err { color: #f87171; }
.op-head h1 { font-size: 24px; margin: 0; color: #f2b928; }
.op-head p { margin: 2px 0 0; color: #94a3b8; font-size: 13px; }
.op-notice { background: rgba(242,185,40,.12); border: 1px solid rgba(242,185,40,.35); color: #fbd34d; border-radius: 8px; padding: 8px 12px; font-size: 13px; }
.proj-link { display: flex; align-items: center; justify-content: space-between; gap: 10px; background: #15151b; border: 1px solid rgba(255,255,255,.12); border-radius: 12px; padding: 11px 12px; }
.proj-link .pl-text { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.proj-link .pl-text b { font-size: 14px; }
.proj-link .pl-text small { color: #94a3b8; font-size: 12px; line-height: 1.35; }
.proj-link .pl-actions { display: flex; gap: 6px; flex: none; }
.proj-link .pl-open { background: #f2b928; color: #111; text-decoration: none; border-radius: 8px; padding: 9px 16px; font-weight: 800; font-size: 13px; }
.proj-link .pl-copy { background: transparent; border: 1px solid rgba(255,255,255,.22); color: #cbd5e1; border-radius: 8px; padding: 9px 12px; font-size: 13px; }
.hint { color: #94a3b8; font-size: 13.5px; }
.center { text-align: center; padding-top: 18vh; }
h3 { margin: 12px 0 2px; font-size: 12px; color: #94a3b8; text-transform: uppercase; letter-spacing: 1px; }
button { cursor: pointer; }
.row-match { display: flex; align-items: center; justify-content: space-between; gap: 10px; background: #15151b; border: 1px solid rgba(255,255,255,.09); border-radius: 10px; padding: 11px 12px; }
.row-match.done { opacity: .7; }
.row-match small { color: #f2b928; }
.start { background: #16a34a; color: #fff; border: none; border-radius: 8px; padding: 9px 18px; font-weight: 800; }
.ghost { background: transparent; border: 1px solid rgba(255,255,255,.2); color: #cbd5e1; border-radius: 7px; padding: 6px 12px; }
.score-card { background: linear-gradient(180deg, rgba(139,32,38,.4), #15151b 60%); border: 1px solid rgba(255,255,255,.1); border-radius: 14px; padding: 12px; }
.sc-head { display: flex; align-items: center; gap: 8px; font-size: 12px; font-weight: 800; letter-spacing: 1px; color: #fca5a5; margin-bottom: 8px; }
.sc-head .dot { width: 8px; height: 8px; border-radius: 50%; background: #e5484d; }
.sc-grid { display: grid; grid-template-columns: 1fr auto 1fr; gap: 10px; align-items: start; }
.sc-team { display: flex; flex-direction: column; align-items: center; gap: 8px; }
.sc-team .name { font-weight: 900; text-transform: uppercase; font-size: 13px; text-align: center; }
.sc-team .cur { font-size: 48px; font-weight: 900; line-height: 1; font-variant-numeric: tabular-nums; }
.sc-team .big { width: 100%; max-width: 140px; font-size: 28px; font-weight: 900; padding: 18px 0; border: none; border-radius: 14px; background: #f2b928; color: #111; }
.sc-team .undo { background: transparent; border: 1px solid rgba(255,255,255,.25); color: #cbd5e1; border-radius: 8px; padding: 8px 16px; font-size: 13px; }
.sc-mid { display: flex; flex-direction: column; align-items: center; gap: 6px; padding-top: 4px; }
.sc-mid .sets { font-size: 22px; font-weight: 900; color: #f2b928; }
.sc-mid .setlist { font-size: 12px; color: #94a3b8; }
.close-set { background: #1d4ed8; color: #fff; border: none; border-radius: 8px; padding: 10px 14px; font-weight: 800; font-size: 13px; }
.undo-set { background: transparent; border: 1px solid rgba(251,191,36,.55); color: #fbbf24; border-radius: 8px; padding: 8px 11px; font-size: 12px; font-weight: 700; }
.finish { background: transparent; border: 1px solid rgba(248,113,113,.5); color: #f87171; border-radius: 8px; padding: 7px 14px; font-size: 12px; }
</style>
