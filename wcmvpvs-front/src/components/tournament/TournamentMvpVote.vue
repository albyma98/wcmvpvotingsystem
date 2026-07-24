<script setup>
// Votazione MVP del pubblico — un box per squadra, i giocatori della rosa
// cliccabili per votare. 1 voto per device (modificabile). Lato tifoso NON si
// vedono conteggi/percentuali: solo i nomi e il proprio voto (✓). I risultati
// sono visibili solo all'admin. Stesso mood pop della home torneo (dark + gold).
import { computed, onMounted, ref } from 'vue'
import { getOrCreateDeviceId } from '@/deviceId'
import { useTournamentStream } from '@/composables/useTournamentStream'
import { track as posthogTrack, EVENTS as PH_EVENTS } from '@/lib/track'

const props = defineProps({
  slug: { type: String, required: true }
})

const teams = ref([])
const byGender = ref(true)     // true = MVP uomo/donna (2 voti), false = MVP unico (1 voto)
const myVoteMale = ref(0)      // playerId uomo votato da questo device (0 = nessuno)
const myVoteFemale = ref(0)    // playerId donna votato da questo device (0 = nessuno)
const myVote = ref(0)          // playerId votato in modalità MVP unico (0 = nessuno)
const activeGender = ref('male') // categoria mostrata (toggle Uomo/Donna)
const loading = ref(true)
const error = ref('')
const voting = ref(0)          // playerId con POST in corso
const justVoted = ref('')      // nome appena votato → banner conferma
const tournamentStarted = ref(true)

const deviceId = getOrCreateDeviceId()

// Voto corrente e stato "ho già votato". In modalità unica c'è un solo voto;
// in modalità separata dipende dalla categoria (Uomo/Donna) attiva.
const activeVote = computed(() => {
  if (!byGender.value) return myVote.value
  return activeGender.value === 'female' ? myVoteFemale.value : myVoteMale.value
})
const hasVotedActive = computed(() => activeVote.value !== 0)

// Squadre votabili: in modalità unica tutti i candidati; in modalità separata
// solo quelli del genere attivo (scartando le squadre senza candidati adatti).
const teamsForActive = computed(() => {
  if (!byGender.value) {
    return (teams.value ?? [])
      .map(t => ({ ...t, candidates: t.candidates ?? [] }))
      .filter(t => t.candidates.length)
  }
  const g = activeGender.value
  return (teams.value ?? [])
    .map(t => ({ ...t, candidates: (t.candidates ?? []).filter(c => (c.gender || 'male') === g) }))
    .filter(t => t.candidates.length)
})

function applyBoard (b) {
  teams.value = b.teams ?? []
  byGender.value = b.byGender !== false
  tournamentStarted.value = b.tournamentStarted !== false
  myVoteMale.value = b.myVoteMale ?? 0
  myVoteFemale.value = b.myVoteFemale ?? 0
  myVote.value = b.myVote ?? 0
}

async function load () {
  try {
    const r = await fetch(`/api/v1/tournaments/${props.slug}/mvp`, {
      headers: { 'X-Device-ID': deviceId }
    })
    if (!r.ok) throw new Error(r.status)
    applyBoard(await r.json())
    error.value = ''
  } catch (e) {
    error.value = 'Votazione non disponibile al momento.'
  } finally {
    loading.value = false
  }
}

async function vote (candidate, team) {
  if (voting.value || !tournamentStarted.value) return
  const gender = candidate.gender || activeGender.value
  // Cambio voto o primo voto per questo slot? (prima dell'aggiornamento board)
  const changed = activeVote.value !== 0
  voting.value = candidate.id
  try {
    const r = await fetch(`/api/v1/tournaments/${props.slug}/mvp/vote`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Device-ID': deviceId },
      body: JSON.stringify({ playerId: candidate.id })
    })
    if (!r.ok) {
      const payload = await r.json().catch(() => ({}))
      if (payload.error === 'tournament_not_started') tournamentStarted.value = false
      throw new Error(payload.error || r.status)
    }
    applyBoard(await r.json())
    posthogTrack(PH_EVENTS.TOURNAMENT_MVP_VOTED, {
      tournament_slug: props.slug,
      surface: 'tournament',
      gender,
      player_id: candidate.id,
      team_id: team?.id,
      team: team?.name,
      changed,
    })
    justVoted.value = candidate.name
    setTimeout(() => { if (justVoted.value === candidate.name) justVoted.value = '' }, 3200)
  } catch (e) {
    error.value = e.message === 'tournament_not_started'
      ? 'Il torneo non è ancora iniziato.'
      : 'Voto non registrato, riprova.'
    setTimeout(() => { error.value = '' }, 2500)
  } finally {
    voting.value = 0
  }
}

