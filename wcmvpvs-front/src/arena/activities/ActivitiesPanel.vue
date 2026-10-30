<script setup>
import { ref, computed, defineAsyncComponent } from "vue";
import Icon from "../ArenaIcon.vue";
import AppearanceControls from "../crowd/AppearanceControls.vue";
import {
  ACTIVITIES,
  TREASURE_NAMES,
  TREASURE_SPOTS,
  itemAvailable,
} from "./model.js";
const props = defineProps({
  state: Object,
  me: Object,
  neighbors: Array,
  busy: Boolean,
  demo: Boolean,
  active: { type: String, default: "" },
  invitation: String,
});
const emit = defineEmits([
  "close",
  "open",
  "command",
  "focus",
  "preview",
  "debug",
  "find",
  "neighbor",
]);
const category = ref("all"),
  preview = ref(null),
  choice = ref("3-0"),
  inviteMessage = ref("");
const DebugPanel = import.meta.env.DEV
  ? defineAsyncComponent(() => import("./DebugPanel.vue"))
  : null;
const match = computed(
  () =>
    [...props.state.matches]
      .filter((m) => m.open)
      .sort((a, b) => new Date(a.starts) - new Date(b.starts))[0] ||
    props.state.matches.find((m) => m.choice) ||
    props.state.matches[0],
);
const items = computed(() =>
  props.state.catalog.filter(
    (i) =>
      category.value === "all" ||
      (category.value === "clothes"
        ? ["shirt", "pants"].includes(i.slot)
        : category.value === "extras"
          ? ["hat", "glasses", "flag", "scarf"].includes(i.slot)
          : ["shoes", "hair", "celebration"].includes(i.slot)),
  ),
);
const owns = (key) => props.state.owned.includes(key);
const equipped = (item) => props.state.equipment[item.slot] === item.key;
function status(key) {
  const s = props.state;
  if (key === "treasure")
    return s.treasure.claimed
      ? "Completata oggi"
      : `${s.treasure.reward} monetine · Disponibile oggi`;
  if (key === "wardrobe")
    return `${s.owned.length} accessori · ${s.featured.filter((k) => !owns(k)).length} in evidenza da scoprire`;
  if (key === "predictions")
    return match.value?.open
      ? "Aperto · " + new Date(match.value.cutoff).toLocaleString("it-IT")
      : match.value?.choice
        ? "Pronostico registrato"
        : "In attesa del calendario";
  if (key === "neighbors")
    return props.me
      ? props.neighbors.length + " vicini nella tua zona"
      : "Accedi per trovare il tuo posto";
  return s.trophies.filter((t) => t.unlocked).length + " trofei conquistati";
}
function tryItem(item) {
  preview.value = item.key;
  emit("preview", { ...props.state.equipment, [item.slot]: item.key });
}
async function copy() {
  try {
    await navigator.clipboard.writeText(props.invitation);
    inviteMessage.value = "Link copiato. Condividilo con il tuo amico.";
  } catch {
    inviteMessage.value = "Seleziona e copia il link qui sotto.";
  }
}
</script>
<template>
  <section
    class="activity-sheet"
    :class="{ 'wardrobe-sheet': active === 'wardrobe' }"
    role="dialog"
    aria-modal="false"
    aria-labelledby="activity-title"
  >
    <header>
      <button
        v-if="active"
        @click="emit('open', '')"
        aria-label="Torna alle attività"
      >
        ←
      </button>
      <div>
        <small>LA TUA SETTIMANA IN TRIBUNA <b v-if="demo">· DEMO</b></small>
        <h2 id="activity-title">
          {{
            ACTIVITIES.find((a) => a.key === active)?.title || "Attività Arena"
          }}
        </h2>
      </div>
      <button @click="emit('close')" aria-label="Chiudi attività">×</button>
    </header>
    <div class="activity-body">
      <div class="activity-wallet">
        <span><Icon name="trophy" />{{ state.wallet }} monetine ABX</span
        ><span>{{ state.points }} punti · L{{ state.level || 1 }}</span>
      </div>
      <p v-if="!me" class="note">
        Accedi con il profilo della società per partecipare e conservare i
        progressi.
      </p>
      <div v-if="!active" class="activity-list">
        <button
          v-for="activity in ACTIVITIES"
          :key="activity.key"
          @click="emit('open', activity.key)"
        >
          <span class="activity-icon"><Icon :name="activity.icon" /></span
          ><span
            ><strong>{{ activity.title }}</strong
            ><small>{{ activity.description }}</small
            ><b>{{ status(activity.key) }}</b></span
          ><span>↗</span>
        </button>
      </div>
      <template v-if="active === 'treasure'">
        <div class="hero-symbol">✦</div>
        <h3>{{ TREASURE_NAMES[state.treasure.kind] }}</h3>
        <p>
          Un oggetto nascosto, la stessa sfida per tutta la società. Ruota il
          palazzetto, avvicinati e tocca il bagliore nel mondo 3D.
        </p>
        <blockquote>
          {{ TREASURE_SPOTS[state.treasure.location].hint }}
        </blockquote>
        <p class="note">
          {{ state.treasure.day }} · Reset a mezzanotte Europe/Rome ·
          {{ state.treasure.reward }} monetine
        </p>
        <strong v-if="state.treasure.claimed" class="success"
          >✓ Tesoro raccolto! Torna domani.</strong
        ><button v-else class="primary" @click="emit('find')">
          Cerca il tesoro
        </button>
      </template>
      <template v-if="active === 'wardrobe'">
        <details class="appearance-section">
          <summary>Aspetto del mio avatar</summary>
          <AppearanceControls
            :equipment="state.equipment"
            :seed="me?.seed || 1"
            :disabled="busy || !me"
            @change="
              (appearance) =>
                emit('command', { action: 'appearance', appearance })
            "
          />
        </details>
        <p class="note">
          Trascina nello spazio 3D per ruotare intorno al tuo avatar.
          L’anteprima non acquista l’oggetto.
        </p>
        <h3>In evidenza questa settimana</h3>
        <div class="featured">
          <span v-for="key in state.featured" :key="key">{{
            state.catalog.find((i) => i.key === key)?.title
          }}</span>
        </div>
        <div class="categories" aria-label="Categorie guardaroba">
          <button
            v-for="[key, label] in [
              ['all', 'Tutto'],
              ['clothes', 'Abbigliamento'],
              ['extras', 'Accessori'],
              ['custom', 'Personalizzazioni'],
            ]"
            :key="key"
            :aria-pressed="category === key"
            @click="category = key"
          >
            {{ label }}
          </button>
        </div>
        <div class="catalog">
          <article
            v-for="item in items"
            :key="item.key"
            :class="{ preview: preview === item.key }"
          >
            <button class="item-preview" :disabled="!me" @click="tryItem(item)">
              <Icon
                :name="
                  ['shirt', 'pants'].includes(item.slot)
                    ? 'fans'
                    : item.slot === 'celebration'
                      ? 'wave'
                      : 'scarf'
                "
              /><strong>{{ item.title }}</strong
              ><small>{{
                equipped(item)
                  ? "Indossato"
                  : owns(item.key)
                    ? "Nel tuo guardaroba"
                    : item.cost + " monetine"
              }}</small
              ><small v-if="!owns(item.key) && !itemAvailable(item, state)">{{
                state.level < item.level
                  ? "Sblocco al livello " + item.level
                  : item.availability === "event"
                    ? "Disponibile nella settimana di gara"
                    : "Torna nella settimana dedicata"
              }}</small></button
            ><button
              v-if="owns(item.key)"
              :disabled="busy || !me"
              @click="
                emit('command', {
                  action: equipped(item) ? 'remove' : 'equip',
                  key: item.key,
                })
              "
            >
              {{ equipped(item) ? "Rimuovi" : "Equipaggia" }}</button
            ><button
              v-else
              :disabled="
                busy ||
                !me ||
                state.wallet < item.cost ||
                !itemAvailable(item, state)
              "
              @click="emit('command', { action: 'buy', key: item.key })"
            >
              Acquista · {{ item.cost }}
            </button>
          </article>
        </div>
        <button
          v-if="state.equipment.celebration"
          class="primary"
          @click="emit('command', { action: 'celebrate' })"
        >
          Prova l’esultanza speciale
        </button>
      </template>
      <template v-if="active === 'predictions'">
        <template v-if="match"
          ><div class="match-card">
            <small>{{
              demo ? "PARTITA DIMOSTRATIVA" : "DAL CALENDARIO DELLA SOCIETÀ"
            }}</small>
            <h3>{{ match.home }}</h3>
            <span>vs</span>
            <h3>{{ match.away }}</h3>
            <p>{{ new Date(match.starts).toLocaleString("it-IT") }}</p>
          </div>
          <p>
            Risultato esatto in set.
            <strong>Il primo numero è sempre {{ match.home }}.</strong>
          </p>
          <p class="note">
            Esatto: {{ state.config.exact }} punti · Solo vincitore:
            {{ state.config.winner }} punti. Partecipazione gratuita, senza
            puntate.
          </p>
          <label
            >Il tuo pronostico<select
              v-model="choice"
              :disabled="!match.open || busy"
            >
              <option
                v-for="score in ['3-0', '3-1', '3-2', '2-3', '1-3', '0-3']"
                :key="score"
              >
                {{ score }}
              </option>
            </select></label
          ><button
            class="primary"
            :disabled="!me || busy || !match.open"
            @click="
              emit('command', {
                action: 'predict',
                event_id: match.id,
                score: choice,
              })
            "
          >
            {{ match.choice ? "Aggiorna" : "Conferma" }} pronostico
          </button>
          <p v-if="match.choice" class="success">
            Il tuo risultato: {{ match.choice }}
            {{ match.settled ? "· +" + match.reward + " punti" : "" }}
          </p>
          <p class="note">
            {{
              match.open
                ? "Modificabile fino al " +
                  new Date(match.cutoff).toLocaleString("it-IT")
                : "Pronostici chiusi"
            }}. Premi soltanto dopo conferma del risultato ufficiale.
          </p>
          <p v-if="match.score">Risultato confermato: {{ match.score }}</p>
          <div
            v-for="past in state.matches.filter(
              (m) => m.id !== match.id && m.choice,
            )"
            :key="past.id"
            class="history"
          >
            {{ past.home }} – {{ past.away }} · {{ past.choice }}
            <b>{{
              past.settled
                ? "+" + past.reward + " punti"
                : "In attesa del risultato"
            }}</b>
          </div>
        </template>
        <p v-else>
          Non ci sono partite in calendario. Torna quando la società pubblica il
          prossimo incontro.
        </p>
      </template>
      <template v-if="active === 'neighbors'">
        <p>
          Il tuo posto virtuale:
          <strong>{{ me?.seat.id || "Accedi per scoprirlo" }}</strong
          >. I posti dell’arena non corrispondono ai posti del biglietto.
        </p>
        <label class="privacy"
          ><input
            type="checkbox"
            :checked="state.member?.public"
            :disabled="busy || !me"
            @change="
              emit('command', {
                action: 'privacy',
                public: $event.target.checked,
              })
            "
          />Rendi visibili il mio nickname e la mia scheda nella ricerca
          pubblica</label
        >
        <div class="neighbors">
          <button
            v-for="actor in neighbors"
            :key="actor.id"
            :disabled="actor.public === false"
            @click="emit('neighbor', actor)"
          >
            <Icon name="fans" /><span
              >{{ actor.profile.nickname
              }}<small
                >{{ actor.seat.sector }} · fila {{ actor.seat.row }} · posto
                {{ actor.seat.number }}</small
              ></span
            >
          </button>
          <p v-if="!neighbors.length">Nessun vicino in questa zona, per ora.</p>
        </div>
        <button
          class="primary"
          :disabled="!me || busy"
          @click="emit('command', { action: 'invite' })"
        >
          Invita un amico vicino a te
        </button>
        <template v-if="invitation"
          ><label
            >Link da condividere<input
              readonly
              :value="invitation"
              aria-label="Link invito" /></label
          ><button @click="copy">Copia il link</button>
          <p role="status">{{ inviteMessage }}</p></template
        >
        <p class="note">
          L’amico accede con il proprio profilo e conferma la richiesta. Il
          sistema cerca un posto libero vicino a te o nello stesso settore. Gli
          altri tifosi mantengono il loro posto.
        </p>
      </template>
      <template v-if="active === 'museum'">
        <p>
          Le coppe nelle vetrine raccontano traguardi misurati della community.
          Tocca una vetrina nel 3D o esplora i dettagli.
        </p>
        <article
          v-for="(t, index) in state.trophies"
          :key="t.key"
          class="trophy-card"
          :class="{ unlocked: t.unlocked }"
        >
          <button @click="emit('focus', 'museum', index)">
            <Icon name="trophy" /><span
              ><strong>{{ t.title }}</strong
              ><small>{{ t.description }}</small
              ><b>{{
                t.unlocked
                  ? "Conquistato il " +
                    new Date(t.unlocked).toLocaleDateString("it-IT")
                  : "Da conquistare · obiettivo " + t.target
              }}</b></span
            >
          </button>
        </article>
        <button @click="emit('focus', 'general')">
          Torna alla panoramica dell’arena
        </button>
      </template>
      <DebugPanel
        v-if="demo && DebugPanel"
        :state="state"
        @debug="(action, value) => emit('debug', action, value)"
        @focus="(kind) => emit('focus', kind)"
      />
    </div>
  </section>
