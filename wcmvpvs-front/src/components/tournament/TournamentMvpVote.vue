<script setup>
// Votazione MVP del pubblico — un box per squadra, i giocatori della rosa
// cliccabili per votare. 1 voto per device (modificabile). Lato tifoso NON si
// vedono conteggi/percentuali: solo i nomi e il proprio voto (✓). I risultati
// sono visibili solo all'admin. Stesso mood pop della home torneo (dark + gold).
import { computed, onMounted, ref } from 'vue'
import { getOrCreateDeviceId } from '@/deviceId'
import { useTournamentStream } from '@/composables/useTournamentStream'

const props = defineProps({
  slug: { type: String, required: true }
})

const teams = ref([])
const myVote = ref(0)          // playerId votato da questo device (0 = nessuno)
const loading = ref(true)
const error = ref('')
const voting = ref(0)          // playerId con POST in corso
const justVoted = ref('')      // nome appena votato → banner conferma

const deviceId = getOrCreateDeviceId()
const hasVoted = computed(() => myVote.value !== 0)

function applyBoard (b) {
  teams.value = b.teams ?? []
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

async function vote (candidate) {
  if (voting.value) return
  voting.value = candidate.id
  try {
    const r = await fetch(`/api/v1/tournaments/${props.slug}/mvp/vote`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Device-ID': deviceId },
      body: JSON.stringify({ playerId: candidate.id })
    })
    if (!r.ok) throw new Error(r.status)
    applyBoard(await r.json())
    justVoted.value = candidate.name
    setTimeout(() => { if (justVoted.value === candidate.name) justVoted.value = '' }, 3200)
  } catch (e) {
    error.value = 'Voto non registrato, riprova.'
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
      <p>Scegli il giocatore più forte del torneo. Tocca un nome per votare — puoi cambiare quando vuoi.</p>
    </header>

    <transition name="pop">
      <div v-if="justVoted" class="voted-banner">✓ Hai votato <b>{{ justVoted }}</b></div>
    </transition>
    <p v-if="error" class="mvp-error">{{ error }}</p>

    <p v-if="loading" class="mvp-muted">Caricamento…</p>
    <p v-else-if="!teams.length && !error" class="mvp-muted">
      Le rose delle squadre non sono ancora disponibili. Torna più tardi!
    </p>

    <div class="team-grid">
      <section v-for="t in teams" :key="t.id" class="team-box">
        <header class="tb-head">
          <span class="tb-name">{{ t.name }}</span>
        </header>
        <ul class="tb-players">
          <li v-for="c in t.candidates" :key="c.id">
            <button
              class="player"
              :class="{ chosen: myVote === c.id, dim: hasVoted && myVote !== c.id }"
              :disabled="!!voting"
              @click="vote(c)"
            >
              <span class="p-name">{{ c.name }}</span>
              <span v-if="voting === c.id" class="p-right">…</span>
              <span v-else-if="myVote === c.id" class="p-right"><span class="check">✓</span></span>
            </button>
          </li>
        </ul>
      </section>
    </div>
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
.mvp-muted { text-align: center; color: rgba(255,255,255,.5); font-size: 14px; padding: 20px 0; }

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
