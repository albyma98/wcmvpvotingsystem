<script setup>
import { inject, onBeforeUnmount, proxyRefs, ref, watch } from 'vue';
import { useController } from '../../components/AdminLottery.vue?admin-v2-controller';
import V2Panel from '../components/shared/V2Panel.vue';
import V2Icon from '../components/shared/V2Icon.vue';

const props = defineProps({ detail: { type: String, default: '' } });
const c = inject('admin-v2');
const initializing = ref(true);
let disposed = false;
const l = proxyRefs(useController({}, () => {}, {
 onUnauthorized() { c.handleUnauthorized(); },
 onReturn() { c.navigate('closing'); },
 onLifecycleStart() { if (!disposed) initializing.value = true; },
 onLifecycleComplete() { if (!disposed) initializing.value = false; },
 onLifecycleError(error) { if (!disposed) { initializing.value = false; c.notify(error?.response?.data?.message || 'Impossibile caricare la lotteria. Riprova.', 'error'); } },
}));
watch([() => props.detail, () => l.events], () => {
 const eventId = Number(props.detail);
 if (eventId && l.events.some(event => Number(event.id) === eventId) && l.selectedEventId !== eventId) l.selectedEventId = eventId;
}, { immediate: true });
watch(() => l.lastWinner, winner => { if (winner && !disposed) c.notify(`Vincitore assegnato: ${l.formatTicketDisplay(winner)}`); });
function selectEvent(event) {
 if (l.isDrawing) return;
 c.navigate('lottery', String(event.target.value));
}
function draw(prize) {
 c.confirmAction('Estrarre il vincitore?', `Verrà assegnato il premio “${prize.name || `Premio ${prize.position}`}” a uno dei ${l.allowedCodesCount} ticket ammessi. L’assegnazione e gli SMS seguono le regole esistenti.`, () => { if (!disposed) l.startDraw(prize); });
}
function reset(prize) {
 c.confirmAction('Annullare il vincitore?', `L’assegnazione del premio “${prize.name || `Premio ${prize.position}`}” verrà annullata. Le assegnazioni definitive della chiusura automatica restano protette dal sistema.`, () => c.run(`lottery-reset-${prize.id}`, () => l.resetPrize(prize), 'Vincitore annullato.'));
}
onBeforeUnmount(() => { disposed = true; l.clearTimers(); });
</script>

<template>
 <div class="v2-stack v2-lottery">
  <V2Panel v-if="l.isStaff" title="Accesso non disponibile"><p class="v2-muted">Il ruolo staff non può accedere alla lotteria.</p><button class="v2-button" @click="c.navigate('closing')">Gestisci votazioni</button></V2Panel>
  <template v-else>
   <V2Panel title="Lotteria della partita" description="Assegna i premi ai ticket validi della società e controlla i vincitori.">
    <div class="v2-toolbar v2-lottery-selection"><label class="v2-field">Evento<select :value="l.selectedEventId" :disabled="initializing || l.isLoadingEvents || !l.events.length || l.isDrawing" @change="selectEvent"><option :value="0" disabled>Seleziona un evento</option><option v-for="event in l.events" :key="event.id" :value="event.id">{{ l.eventLabel(event) }} · #{{ event.id }}</option></select></label><button class="v2-button" :disabled="!l.selectedEventId || l.isLoadingTickets || l.isDrawing || c.busy('lottery-tickets')" @click="c.run('lottery-tickets',()=>l.loadTickets(l.selectedEventId))"><V2Icon name="refresh" :size="16" />Aggiorna ticket</button><button class="v2-button" :disabled="l.isDrawing || initializing || c.busy('lottery-refresh')" @click="c.run('lottery-refresh',l.loadInitialData)">Aggiorna eventi</button></div>
    <p v-if="initializing || l.isLoadingEvents" class="v2-muted v2-lottery-loading" role="status">Caricamento eventi e premi…</p>
    <p v-if="l.isLoadingTickets" class="v2-muted v2-lottery-loading" role="status">Aggiornamento dei ticket ammessi…</p>
    <p v-if="l.globalError" class="v2-error" role="alert">{{ l.globalError }}</p>
    <p v-if="l.drawError" class="v2-error" role="alert">{{ l.drawError }}</p>
    <div v-if="!initializing && !l.isLoadingEvents && !l.events.length && !l.globalError" class="v2-empty">Nessun evento disponibile per la lotteria.<button v-if="c.isSuperAdmin" class="v2-button" @click="c.navigate('events')">Gestisci eventi</button></div>
    <div v-if="l.selectedEvent" class="v2-lottery-context"><span class="v2-badge">{{ l.eventLabel(l.selectedEvent) }}</span><span class="v2-muted">{{ c.formatEventDate(l.selectedEvent.start_datetime) }}<template v-if="l.selectedEvent.location"> · {{ l.selectedEvent.location }}</template></span><button v-if="c.isSuperAdmin" class="v2-link" @click="c.navigate('events',`${l.selectedEventId}/prizes`)">Configura premi</button></div>
   </V2Panel>
   <template v-if="l.selectedEventId">
    <div class="v2-grid-3"><article class="v2-metric"><span>Ticket ammessi</span><strong>{{ l.allowedCodesCount }}</strong><small>Codici validi per la prossima estrazione.</small></article><article class="v2-metric"><span>Ticket totali</span><strong>{{ l.totalCodesCount }}</strong><small>Tutti i codici registrati per questo evento.</small></article><article class="v2-metric"><span>Premi configurati</span><strong>{{ l.selectedPrizes.length }}</strong><small>Premi associati all’evento selezionato.</small></article></div>
    <V2Panel v-if="!l.selectedPrizes.length" title="Nessun premio configurato"><p class="v2-muted">Aggiungi i premi dalla configurazione dell’evento prima di eseguire un’estrazione.</p><button v-if="c.isSuperAdmin" class="v2-button v2-primary" @click="c.navigate('events',`${l.selectedEventId}/prizes`)">Configura premi evento</button></V2Panel>
    <template v-else>
     <section class="v2-lottery-display" :aria-busy="l.isDrawing"><span class="v2-eyebrow">{{ l.isDrawing ? 'ESTRAZIONE IN CORSO' : 'IL TABELLONE DELLA LOTTERIA' }}</span><h2 v-if="l.currentPrizeLabel">{{ l.currentPrizeLabel }}</h2><p class="v2-lottery-code" :aria-live="l.isDrawing ? 'off' : 'polite'">{{ l.currentDisplay }}</p><p v-if="l.isDrawing" role="status" class="v2-muted">Estrazione in corso. Attendi la conferma dell’assegnazione.</p><p v-else-if="l.lastWinner" class="v2-success" role="status">Ultimo vincitore: <strong>{{ l.lastWinnerDisplay }}</strong></p><p v-else class="v2-muted">Scegli un premio dall’elenco per iniziare.</p></section>
     <div class="v2-stack"><article v-for="prize in l.selectedPrizes" :key="prize.id" class="v2-panel v2-lottery-prize" :class="{'v2-lottery-prize-current':l.currentPrizeId === prize.id}"><div class="v2-lottery-prize-number"><V2Icon name="lottery" :size="22" /><small>#{{ prize.position }}</small></div><div class="v2-lottery-prize-info"><h2>{{ prize.name || `Premio ${prize.position}` }}</h2><p v-if="prize.winner" class="v2-lottery-winner">Vincitore: <strong>{{ prize.winnerDisplay }}</strong></p><p v-else class="v2-muted">Premio in attesa di assegnazione.</p><span v-if="prize.winner" class="v2-badge" :class="prize.winner.status === 'notified_failed' ? 'danger' : prize.winner.status === 'notified' ? 'success' : ''">SMS: {{ l.smsStatusLabel(prize.winner.status) }}</span></div><div class="v2-toolbar v2-lottery-prize-actions"><button class="v2-button v2-primary" :disabled="!l.canDrawPrize(prize) || !!l.isResettingPrize" @click="draw(prize)">{{ prize.winner ? 'Premio assegnato' : l.isDrawing && l.currentPrizeId === prize.id ? 'Estrazione in corso…' : 'Estrai vincitore' }}</button><button v-if="prize.winner" class="v2-button v2-danger" :disabled="l.isDrawing || !!l.isResettingPrize || c.busy(`lottery-reset-${prize.id}`)" @click="reset(prize)">{{ l.isResettingPrize === prize.id ? 'Annullamento…' : 'Annulla vincitore' }}</button></div></article></div>
    </template>
   </template>
  </template>
 </div>
