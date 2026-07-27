<script setup>
// Tabellone da proiettare a schermo intero (/proietta/:slug/:court).
// Vista pubblica (nessun login): pensata per un proiettore/monitor rivolto al
// pubblico. 80% larghezza = punteggio della partita in corso sul campo (punti
// del set + set vinti), 20% destra = classifica aggiornata del girone.
// Aggiornamento real-time via SSE (stesso stream del resto del torneo).
import { computed, onMounted, ref } from 'vue'
import { useTournamentStream } from '@/composables/useTournamentStream'

const props = defineProps({
  slug: { type: String, required: true },
  court: { type: String, default: '' } // vuoto = prima partita live di qualsiasi campo
})

const matches = ref([])
const groups = ref([])
const allowDraws = ref(true)
const qualifiersPerGroup = ref(2)
const tournamentName = ref('')
const loading = ref(true)

async function load () {
  try {
    const [mr, sr] = await Promise.all([
      fetch(`/api/v1/tournaments/${props.slug}/matches`, { cache: 'no-store' }),
      fetch(`/api/v1/tournaments/${props.slug}/standings`, { cache: 'no-store' })
    ])
    if (mr.ok) matches.value = (await mr.json()).matches ?? []
    if (sr.ok) {
      const s = await sr.json()
      groups.value = s.groups ?? []
      allowDraws.value = s.allowDraws !== false
      qualifiersPerGroup.value = Math.max(1, Number(s.qualifiersPerGroup) || 2)
    }
  } catch (e) { /* mantiene i dati precedenti in caso di blip di rete */ }
  finally { loading.value = false }
}

// Partita da mostrare: la live sul campo indicato (o la prima live in assoluto
// se il campo non è specificato).
const live = computed(() => {
  const liveMatches = matches.value.filter(m => m.status === 'live')
  if (props.court) return liveMatches.find(m => m.court === props.court) ?? null
  return liveMatches[0] ?? null
})

// Se non c'è nulla di live: la prossima partita in programma sul campo.
const nextOnCourt = computed(() => {
  const sched = matches.value.filter(m => m.status === 'scheduled' && (!props.court || m.court === props.court))
  return sched[0] ?? null
})

// Classifica mostrata: il girone della partita live (se identificabile),
// altrimenti il primo girone disponibile.
const shownGroup = computed(() => {
  if (!groups.value.length) return null
  const g = live.value?.group
  if (g) {
    const found = groups.value.find(x => x.group === g)
    if (found) return found
  }
  return groups.value[0]
})

// La fase finale inizia solo quando tutti gli incontri dei gironi sono chiusi
// e il tabellone è stato effettivamente generato. Fino a quel momento resta
// visibile la classifica, anche dopo l'ultima gara ma prima della generazione.
const finalStageMatches = computed(() =>
  matches.value.filter(match => String(match.stage || '').trim())
)
const finalPhase = computed(() => {
  const groupMatches = matches.value.filter(match => !String(match.stage || '').trim())
  return finalStageMatches.value.length > 0 &&
    groupMatches.length > 0 &&
    groupMatches.every(match => match.status === 'finished')
})

const stageOrder = ['OTTAVI', 'QUARTI', 'SEMIFINALE', 'FINALE 3° POSTO', 'FINALE']
const bracketRounds = computed(() => {
  const byStage = new Map()
  for (const match of finalStageMatches.value) {
    const stage = String(match.stage || 'FASE FINALE').trim().toUpperCase()
    if (!byStage.has(stage)) byStage.set(stage, [])
    byStage.get(stage).push(match)
  }
  return [...byStage.entries()]
    .sort(([a], [b]) => {
      const ai = stageOrder.indexOf(a)
      const bi = stageOrder.indexOf(b)
      return (ai < 0 ? 99 : ai) - (bi < 0 ? 99 : bi)
    })
    .map(([stage, roundMatches]) => ({ stage, matches: roundMatches }))
})

onMounted(load)
useTournamentStream(() => props.slug, load)
</script>

