<script setup>
// Sezioni fan del torneo — stesse variabili visive della home (dark + gold).
// Calendario, Classifiche e Tabellone sono dati veri dagli endpoint pubblici;
// le altre sezioni restano placeholder finché non hanno contenuto (P2/P3).
import { computed, onMounted, ref, watch } from 'vue'
import { useTournamentStream } from '@/composables/useTournamentStream'
import TournamentGallery from '@/components/tournament/TournamentGallery.vue'
import TournamentMvpVote from '@/components/tournament/TournamentMvpVote.vue'
import { track as posthogTrack, EVENTS as PH_EVENTS } from '@/lib/track'

const props = defineProps({
  slug: { type: String, required: true },
  section: { type: String, required: true }
})
const emit = defineEmits(['navigate'])

const tournamentLayout = ref('')

async function loadTournamentContext () {
  try {
    const response = await fetch(`/api/v1/tournaments/${props.slug}/home`)
    if (!response.ok) throw new Error(response.status)
    const data = await response.json()
    tournamentLayout.value = data.tournament?.layout || 'classic'
  } catch {
    tournamentLayout.value = 'unknown'
  }
}

// section_viewed: quale sezione apre il tifoso. immediate + watch così scatta
// sia al mount sia quando cambia la sezione senza remount del componente.
watch(
  [() => props.section, tournamentLayout],
  ([section, layout]) => {
    if (!layout) return
    posthogTrack(PH_EVENTS.TOURNAMENT_SECTION_VIEWED, {
      tournament_slug: props.slug,
      surface: 'tournament',
      section,
      layout,
    })
  },
  { immediate: true },
)

const titles = {
  calendar: 'Calendario', standings: 'Classifiche', bracket: 'Tabellone',
  mvp: 'Vota MVP', prizes: 'Premi', gallery: 'Gallery',
  rules: 'Regolamento', event: 'Info Evento', shop: 'Shop', example: 'Esempio'
}

const matches = ref([])
const groups = ref([])
const allowDraws = ref(true)   // classifica: mostrare la colonna pareggi "N"?
const qualifiersPerGroup = ref(2)
const standingsLegendText = ref('Primi 2 di ogni girone alla fase finale · Ordinamento: punti, quoziente set, quoziente punti')
const photos = ref([])
const tournamentStarted = ref(false)
const loading = ref(true)
const error = ref('')

// silent = refresh in background (tick SSE): aggiorna SOLO i dati senza toccare
// `loading`/`error`, così Vue fa il diff sulle righe :key-ate e ridisegna solo il
// punteggio cambiato — niente flash di ricaricamento dell'intera pagina.
async function load (silent = false) {
  if (!silent) { loading.value = true; error.value = '' }
  try {
    // no-store: le viste live (calendario/classifiche) si aggiornano via SSE ad
    // ogni punto; il fetch non deve tornare la copia cachata dal browser.
    if (props.section === 'standings') {
      const r = await fetch(`/api/v1/tournaments/${props.slug}/standings`, { cache: 'no-store' })
      if (!r.ok) throw new Error(r.status)
      const data = await r.json()
      groups.value = data.groups ?? []
      allowDraws.value = data.allowDraws !== false
      qualifiersPerGroup.value = Math.max(1, Number(data.qualifiersPerGroup) || 2)
      standingsLegendText.value = typeof data.legendText === 'string' ? data.legendText : standingsLegendText.value
    } else if (props.section === 'calendar' || props.section === 'bracket') {
      const r = await fetch(`/api/v1/tournaments/${props.slug}/matches`, { cache: 'no-store' })
      if (!r.ok) throw new Error(r.status)
      matches.value = (await r.json()).matches ?? []
    } else if (props.section === 'gallery') {
      const r = await fetch(`/api/v1/tournaments/${props.slug}/gallery`)
      if (!r.ok) throw new Error(r.status)
      const data = await r.json()
      photos.value = data.photos ?? []
      tournamentStarted.value = data.tournamentStarted !== false
    }
  } catch (e) {
    if (!silent) {
      error.value = 'Dati non disponibili al momento.'
      posthogTrack(PH_EVENTS.TOURNAMENT_SECTION_LOAD_FAILED, {
        tournament_slug: props.slug,
        surface: 'tournament',
        section: props.section,
        layout: tournamentLayout.value || 'unknown',
        reason: String(e?.message || 'request_failed'),
      })
    }
  } finally {
    if (!silent) loading.value = false
  }
}