onMounted(load)
// Ricarica lo stato (incl. il proprio voto) quando qualcosa cambia (SSE push).
useTournamentStream(() => props.slug, load)
</script>

<template>
  <div class="mvp">
    <header class="mvp-intro">
      <div class="trophy">🏆</div>
      <h2>Vota il tuo MVP</h2>
      <p v-if="byGender">Hai due voti: un MVP <b>uomo</b> e un MVP <b>donna</b>. Tocca un nome per votare — puoi cambiare quando vuoi.</p>
      <p v-else>Hai <b>un voto</b>. Tocca il nome del giocatore che ti ha impressionato di più — puoi cambiare quando vuoi.</p>
    </header>

    <p v-if="!tournamentStarted" class="mvp-locked">
      🔒 Il torneo non è ancora iniziato. Le votazioni MVP apriranno all'inizio.
    </p>

    <transition name="pop">
      <div v-if="justVoted" class="voted-banner">✓ Hai votato <b>{{ justVoted }}</b></div>
    </transition>
    <p v-if="error" class="mvp-error">{{ error }}</p>

    <p v-if="loading" class="mvp-muted">Caricamento…</p>
    <p v-else-if="!teams.length && !error" class="mvp-muted">
      Le rose delle squadre non sono ancora disponibili. Torna più tardi!
    </p>

    <template v-if="!loading && teams.length">
      <!-- Toggle categoria: Uomo / Donna, con ✓ quando il voto è già espresso.
           Nascosto in modalità MVP unico (un solo voto, nessuna distinzione). -->
      <div v-if="byGender" class="gender-switch" role="tablist">
        <button
          class="gs-btn"
          :class="{ active: activeGender === 'male' }"
          role="tab"
          @click="activeGender = 'male'"
        >
          Uomo <span v-if="myVoteMale" class="gs-check">✓</span>
        </button>
        <button
          class="gs-btn"
          :class="{ active: activeGender === 'female' }"
          role="tab"
          @click="activeGender = 'female'"
        >
          Donna <span v-if="myVoteFemale" class="gs-check">✓</span>
        </button>
      </div>

      <p v-if="byGender && !teamsForActive.length" class="mvp-muted">
        Nessun {{ activeGender === 'female' ? 'a giocatrice' : ' giocatore' }} in questa categoria.
      </p>

      <div class="team-grid">
        <section v-for="t in teamsForActive" :key="t.id" class="team-box">
          <header class="tb-head">
            <span class="tb-name">{{ t.name }}</span>
          </header>
          <ul class="tb-players">
            <li v-for="c in t.candidates" :key="c.id">
              <button
                class="player"
                :class="{ chosen: activeVote === c.id, dim: hasVotedActive && activeVote !== c.id }"
                :disabled="!!voting || !tournamentStarted"
                @click="vote(c, t)"
              >
                <span class="p-name">{{ c.name }}</span>
                <span v-if="voting === c.id" class="p-right">…</span>
                <span v-else-if="activeVote === c.id" class="p-right"><span class="check">✓</span></span>
              </button>
            </li>
          </ul>
        </section>
      </div>
    </template>
  </div>
</template>

<style scoped>
.mvp { display: flex; flex-direction: column; gap: 14px; }