</template>

<style scoped>
.v2-lottery-selection{align-items:flex-end}.v2-lottery-selection>.v2-field{max-width:570px;flex:1;min-width:min(260px,100%)}.v2-lottery-loading,.v2-lottery .v2-error{margin-top:18px!important}.v2-lottery-context{display:flex;gap:12px;align-items:center;flex-wrap:wrap;padding-top:20px;margin-top:22px;border-top:1px solid var(--v2-line);font-size:12px}.v2-lottery-display{display:grid;gap:18px;text-align:center;justify-items:center;border:1px solid #bdd8ca;padding:32px 24px;border-radius:14px;background:#edf6f0}.v2-lottery-display h2{font-size:20px}.v2-lottery-code{font-size:clamp(22px,3vw,36px);font-weight:700;letter-spacing:-.8px;min-height:110px;padding:26px;display:grid;place-content:center;width:min(700px,100%);background:#fff;border:1px solid #c4dacf;border-radius:12px;overflow-wrap:anywhere}.v2-lottery-prize{display:flex;gap:20px;align-items:center}.v2-lottery-prize-current{border-color:#8fbea8!important}.v2-lottery-prize-number{display:grid;justify-items:center;align-content:center;gap:5px;flex:none;background:var(--v2-soft);color:var(--v2-primary);width:58px;height:66px;border-radius:10px}.v2-lottery-prize-info{flex:1;min-width:0;display:grid;gap:8px}.v2-lottery-prize-info>.v2-badge{justify-self:start}.v2-lottery-winner{color:var(--v2-success);overflow-wrap:anywhere}.v2-lottery .v2-empty{display:grid;gap:18px;justify-items:center;margin-top:20px}.v2-lottery .v2-panel>p+.v2-button{margin-top:18px}@media(max-width:1100px){.v2-lottery-prize{flex-wrap:wrap}.v2-lottery-prize-actions{margin-left:78px}}@media(max-width:640px){.v2-lottery-selection>.v2-field{max-width:none;width:100%}.v2-lottery-selection>.v2-button{flex:1}.v2-lottery-prize{gap:12px}.v2-lottery-prize-actions{margin-left:0;width:100%}.v2-lottery-prize-actions>.v2-button{flex:1}.v2-lottery-display{padding:24px 16px}.v2-lottery-code{padding:20px 12px}.v2-lottery-context{align-items:flex-start;flex-direction:column}}
</style>