// Calendario raggruppato per campo
const byCourt = computed(() => {
  const map = {}
  for (const m of matches.value) (map[m.court] ||= []).push(m)
  return Object.entries(map).sort(([a], [b]) => a.localeCompare(b))
})

// Tabellone: solo partite con stage, in ordine di fase
const stageOrder = ['OTTAVI', 'QUARTI', 'SEMIFINALE', 'FINALE 3° POSTO', 'FINALE']
const byStage = computed(() => {
  const map = {}
  for (const m of matches.value) if (m.stage) (map[m.stage.toUpperCase()] ||= []).push(m)
  return Object.entries(map).sort(([a], [b]) => {
    const ia = stageOrder.indexOf(a); const ib = stageOrder.indexOf(b)
    return (ia === -1 ? 99 : ia) - (ib === -1 ? 99 : ib)
  })
})

const hasData = computed(() =>
  ['calendar', 'standings', 'bracket', 'gallery'].includes(props.section))

// Le sezioni live si aggiornano da sole via SSE (push, niente polling).
onMounted(() => {
  loadTournamentContext()
  load()
})
// Refresh SSE in modalità silent: aggiorna i dati senza il flash di "Caricamento…".
useTournamentStream(props.slug, (update) => {
  if (!hasData.value) return
  // I punti live aggiornano calendario/tabellone, ma non modificano la
  // classifica finché la partita non viene conclusa (o riaperta).
  if (props.section === 'standings' && update?.standings === false) return
  load(true)
})

function goBack () {
  posthogTrack(PH_EVENTS.TOURNAMENT_SECTION_BACK, {
    tournament_slug: props.slug,
    surface: 'tournament',
    section: props.section,
    layout: tournamentLayout.value || 'unknown',
  })
  emit('navigate', `/t/${props.slug}`)
}
</script>