</template>
<style scoped>
.activity-sheet {
  position: absolute;
  right: 18px;
  top: 82px;
  width: 390px;
  max-width: calc(100% - 28px);
  max-height: calc(100dvh - 185px);
  background: #f7f9eff5;
  border: 1px solid #ffffffb3;
  border-radius: 22px;
  box-shadow: 0 15px 80px #14251f38;
  backdrop-filter: blur(18px);
  z-index: 8;
  color: #2d4432;
  font:
    13px/1.6 Arial,
    sans-serif;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.activity-sheet * {
  box-sizing: border-box;
}
.activity-sheet button,
.activity-sheet select,
.activity-sheet input {
  font: inherit;
  color: inherit;
}
.activity-sheet button {
  border: 0;
  cursor: pointer;
  border-radius: 9px;
  background: #e5edda;
  padding: 10px;
  min-height: 42px;
}
.activity-sheet button:disabled {
  opacity: 0.4;
  cursor: default;
}
.activity-sheet button:focus-visible,
.activity-sheet input:focus-visible {
  outline: 3px solid #718d43;
}
.activity-sheet header {
  display: flex;
  align-items: center;
  padding: 20px;
  gap: 10px;
  border-bottom: 1px solid #dce6cc;
}
.activity-sheet header > div {
  flex: 1;
}
.activity-sheet header small {
  font-size: 8px;
  letter-spacing: 1.2px;
  color: #82966a;
}
.activity-sheet header h2 {
  font-size: 22px;
  letter-spacing: -0.8px;
  margin: 5px 0 0;
}
.activity-sheet header > button {
  font-size: 21px;
  background: transparent;
}
.activity-body {
  padding: 18px 22px 24px;
  overflow: auto;
  overscroll-behavior: contain;
  scrollbar-width: thin;
}
.activity-body p {
  margin: 14px 0;
}
.activity-body h3 {
  font-size: 18px;
  margin: 16px 0 9px;
}
.activity-body svg {
  width: 22px;
  height: 22px;
  flex-shrink: 0;
}
.activity-wallet {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  padding-bottom: 15px;
  border-bottom: 1px solid #dce6cc;
  font-size: 11px;
}
.activity-wallet span {
  display: flex;
  align-items: center;
  gap: 5px;
}
.activity-wallet svg {
  width: 15px;
  height: 15px;
}
.activity-list > button {
  display: flex;
  align-items: center;
  gap: 13px;
  width: 100%;
  padding: 18px 0;
  background: transparent;
  text-align: left;
  border-radius: 0;
  border-bottom: 1px solid #dce6cc;
}
.activity-list > button > span:nth-child(2) {
  flex: 1;
}
.activity-list strong,
.activity-list small,
.activity-list b {
  display: block;
}
.activity-list strong {
  font-size: 15px;
}
.activity-list small {
  font-size: 11px;
  color: #7c8e6b;
  margin: 5px 0;
}
.activity-list b {
  font-size: 10px;
  font-weight: 500;
  color: #657f3f;
}
.activity-icon {
  padding: 11px;
  border-radius: 13px;
  background: #e4edce;
}
.note {
  font-size: 11px;
  color: #798b6a;
  line-height: 1.8;
}
.primary {
  background: #d3e7a2 !important;
  width: 100%;
  margin: 14px 0 4px;
  font-weight: 600 !important;
}
.hero-symbol {
  text-align: center;
  color: #b6a044;
  font-size: 54px;
}
blockquote {
  background: #ebf0db;
  margin: 18px 0;
  padding: 15px;
  border-radius: 12px;
  font-size: 13px;
}
.success {
  color: #55792e;
  background: #e9f0d7;
  padding: 12px;
  border-radius: 10px;
  display: block;
}
.featured {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}
.featured span {
  font-size: 10px;
  background: #eaf0dc;
  padding: 5px 9px;
  border-radius: 12px;
}
.categories {
  display: flex;
  gap: 5px;
  overflow: auto;
  margin: 17px 0;
}
.categories button {
  font-size: 10px;
  white-space: nowrap;
  background: transparent;
  border: 1px solid #d8e3c8;
  padding: 7px;
  min-height: 34px;
}
.categories [aria-pressed="true"] {
  background: #dce9bf;
}
.catalog {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}
.catalog article {
  border: 1px solid #dbe4ce;
  border-radius: 13px;
  padding: 7px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}
.catalog article.preview {
  border-color: #8baf50;
}
.catalog .item-preview {
  background: transparent;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 9px 3px;
}
.catalog strong {
  font-size: 12px;
}
.catalog small {
  font-size: 10px;
}
.catalog article > button:last-child {
  font-size: 11px;
  background: #e3edce;
}
.match-card {
  text-align: center;
  background: #e9efdc;
  padding: 18px;
  border-radius: 14px;
}
.match-card h3 {
  font-size: 18px;
}
.match-card small {
  font-size: 8px;
  letter-spacing: 1px;
}
.activity-body label {
  display: block;
  margin: 14px 0;
}
.activity-body select,
.activity-body input:not([type="checkbox"]) {
  width: 100%;
  min-height: 44px;
  border: 1px solid #c9d7b6;
  background: #fffef4;
  border-radius: 8px;
  padding: 9px;
  margin-top: 7px;
}
.privacy {
  display: flex !important;
  gap: 10px;
  font-size: 12px;
}
.privacy input {
  width: 20px;
  min-width: 20px;
}
.neighbors > button,
.trophy-card button {
  background: transparent;
  display: flex;
  gap: 12px;
  width: 100%;
  text-align: left;
  align-items: center;
}
.neighbors small,
.trophy-card small,
.trophy-card b {
  display: block;
  font-size: 10px;
  margin-top: 4px;
}
.trophy-card {
  padding: 8px 0;
  border-bottom: 1px solid #dce6cc;
  opacity: 0.65;
}
.trophy-card.unlocked {
  opacity: 1;
}
.trophy-card.unlocked svg {
  color: #ba9d39;
}
.history {
  padding: 12px 0;
  border-top: 1px solid #dce6cc;
  font-size: 11px;
}
.debug {
  margin-top: 24px;
  border-top: 1px dashed #b9c8a0;
  padding-top: 16px;
}
.debug summary {
  cursor: pointer;
  font-size: 11px;
}
.debug p {
  font-size: 10px;
}
.debug > div {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.debug button {
  font-size: 10px;
  flex: 1 0 44%;
}
.debug select {
  font-size: 12px;
}
@media (max-width: 700px) {
  .activity-sheet {
    inset: auto 12px max(12px, env(safe-area-inset-bottom));
    width: auto;
    max-width: none;
    max-height: 70dvh;
  }
  .activity-sheet.wardrobe-sheet {
    max-height: 49dvh;
  }
  .activity-sheet header {
    padding: 15px 18px;
  }
  .activity-sheet header h2 {
    font-size: 20px;
  }
  .activity-body {
    padding: 15px 19px 22px;
  }
  .activity-body select,
  .activity-body input:not([type="checkbox"]) {
    font-size: 16px;
  }
  .activity-wallet {
    font-size: 10px;
  }
}
</style>