/* Intro */
.mvp-intro { text-align: center; padding: 6px 8px 2px; }
.mvp-intro .trophy { font-size: 34px; line-height: 1; filter: drop-shadow(0 4px 10px rgba(242,185,40,.35)); }
.mvp-intro h2 {
  margin: 8px 0 4px; font-size: 22px; font-weight: 900; letter-spacing: .5px;
  background: linear-gradient(180deg, #FFE08A, #F2B928); -webkit-background-clip: text;
  background-clip: text; color: transparent; text-transform: none;
}
.mvp-intro p { margin: 0 auto; max-width: 320px; font-size: 13px; color: rgba(255,255,255,.6); line-height: 1.45; }

/* Banner conferma voto */
.voted-banner {
  position: sticky; top: 8px; z-index: 4; align-self: center;
  background: linear-gradient(135deg, #00C897, #00A578); color: #04150F;
  font-weight: 900; font-size: 13.5px; padding: 9px 18px; border-radius: 999px;
  box-shadow: 0 8px 22px rgba(0,200,151,.35);
}
.mvp-error { text-align: center; color: #FF8589; font-size: 13px; font-weight: 700; }
.mvp-locked {
  margin: 0; padding: 12px 14px; border: 1px solid rgba(242,185,40,.35);
  border-radius: 12px; background: rgba(242,185,40,.1); color: #FFD66B;
  text-align: center; font-size: 13px; font-weight: 800; line-height: 1.4;
}
.mvp-muted { text-align: center; color: rgba(255,255,255,.5); font-size: 14px; padding: 20px 0; }

/* Toggle categoria Uomo/Donna */
.gender-switch {
  display: flex; gap: 6px; align-self: center; padding: 4px;
  background: #15151b; border: 1px solid rgba(255,255,255,.1); border-radius: 999px;
}
.gs-btn {
  display: inline-flex; align-items: center; gap: 6px; border: none; cursor: pointer;
  background: transparent; color: rgba(255,255,255,.65); font-weight: 800; font-size: 14px;
  padding: 8px 20px; border-radius: 999px; transition: background .15s ease, color .15s ease;
  -webkit-tap-highlight-color: transparent;
}
.gs-btn.active {
  background: linear-gradient(180deg, #FFE08A, #F2B928); color: #14110A;
  box-shadow: 0 4px 12px rgba(242,185,40,.3);
}
.gs-check {
  display: inline-grid; place-items: center; width: 16px; height: 16px; border-radius: 50%;
  background: #00C897; color: #04150F; font-size: 10px; font-weight: 900;
}
.gs-btn.active .gs-check { background: #04150F; color: #F2B928; }

/* Griglia squadre: 2 colonne, si adatta */
.team-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 12px; }

.team-box {
  background: linear-gradient(180deg, #17171F, #101015);
  border: 1px solid rgba(255,255,255,.08); border-radius: 16px; overflow: hidden;
  display: flex; flex-direction: column;
}
.tb-head {
  padding: 10px 12px; border-bottom: 1px solid rgba(255,255,255,.07);
  background: rgba(242,185,40,.06);
}
/* Nome squadra: pieno, può andare a capo (niente troncamento) */
.tb-name { font-weight: 900; font-size: 14px; letter-spacing: .3px; text-transform: uppercase; line-height: 1.2; }

.tb-players { list-style: none; margin: 0; padding: 8px; display: flex; flex-direction: column; gap: 6px; }

/* Bottone giocatore */
.player {
  position: relative; width: 100%; display: flex; align-items: center; justify-content: space-between; gap: 8px;
  background: #1E1E27; border: 1px solid rgba(255,255,255,.1); border-radius: 11px;
  padding: 11px 12px; color: #fff; font-size: 14px; font-weight: 700; text-align: left;
  transition: transform .12s ease, border-color .15s ease, background .15s ease;
  -webkit-tap-highlight-color: transparent;
}
.player:not(:disabled):active { transform: scale(.97); }
.player:not(:disabled):hover { border-color: rgba(242,185,40,.5); }
.player:disabled { cursor: default; }
.player.dim { opacity: .62; }
.player.chosen {
  border-color: #F2B928;
  background: linear-gradient(90deg, rgba(242,185,40,.28), rgba(242,185,40,.12));
  box-shadow: 0 0 0 1px rgba(242,185,40,.5) inset;
}
/* Nome giocatore: sempre completo e leggibile, va a capo se serve */
.p-name { flex: 1; min-width: 0; line-height: 1.25; overflow-wrap: anywhere; }
.p-right { display: inline-flex; align-items: center; flex: none; }
.check {
  display: inline-grid; place-items: center; width: 20px; height: 20px; border-radius: 50%;
  background: #F2B928; color: #14110A; font-size: 13px; font-weight: 900;
}

/* Transizione banner */
.pop-enter-active { transition: transform .25s cubic-bezier(.22,1.4,.5,1), opacity .25s ease; }
.pop-leave-active { transition: transform .2s ease, opacity .2s ease; }
.pop-enter-from, .pop-leave-to { opacity: 0; transform: translateY(-8px) scale(.9); }
</style>