<template>
  <div class="section-page">
    <header class="section-head">
      <button class="back" @click="goBack" aria-label="Indietro">‹</button>
      <h1>{{ titles[props.section] || props.section }}</h1>
    </header>

    <div class="section-body">
      <!-- GALLERY: sempre montata (niente flash di "Caricamento" sui refresh live) -->
      <TournamentGallery
        v-if="section === 'gallery'"
        :slug="slug" :photos="photos" :started="tournamentStarted"
        :layout="tournamentLayout || 'unknown'" @uploaded="load"
      />

      <!-- VOTA MVP: componente autonomo (fetch/voto/live via device) -->
      <TournamentMvpVote
        v-else-if="section === 'mvp'"
        :slug="slug"
        :layout="tournamentLayout || 'unknown'"
      />

      <p v-else-if="hasData && loading" class="muted">Caricamento…</p>
      <p v-else-if="hasData && error" class="muted">{{ error }}</p>

      <!-- CALENDARIO -->
      <template v-else-if="section === 'calendar'">
        <p v-if="!matches.length" class="muted">Il calendario non è ancora pubblicato.</p>
        <section v-for="[court, list] in byCourt" :key="court" class="court-block">
          <h2>{{ court }}</h2>
          <article v-for="m in list" :key="m.id" class="match-row" :class="m.status">
            <div class="mr-time">
              <span v-if="m.status === 'live'" class="pill live"><span class="dot"></span>LIVE</span>
              <span v-else-if="m.status === 'finished'" class="pill done">FINITA</span>
              <span v-else class="pill">{{ m.time || '—' }}</span>
              <small v-if="m.stage" class="stage">{{ m.stage }}</small>
            </div>
            <div class="mr-teams">
              <span class="team-line" :class="{ winner: m.status === 'finished' && m.scoreA > m.scoreB }">
                <img v-if="m.teamALogo" :src="m.teamALogo" :alt="m.teamA" />{{ m.teamA }}
              </span>
              <span class="team-line" :class="{ winner: m.status === 'finished' && m.scoreB > m.scoreA }">
                <img v-if="m.teamBLogo" :src="m.teamBLogo" :alt="m.teamB" />{{ m.teamB }}
              </span>
            </div>
            <div class="mr-score" v-if="m.status !== 'scheduled'">
              <!-- LIVE: punti del set in corso (grandi) + set vinti e n° set -->
              <template v-if="m.status === 'live'">
                <b class="live-pt">{{ m.curA }}</b><b class="live-pt">{{ m.curB }}</b>
                <small>{{ m.setLabel }} · Set {{ m.scoreA }}-{{ m.scoreB }}</small>
              </template>
              <template v-else>
                <b>{{ m.scoreA }}</b><b>{{ m.scoreB }}</b>
                <small v-if="m.sets.length">{{ m.sets.join(' ') }}</small>
              </template>
            </div>
          </article>
        </section>
      </template>

      <!-- CLASSIFICHE -->
      <template v-else-if="section === 'standings'">
        <p v-if="!groups.length" class="muted">Le classifiche appariranno dopo le prime partite concluse.</p>
        <section v-for="g in groups" :key="g.group" class="group-block">
          <h2>{{ g.group ? `Girone ${g.group}` : 'Classifica' }}</h2>
          <table>
            <thead><tr><th></th><th class="tl">Squadra</th><th>G</th><th>V</th><th v-if="allowDraws">N</th><th>P</th><th>Set</th><th class="pt">Pt</th></tr></thead>
            <tbody>
              <tr v-for="(row, i) in g.rows" :key="row.teamId" :class="{ top: i < qualifiersPerGroup }">
                <td class="pos">{{ i + 1 }}</td>
                <td class="tl name">
                  <span class="standing-team"><img v-if="row.logoUrl" :src="row.logoUrl" :alt="row.team" />{{ row.team }}</span>
                </td>
                <td>{{ row.played }}</td>
                <td class="w">{{ row.wins }}</td>
                <td v-if="allowDraws">{{ row.draws }}</td>
                <td>{{ row.losses }}</td>
                <td>{{ row.setsWon }}:{{ row.setsLost }}</td>
                <td class="pt">{{ row.points }}</td>
              </tr>
            </tbody>
          </table>
        </section>
        <p v-if="groups.length && standingsLegendText" class="legend">{{ standingsLegendText }}</p>
      </template>

      <!-- TABELLONE -->
      <template v-else-if="section === 'bracket'">
        <p v-if="!byStage.length" class="muted">Il tabellone sarà pubblicato al termine dei gironi.</p>
        <section v-for="[stage, list] in byStage" :key="stage" class="stage-block">
          <h2>{{ stage }}</h2>
          <article v-for="m in list" :key="m.id" class="bracket-card" :class="m.status">
            <div class="bc-row" :class="{ winner: m.status === 'finished' && m.scoreA > m.scoreB }">
              <span>{{ m.teamA }}</span><b>{{ m.status === 'scheduled' ? '' : m.scoreA }}</b>
            </div>
            <div class="bc-row" :class="{ winner: m.status === 'finished' && m.scoreB > m.scoreA }">
              <span>{{ m.teamB }}</span><b>{{ m.status === 'scheduled' ? '' : m.scoreB }}</b>
            </div>
            <div class="bc-meta">
              <span v-if="m.status === 'live'" class="pill live"><span class="dot"></span>LIVE · {{ m.setLabel }}</span>
              <span v-else>{{ m.court }} · {{ m.time || '—' }}</span>
              <small v-if="m.sets.length">{{ m.sets.join(' ') }}</small>
            </div>
          </article>
        </section>
      </template>

      <!-- ALTRE SEZIONI (P2/P3) -->
      <p v-else class="muted">Sezione «{{ titles[props.section] || props.section }}» in arrivo.</p>
    </div>
  </div>
</template>

