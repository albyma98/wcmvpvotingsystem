<script setup>
// Sezioni fan del torneo — stesse variabili visive della home (dark + gold).
// Calendario, Classifiche e Tabellone sono dati veri dagli endpoint pubblici;
// le altre sezioni restano placeholder finché non hanno contenuto (P2/P3).
import { computed, onMounted, ref } from 'vue'
import { useTournamentStream } from '@/composables/useTournamentStream'
import TournamentGallery from '@/components/tournament/TournamentGallery.vue'
import TournamentMvpVote from '@/components/tournament/TournamentMvpVote.vue'

const props = defineProps({
  slug: { type: String, required: true },
  section: { type: String, required: true }
})
const emit = defineEmits(['navigate'])

const titles = {
  calendar: 'Calendario', standings: 'Classifiche', bracket: 'Tabellone',
  mvp: 'Vota MVP', prizes: 'Premi', gallery: 'Gallery',
  rules: 'Regolamento', event: 'Info Evento'
}

const matches = ref([])
const groups = ref([])
const photos = ref([])
const loading = ref(true)
const error = ref('')

async function load () {
  loading.value = true
  error.value = ''
  try {
    if (props.section === 'standings') {
      const r = await fetch(`/api/v1/tournaments/${props.slug}/standings`)
      if (!r.ok) throw new Error(r.status)
      groups.value = (await r.json()).groups ?? []
    } else if (props.section === 'calendar' || props.section === 'bracket') {
      const r = await fetch(`/api/v1/tournaments/${props.slug}/matches`)
      if (!r.ok) throw new Error(r.status)
      matches.value = (await r.json()).matches ?? []
    } else if (props.section === 'gallery') {
      const r = await fetch(`/api/v1/tournaments/${props.slug}/gallery`)
      if (!r.ok) throw new Error(r.status)
      photos.value = (await r.json()).photos ?? []
    }
  } catch (e) {
    error.value = 'Dati non disponibili al momento.'
  } finally {
    loading.value = false
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
onMounted(load)
useTournamentStream(props.slug, () => { if (hasData.value) load() })
</script>

<template>
  <div class="section-page">
    <header class="section-head">
      <button class="back" @click="emit('navigate', `/t/${props.slug}`)" aria-label="Indietro">‹</button>
      <h1>{{ titles[props.section] || props.section }}</h1>
    </header>

    <div class="section-body">
      <!-- GALLERY: sempre montata (niente flash di "Caricamento" sui refresh live) -->
      <TournamentGallery
        v-if="section === 'gallery'"
        :slug="slug" :photos="photos" @uploaded="load"
      />

      <!-- VOTA MVP: componente autonomo (fetch/voto/live via device) -->
      <TournamentMvpVote v-else-if="section === 'mvp'" :slug="slug" />

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
              <span :class="{ winner: m.status === 'finished' && m.scoreA > m.scoreB }">{{ m.teamA }}</span>
              <span :class="{ winner: m.status === 'finished' && m.scoreB > m.scoreA }">{{ m.teamB }}</span>
            </div>
            <div class="mr-score" v-if="m.status !== 'scheduled'">
              <b>{{ m.scoreA }}</b><b>{{ m.scoreB }}</b>
              <small v-if="m.sets.length">{{ m.sets.join(' ') }}</small>
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
            <thead><tr><th></th><th class="tl">Squadra</th><th>G</th><th>V</th><th>N</th><th>P</th><th>Set</th><th class="pt">Pt</th></tr></thead>
            <tbody>
              <tr v-for="(row, i) in g.rows" :key="row.teamId" :class="{ top: i < 2 }">
                <td class="pos">{{ i + 1 }}</td>
                <td class="tl name">{{ row.team }}</td>
                <td>{{ row.played }}</td>
                <td class="w">{{ row.wins }}</td>
                <td>{{ row.draws }}</td>
                <td>{{ row.losses }}</td>
                <td>{{ row.setsWon }}:{{ row.setsLost }}</td>
                <td class="pt">{{ row.points }}</td>
              </tr>
            </tbody>
          </table>
        </section>
        <p v-if="groups.length" class="legend">Primi 2 di ogni girone alla fase finale · Ordinamento: punti, quoziente set, quoziente punti</p>
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
.mr-teams .winner { opacity: 1; color: #F2B928; }
.mr-score { display: grid; grid-template-rows: auto auto auto; justify-items: end; font-variant-numeric: tabular-nums; }
.mr-score b { font-size: 15px; line-height: 1.25; }
.mr-score small { color: rgba(255,255,255,.45); font-size: 10px; grid-column: 1; }

/* classifiche */
.group-block table { width: 100%; border-collapse: collapse; background: #15151B; border-radius: 12px; overflow: hidden; }
.group-block th { font-size: 10px; letter-spacing: 1px; color: rgba(255,255,255,.5); padding: 9px 6px; text-align: center; border-bottom: 1px solid rgba(255,255,255,.08); }
.group-block td { padding: 10px 6px; text-align: center; font-size: 13px; border-bottom: 1px solid rgba(255,255,255,.05); font-variant-numeric: tabular-nums; }
.tl { text-align: left !important; padding-left: 12px !important; }
.pos { color: rgba(255,255,255,.5); width: 28px; }
.name { font-weight: 800; }
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