<template>
  <div class="proj">
    <!-- 80%: TABELLONE PARTITA -->
    <main class="board">
      <header class="board-top">
        <span class="court">{{ court || 'Tabellone' }}</span>
        <span v-if="live" class="live"><span class="dot"></span>LIVE</span>
        <span v-else class="idle-pill">In attesa</span>
        <span v-if="live && live.stage" class="stage">{{ live.stage }}</span>
        <span v-else-if="live && live.group" class="stage">Girone {{ live.group }}</span>
      </header>

      <div v-if="live" class="scoreboard">
        <div class="team">
          <div class="tname">{{ live.teamA }}</div>
          <div class="points">{{ live.curA }}</div>
          <div class="sets"><span class="sets-lbl">SET</span> {{ live.scoreA }}</div>
        </div>

        <div class="mid">
          <div class="setlabel">{{ live.setLabel || '—' }}</div>
          <div class="colon">:</div>
          <div class="setlist">{{ live.sets.length ? live.sets.join('   ') : '&nbsp;' }}</div>
        </div>

        <div class="team">
          <div class="tname">{{ live.teamB }}</div>
          <div class="points">{{ live.curB }}</div>
          <div class="sets"><span class="sets-lbl">SET</span> {{ live.scoreB }}</div>
        </div>
      </div>

      <div v-else class="idle-board">
        <div class="idle-title">In attesa della prossima partita</div>
        <div v-if="nextOnCourt" class="idle-next">
          <span class="itime">{{ nextOnCourt.time || 'Orario da definire' }}</span>
          <span class="iteams">{{ nextOnCourt.teamA }} <b>vs</b> {{ nextOnCourt.teamB }}</span>
        </div>
      </div>
    </main>

    <!-- 20%: classifica nei gironi, tabellone dopo la conclusione dei gironi -->
    <aside class="standings" :class="{ 'bracket-mode': finalPhase, dense: finalStageMatches.length > 8 }">
      <template v-if="finalPhase">
        <h2>Tabellone</h2>
        <div class="bracket-rounds">
          <section v-for="round in bracketRounds" :key="round.stage" class="bracket-round">
            <h3>{{ round.stage }}</h3>
            <article
              v-for="match in round.matches"
              :key="match.id"
              class="bracket-match"
              :class="[`is-${match.status}`]"
            >
              <div class="bracket-meta">
                <span>{{ match.court || 'Campo da definire' }}</span>
                <b v-if="match.status === 'live'">LIVE</b>
                <span v-else-if="match.time">{{ match.time }}</span>
              </div>
              <div class="bracket-team" :class="{ winner: match.status === 'finished' && match.scoreA > match.scoreB }">
                <span>{{ match.teamA || 'Da definire' }}</span><b>{{ match.status === 'scheduled' ? '–' : match.scoreA }}</b>
              </div>
              <div class="bracket-team" :class="{ winner: match.status === 'finished' && match.scoreB > match.scoreA }">
                <span>{{ match.teamB || 'Da definire' }}</span><b>{{ match.status === 'scheduled' ? '–' : match.scoreB }}</b>
              </div>
            </article>
          </section>
        </div>
      </template>

      <template v-else>
        <h2>{{ shownGroup && shownGroup.group ? `Girone ${shownGroup.group}` : 'Classifica' }}</h2>
        <table v-if="shownGroup && shownGroup.rows.length">
          <thead>
            <tr><th class="c">#</th><th class="tl">Squadra</th><th class="c">G</th><th class="c">Pt</th></tr>
          </thead>
          <tbody>
            <tr v-for="(row, i) in shownGroup.rows" :key="row.teamId" :class="{ top: i < qualifiersPerGroup }">
              <td class="c pos">{{ i + 1 }}</td>
              <td class="tl name">{{ row.team }}</td>
              <td class="c">{{ row.played }}</td>
              <td class="c pt">{{ row.points }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="st-empty">La classifica apparirà dopo le prime partite concluse.</p>
      </template>
    </aside>
  </div>
</template>

<style scoped>
.proj {
  position: fixed; inset: 0; display: flex; width: 100vw; height: 100vh; overflow: hidden;
  background: #0a0a0e; color: #f1f5f9;
  font-family: 'Inter', system-ui, -apple-system, sans-serif;
}

/* ============ TABELLONE (80%) ============ */
.board {
  flex: 0 0 80%; display: flex; flex-direction: column; min-width: 0;
  padding: 2.2vh 2vw; gap: 2vh;
}
.board-top {
  display: flex; align-items: center; gap: 1.4vw; flex: none;
}
.board-top .court {
  font-size: 3.4vh; font-weight: 900; letter-spacing: .04em; text-transform: uppercase;
  color: #f2b928;
}
.board-top .live {
  display: inline-flex; align-items: center; gap: .7vw;
  background: #e5484d; color: #fff; font-weight: 900; font-size: 2.2vh;
  padding: .5vh 1.4vw; border-radius: 999px; letter-spacing: .1em;
}
.board-top .live .dot {
  width: 1.4vh; height: 1.4vh; border-radius: 50%; background: #fff;
  animation: blink 1s steps(2, start) infinite;
}
@keyframes blink { to { opacity: .25; } }
.board-top .idle-pill {
  background: rgba(255,255,255,.1); color: #cbd5e1; font-weight: 800; font-size: 2.2vh;
  padding: .5vh 1.4vw; border-radius: 999px;
}
.board-top .stage {
  margin-left: auto; font-size: 2.4vh; font-weight: 800; color: #94a3b8;
  text-transform: uppercase; letter-spacing: .06em;
}

/* Griglia punteggio: due squadre + centro */
.scoreboard {
  flex: 1 1 auto; display: grid; grid-template-columns: 1fr auto 1fr;
  align-items: center; gap: 2vw; min-height: 0;
}
.team {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 1.5vh; min-width: 0; text-align: center;
}
.team .tname {
  font-size: 4.6vh; font-weight: 900; text-transform: uppercase; letter-spacing: .01em;
  line-height: 1.05; max-width: 100%; overflow-wrap: anywhere;
}
.team .points {
  font-size: 34vh; font-weight: 900; line-height: .9; font-variant-numeric: tabular-nums;
  color: #fff; text-shadow: 0 0 6vh rgba(242,185,40,.18);
}
.team .sets {
  font-size: 4vh; font-weight: 900; color: #f2b928; font-variant-numeric: tabular-nums;
  display: inline-flex; align-items: baseline; gap: .8vw;
}
.team .sets .sets-lbl { font-size: 2.2vh; color: #94a3b8; letter-spacing: .1em; }

.mid {
  display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 2vh;
}
.mid .setlabel {
  font-size: 2.6vh; font-weight: 800; color: #94a3b8; text-transform: uppercase; letter-spacing: .08em;
  white-space: nowrap;
}
.mid .colon { font-size: 12vh; font-weight: 900; line-height: 1; color: rgba(255,255,255,.25); }
.mid .setlist {
  font-size: 2.4vh; font-weight: 800; color: #cbd5e1; font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

/* Stato in attesa */
.idle-board {
  flex: 1 1 auto; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 3vh;
}
.idle-title { font-size: 5vh; font-weight: 900; color: #64748b; text-align: center; }
.idle-next { display: flex; flex-direction: column; align-items: center; gap: 1.5vh; }
.idle-next .itime { font-size: 3vh; font-weight: 800; color: #f2b928; }
.idle-next .iteams { font-size: 4vh; font-weight: 900; text-transform: uppercase; text-align: center; }
.idle-next .iteams b { color: #64748b; margin: 0 1vw; }

/* ============ CLASSIFICA (20%) ============ */
.standings {
  flex: 0 0 20%; height: 100%; background: #101017; border-left: 1px solid rgba(255,255,255,.09);
  display: flex; flex-direction: column; padding: 2.2vh 1.2vw; box-sizing: border-box; min-width: 0;
}
.standings h2 {
  margin: 0 0 1.6vh; font-size: 2.8vh; font-weight: 900; color: #f2b928;
  text-transform: uppercase; letter-spacing: .04em; text-align: center; flex: none;
}
.standings table { width: 100%; border-collapse: collapse; }
.standings th {
  font-size: 1.6vh; color: #94a3b8; font-weight: 800; text-transform: uppercase; letter-spacing: .05em;
  padding: .8vh .4vw; border-bottom: 2px solid rgba(255,255,255,.12);
}
.standings td {
  font-size: 2.1vh; font-weight: 700; padding: 1vh .4vw; border-bottom: 1px solid rgba(255,255,255,.06);
  font-variant-numeric: tabular-nums;
}
.standings .c { text-align: center; }
.standings .tl { text-align: left; }
.standings td.name { overflow-wrap: anywhere; line-height: 1.15; }
.standings td.pos { color: #94a3b8; }
.standings td.pt { color: #f2b928; font-weight: 900; }
.standings tr.top td.pos { color: #f2b928; }
.standings tr.top td.name { color: #fff; }
.st-empty { color: #64748b; font-size: 2vh; text-align: center; margin-top: 3vh; line-height: 1.4; }

/* Tabellone compatto della fase finale: tutti i turni restano visibili nel
   pannello laterale del proiettore, senza richiedere interazione o scroll. */
.bracket-rounds { min-height: 0; display: flex; flex-direction: column; gap: .7vh; overflow: hidden; }
.bracket-round { display: flex; flex-direction: column; gap: .35vh; }
.bracket-round h3 {
  margin: 0; color: #94a3b8; font-size: 1.35vh; font-weight: 900;
  letter-spacing: .08em; text-transform: uppercase;
}
.bracket-match {
  padding: .5vh .55vw; border-radius: .6vh; background: rgba(255,255,255,.045);
  border: 1px solid rgba(255,255,255,.08);
}
.bracket-match.is-live {
  border-color: rgba(229,72,77,.8); background: rgba(229,72,77,.12);
  box-shadow: 0 0 1.2vh rgba(229,72,77,.18);
}
.bracket-meta {
  display: flex; justify-content: space-between; gap: .4vw; margin-bottom: .2vh;
  color: #64748b; font-size: 1.05vh; font-weight: 800; text-transform: uppercase;
}
.bracket-meta b { color: #f87171; letter-spacing: .08em; }
.bracket-team {
  display: flex; align-items: center; justify-content: space-between; gap: .45vw;
  color: #cbd5e1; font-size: 1.45vh; font-weight: 750; line-height: 1.25;
}
.bracket-team span { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.bracket-team b { color: #94a3b8; font-size: 1.6vh; font-variant-numeric: tabular-nums; }
.bracket-team.winner { color: #fff; font-weight: 900; }
.bracket-team.winner b { color: #f2b928; }
.standings.dense { padding-top: 1.5vh; padding-bottom: 1.5vh; }
.standings.dense h2 { margin-bottom: .8vh; font-size: 2.3vh; }
.standings.dense .bracket-rounds { gap: .35vh; }
.standings.dense .bracket-round { gap: .18vh; }
.standings.dense .bracket-match { padding-top: .22vh; padding-bottom: .22vh; }
.standings.dense .bracket-meta { display: none; }
.standings.dense .bracket-team { font-size: 1.15vh; line-height: 1.15; }
.standings.dense .bracket-team b { font-size: 1.25vh; }
</style>