<style scoped>
.section-page { min-height: 100dvh; background: #0A0A0E; color: #fff; }
.section-head {
  position: sticky; top: 0; z-index: 5; display: flex; align-items: center; gap: 10px;
  padding: calc(env(safe-area-inset-top, 0px) + 12px) 16px 12px;
  background: rgba(10,10,14,.92); backdrop-filter: blur(8px);
  border-bottom: 1px solid rgba(255,255,255,.08);
}
.back {
  width: 34px; height: 34px; border-radius: 10px; border: 1px solid rgba(255,255,255,.15);
  background: #15151B; color: #fff; font-size: 20px; line-height: 1;
}
.section-head h1 { font-size: 18px; font-weight: 900; letter-spacing: .5px; margin: 0; }
.section-body { padding: 14px 16px calc(env(safe-area-inset-bottom, 0px) + 24px); display: flex; flex-direction: column; gap: 14px; }
.muted { color: rgba(255,255,255,.55); font-size: 14px; }
h2 { font-size: 12px; letter-spacing: 1.6px; color: #F2B928; text-transform: uppercase; margin: 4px 0 8px; }

/* calendario */
.court-block { display: flex; flex-direction: column; }
.match-row {
  display: grid; grid-template-columns: 74px 1fr auto; gap: 10px; align-items: center;
  background: #15151B; border: 1px solid rgba(255,255,255,.08); border-radius: 12px;
  padding: 10px 12px; margin-bottom: 8px;
}
.match-row.live { border-color: rgba(229,72,77,.5); }
.mr-time { display: flex; flex-direction: column; gap: 3px; align-items: flex-start; }
.pill { font-size: 10.5px; font-weight: 800; letter-spacing: .8px; background: rgba(255,255,255,.08); border-radius: 6px; padding: 3px 7px; }
.pill.live { background: rgba(229,72,77,.18); color: #FF8589; display: inline-flex; align-items: center; gap: 4px; }
.pill.live .dot { width: 6px; height: 6px; border-radius: 50%; background: #E5484D; }
.pill.done { color: rgba(255,255,255,.5); }
.stage { color: #F2B928; font-size: 9.5px; letter-spacing: 1px; font-weight: 800; }
.mr-teams { display: flex; flex-direction: column; gap: 3px; font-weight: 800; font-size: 13.5px; min-width: 0; }
.mr-teams span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; opacity: .85; }
.mr-teams .team-line { display: flex; align-items: center; gap: 7px; }
.team-line img { width: 22px; height: 22px; flex: none; object-fit: contain; border-radius: 50%; background: #fff; }
.mr-teams .winner { opacity: 1; color: #F2B928; }
.mr-score { display: grid; grid-template-rows: auto auto auto; justify-items: end; font-variant-numeric: tabular-nums; }
.mr-score b { font-size: 15px; line-height: 1.25; }
.mr-score .live-pt { color: #F2B928; font-size: 19px; font-weight: 900; }
.mr-score small { color: rgba(255,255,255,.45); font-size: 10px; grid-column: 1; }

/* classifiche */
.group-block table { width: 100%; border-collapse: collapse; background: #15151B; border-radius: 12px; overflow: hidden; }
.group-block th { font-size: 10px; letter-spacing: 1px; color: rgba(255,255,255,.5); padding: 9px 6px; text-align: center; border-bottom: 1px solid rgba(255,255,255,.08); }
.group-block td { padding: 10px 6px; text-align: center; font-size: 13px; border-bottom: 1px solid rgba(255,255,255,.05); font-variant-numeric: tabular-nums; }
.tl { text-align: left !important; padding-left: 12px !important; }
.pos { color: rgba(255,255,255,.5); width: 28px; }
.name { font-weight: 800; }
.standing-team { display: inline-flex; align-items: center; gap: 7px; }
.standing-team img { width: 24px; height: 24px; flex: none; object-fit: contain; border-radius: 50%; background: #fff; }
.w { color: #F2B928; font-weight: 800; }
.pt { font-weight: 900; color: #fff; }
tr.top .pos { color: #00C897; font-weight: 900; }
.legend { font-size: 11px; color: rgba(255,255,255,.4); }

/* tabellone */
.bracket-card { background: #15151B; border: 1px solid rgba(255,255,255,.08); border-radius: 12px; padding: 10px 12px; margin-bottom: 8px; }
.bracket-card.live { border-color: rgba(229,72,77,.5); }
.bc-row { display: flex; justify-content: space-between; font-weight: 800; font-size: 14px; padding: 3px 0; opacity: .8; }
.bc-row.winner { opacity: 1; color: #F2B928; }
.bc-meta { display: flex; justify-content: space-between; align-items: center; margin-top: 6px; font-size: 11px; color: rgba(255,255,255,.5); }
</style>
