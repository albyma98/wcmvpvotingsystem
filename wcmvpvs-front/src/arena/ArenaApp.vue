<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import Icon from "./ArenaIcon.vue";
import FanPanel from "./ui/FanPanel.vue";
import MvpDialog from "./ui/MvpDialog.vue";
import { createArena } from "./scene/createArena.js";
import { presets, TOUR_SECONDS } from "./camera/cameraSystem.js";
import { createDemoSession, FAN_ACTIONS } from "./actions/demoSession.js";
import { DEMO_PLAYERS } from "./mvp/players.js";

let storage;
try {
  storage = window.localStorage;
} catch {}
const session = createDemoSession(storage);
const state = ref(session.snapshot());
const selectedFan = computed(() =>
  state.value.profiles.find((profile) => profile.id === state.value.selectedId),
);
const activeFans = computed(
  () => state.value.profiles.filter((profile) => profile.active).length,
);
const votedPlayer = computed(
  () =>
    DEMO_PLAYERS.find(
      (player) => player.id === state.value.votes[state.value.selectedId],
    ) || null,
);
const initials = computed(
  () =>
    selectedFan.value.nickname[0] +
    (selectedFan.value.nickname.match(/[A-Z]/g)?.[1] ||
      selectedFan.value.nickname[1]),
);
const fillers = ref(true),
  onlyActive = ref(false);
const visibleFans = computed(
  () =>
    state.value.profiles.filter(
      (profile) => !onlyActive.value || profile.active,
    ).length + (fillers.value && !onlyActive.value ? 280 : 0),
);
const viewport = ref(null),
  sceneFrame = ref(null),
  profileDialog = ref(null),
  prefsDialog = ref(null),
  photoDialog = ref(null),
  mvpDialog = ref(null);
const ready = ref(false),
  error = ref(""),
  chairs = ref(false),
  roof = ref(true),
  railings = ref(true);
const tour = ref(false),
  progress = ref(0),
  view = ref("general"),
  portrait = ref(false),
  fullscreen = ref(false);
const activePanel = ref(""),
  prefsTab = ref("views"),
  marker = ref({ visible: false }),
  hint = ref(true);
const now = ref(Date.now()),
  notice = ref(null),
  lastAction = ref("");
const photoUrl = import.meta.env.BASE_URL + "arena/reference.jpg";
const cooldowns = computed(() => {
  now.value;
  state.value;
  return Object.fromEntries(
    Object.keys(FAN_ACTIONS).map((kind) => [
      kind,
      Math.ceil(session.remaining(kind, now.value) / 1000),
    ]),
  );
});
const actions = [
  { id: "clap", label: "Applauso", icon: "clap", key: "2" },
  { id: "scarf", label: "Sciarpa", icon: "scarf", key: "3" },
  { id: "wave", label: "Ola", icon: "wave", key: "4" },
];
let arena,
  ticker,
  noticeTimer,
  actionTimer,
  hintTimer,
  alive = true;

function refresh() {
  now.value = Date.now();
  state.value = session.snapshot();
}
function reload() {
  window.location.reload();
}
function notify(text, kind = "check") {
  clearTimeout(noticeTimer);
  notice.value = { text, kind };
  noticeTimer = setTimeout(() => {
    notice.value = null;
  }, 4200);
}
function closePanels() {
  for (const dialog of [
    profileDialog.value,
    prefsDialog.value,
    photoDialog.value,
  ])
    if (dialog?.open) dialog.close();
  activePanel.value = "";
}
function panelClosed(kind) {
  if (activePanel.value === kind) activePanel.value = "";
}
function openPanel(kind) {
  closePanels();
  arena?.stopTour();
  hint.value = false;
  activePanel.value = kind;
  (kind === "profile" ? profileDialog.value : prefsDialog.value)?.showModal();
}
function backdrop(event, kind) {
  if (event.target === event.currentTarget) {
    event.currentTarget.close();
    panelClosed(kind);
  }
}
function selectFan(id) {
  if (session.select(id)) {
    refresh();
    closePanels();
    hint.value = false;
    arena?.selectFan(id);
    arena?.focusFan(id);
  }
}
function findFan() {
  closePanels();
  hint.value = false;
  arena?.focusFan(state.value.selectedId);
}
function choose(id) {
  closePanels();
  hint.value = false;
  arena?.view(id);
}
function play() {
  closePanels();
  hint.value = false;
  if (tour.value) arena?.stopTour();
  else arena?.startTour();
}
function showAction(kind) {
  hint.value = false;
  lastAction.value = kind;
  clearTimeout(actionTimer);
  actionTimer = setTimeout(() => {
    lastAction.value = "";
  }, FAN_ACTIONS[kind]?.duration || 5000);
}
function act(kind) {
  if (!ready.value) return;
  const result = session.action(kind);
  if (!result.ok) {
    notify(result.reason, "info");
    return;
  }
  closePanels();
  refresh();
  arena?.fanAction(kind, result.fanId);
  showAction(kind);
  notify(
    kind === "clap"
      ? "Il tuo settore applaude con te."
      : kind === "scarf"
        ? "Sciarpe in alto!"
        : "L’ola è partita.",
    kind,
  );
}
function openVote() {
  if (!ready.value) return;
  closePanels();
  hint.value = false;
  arena?.focusFan(state.value.selectedId);
  mvpDialog.value?.open();
}
function vote(playerId) {
  const result = session.vote(playerId);
  if (!result.ok) {
    notify(result.reason, "info");
    return;
  }
  refresh();
  arena?.fanAction("vote", result.fanId);
  showAction("vote");
  notify(
    "Hai votato " +
      DEMO_PLAYERS.find((player) => player.id === playerId).name +
      ".",
    "trophy",
  );
}
function resetDemo() {
  if (
    !window.confirm(
      "Ricominciare la demo? Saranno cancellati solo i voti e le interazioni demo su questo dispositivo. I posti rimarranno gli stessi.",
    )
  )
    return;
  session.reset();
  refresh();
  fillers.value = true;
  onlyActive.value = false;
  chairs.value = false;
  arena?.resetCrowd();
  arena?.selectFan(state.value.selectedId);
  choose("general");
  lastAction.value = "";
  notify("Arena demo ripristinata.");
}
function openPhoto() {
  closePanels();
  photoDialog.value?.showModal();
}
function showHelp() {
  closePanels();
  hint.value = true;
  clearTimeout(hintTimer);
  hintTimer = setTimeout(() => {
    hint.value = false;
  }, 9000);
}
function moveTab(event) {
  prefsTab.value = prefsTab.value === "views" ? "scene" : "views";
  const buttons =
    event.currentTarget.parentElement.querySelectorAll("[role=tab]");
  buttons[prefsTab.value === "views" ? 0 : 1]?.focus();
}
async function toggleFullscreen() {
  closePanels();
  try {
    if (document.fullscreenElement) await document.exitFullscreen();
    else if (sceneFrame.value?.requestFullscreen)
      await sceneFrame.value.requestFullscreen();
    else notify("La scena occupa già tutto lo spazio disponibile.", "info");
  } catch {
    notify("Schermo intero non disponibile in questo browser.", "info");
  }
}
function shortcut(event) {
  if (
    event.repeat ||
    event.ctrlKey ||
    event.metaKey ||
    event.altKey ||
    document.querySelector("dialog[open]") ||
    event.target?.closest?.("input,select,textarea,[contenteditable=true]")
  )
    return;
  const key = event.key.toLowerCase(),
    commands = {
      1: openVote,
      2: () => act("clap"),
      3: () => act("scarf"),
      4: () => act("wave"),
      f: findFan,
    };
  if (ready.value && commands[key]) {
    event.preventDefault();
    commands[key]();
  }
}
const syncFullscreen = () => {
  fullscreen.value = Boolean(document.fullscreenElement);
};
watch(chairs, (value) => arena?.setChairs(value));
watch(roof, (value) => arena?.setRoof(value));
watch(railings, (value) => arena?.setRailings(value));
watch([fillers, onlyActive], () =>
  arena?.setCrowdFilters({
    fillers: fillers.value,
    onlyActive: onlyActive.value,
  }),
);
onMounted(async () => {
  document.addEventListener("fullscreenchange", syncFullscreen);
  window.addEventListener("keydown", shortcut);
  ticker = setInterval(() => {
    now.value = Date.now();
  }, 250);
  hintTimer = setTimeout(() => {
    hint.value = false;
  }, 9000);
  await new Promise((resolve) => requestAnimationFrame(resolve));
  if (!alive) return;
  try {
    arena = createArena(
      viewport.value,
      {
        onTour: (running, value) => {
          tour.value = running;
          progress.value = value;
        },
        onView: (value) => {
          view.value = value;
          if (value === "custom") hint.value = false;
        },
        onError: (value) => {
          error.value = value;
          ready.value = false;
        },
        onSelectProfile: selectFan,
        onMarker: (value) => {
          marker.value = value;
        },
      },
      { actors: session.actors, selectedId: state.value.selectedId },
    );
    arena.setChairs(chairs.value);
    arena.setRoof(roof.value);
    arena.setRailings(railings.value);
    ready.value = true;
  } catch {
    error.value =
      "Non riesco ad avviare la scena 3D. Riprova con l’accelerazione grafica attiva nel browser.";
  }
});
onBeforeUnmount(() => {
  alive = false;
  arena?.dispose();
  clearInterval(ticker);
  clearTimeout(noticeTimer);
  clearTimeout(actionTimer);
  clearTimeout(hintTimer);
  document.removeEventListener("fullscreenchange", syncFullscreen);
  window.removeEventListener("keydown", shortcut);
});
</script>

<template>
  <main
    ref="sceneFrame"
    class="arena-play"
    :class="{ 'is-portrait': portrait, 'is-fullscreen': fullscreen }"
    aria-label="Arena Tifosi a tutto schermo"
  >
    <div class="play-canvas">
      <div ref="viewport" class="arena-viewport"></div>
      <button
        v-if="ready && marker.visible"
        class="fan-world-label play-world-label"
        :style="{ left: marker.x + '%', top: marker.y + '%' }"
        @click="findFan"
        :aria-label="'Il mio avatar: ' + selectedFan.nickname"
      >
        <i></i>{{ selectedFan.nickname }}<small>TU SEI QUI</small>
      </button>
    </div>
    <div class="play-vignette" aria-hidden="true"></div>
    <header class="play-top">
      <div class="play-brand">
        <span class="play-brand-mark"><Icon name="arena" /></span>
        <div>
          <small>JOY VOLLEY <i>·</i> DEMO</small>
          <h1>Arena Tifosi</h1>
        </div>
      </div>
      <div class="play-top-tools">
        <div
          class="play-population"
          :aria-label="
            visibleFans +
            ' tifosi in arena, ' +
            activeFans +
            ' profili demo attivi'
          "
          title="Presenze demo in arena e profili attivi"
        >
          <Icon name="fans" /><strong>{{ visibleFans }}</strong
          ><span></span><i></i><b>{{ activeFans }}</b>
        </div>
        <button
          class="play-icon-button"
          :class="{ selected: activePanel === 'prefs' }"
          aria-label="Apri impostazioni arena"
          title="Viste e impostazioni"
          :aria-expanded="activePanel === 'prefs'"
          aria-controls="play-preferences"
          @click="openPanel('prefs')"
        >
          <Icon name="settings" />
        </button>
      </div>
    </header>
    <Transition name="play-toast"
      ><div v-if="notice" class="play-notice" role="status">
        <Icon :name="notice.kind" />
        <p>{{ notice.text }}</p>
        <button aria-label="Chiudi messaggio" @click="notice = null">
          <Icon name="close" />
        </button></div
    ></Transition>
    <div class="play-bottom">
      <button
        class="play-identity"
        aria-label="Cambia profilo tifoso"
        :aria-expanded="activePanel === 'profile'"
        aria-controls="play-profile"
        @click="openPanel('profile')"
      >
        <span class="play-avatar">{{ initials }}<i></i></span
        ><span
          ><strong>{{ selectedFan.nickname }}</strong
          ><small
            >{{ selectedFan.seatId }} <i>·</i> L{{ selectedFan.level }}</small
          ></span
        ><Icon name="chevron" />
      </button>
      <div class="play-action-zone">
        <Transition name="play-toast"
          ><p v-if="hint && ready && !tour" class="play-hint">
            <span>Trascina per esplorare · Tocca un tifoso</span
            ><small
              >Desktop: 1 MVP · 2 Applauso · 3 Sciarpa · 4 Ola · F Posto</small
            >
          </p></Transition
        >
        <nav v-if="!tour" class="play-actions" aria-label="Azioni del tifoso">
          <button
            class="play-action play-mvp"
            :class="{ reacting: lastAction === 'vote' }"
            :disabled="!ready"
            aria-label="Vota MVP"
            title="Vota MVP · 1"
            @click="openVote"
          >
            <Icon name="trophy" /><span>MVP</span
            ><i v-if="votedPlayer" class="play-voted"><Icon name="check" /></i>
          </button>
          <button
            v-for="action in actions"
            :key="action.id"
            class="play-action"
            :class="{ reacting: lastAction === action.id }"
            :disabled="!ready || cooldowns[action.id] > 0"
            :aria-label="FAN_ACTIONS[action.id].label"
            :aria-description="
              cooldowns[action.id]
                ? 'Disponibile tra ' + cooldowns[action.id] + ' secondi'
                : undefined
            "
            :title="FAN_ACTIONS[action.id].label + ' · ' + action.key"
            @click="act(action.id)"
          >
            <Icon :name="action.icon" /><span>{{ action.label }}</span
            ><b v-if="cooldowns[action.id]" class="play-cooldown">{{
              cooldowns[action.id]
            }}</b>
          </button>
        </nav>
        <div v-else class="play-tour">
          <Icon name="view" />
          <div>
            <strong>Un giro in tribuna</strong
            ><span
              >{{
                Math.min(TOUR_SECONDS, Math.floor(progress * TOUR_SECONDS))
                  .toString()
                  .padStart(2, "0")
              }}
              / {{ TOUR_SECONDS }} s</span
            ><i><b :style="{ width: progress * 100 + '%' }"></b></i>
          </div>
          <button
            aria-label="Interrompi tour"
            title="Interrompi tour"
            @click="play"
          >
            <Icon name="stop" />
          </button>
        </div>
      </div>
      <button
        class="play-find play-icon-button"
        :disabled="!ready"
        aria-label="Trova il mio posto"
        title="Trova il mio posto · F"
        @click="findFan"
      >
        <Icon name="pin" /><span>Il mio posto</span>
      </button>
    </div>
    <div v-if="error" class="play-loading play-error" role="alert">
      <Icon name="info" />
      <h2>La tribuna ti aspetta.</h2>
      <p>{{ error }}</p>
      <button @click="reload">Riprova</button><a href="/">Torna ad ABX</a>
    </div>
    <div v-else-if="!ready" class="play-loading">
      <span class="play-loader"><Icon name="arena" /></span>
      <h2>Entra in tribuna.</h2>
      <p>Il tuo posto sta prendendo vita…</p>
    </div>

    <dialog
      ref="profileDialog"
      id="play-profile"
      class="play-sheet play-profile-sheet"
      aria-labelledby="play-profile-title"
      @close="panelClosed('profile')"
      @click="backdrop($event, 'profile')"
    >
      <div class="play-sheet-heading">
        <div>
          <p>IL TUO POSTO NELLA COMMUNITY</p>
          <h2 id="play-profile-title">Il tuo tifoso</h2>
        </div>
        <button aria-label="Chiudi profilo" @click="profileDialog.close()">
          <Icon name="close" />
        </button>
      </div>
      <div class="play-sheet-body">
        <FanPanel
          :profiles="state.profiles"
          :fan="selectedFan"
          :ready="ready"
          @select="selectFan"
          @find="findFan"
        />
        <p v-if="!state.persistent" class="storage-warning" role="status">
          Salvataggio locale non disponibile: i dati resteranno in questa
          sessione.
        </p>
      </div>
    </dialog>

    <dialog
      ref="prefsDialog"
      id="play-preferences"
      class="play-sheet"
      aria-labelledby="play-preferences-title"
      @close="panelClosed('prefs')"
      @click="backdrop($event, 'prefs')"
    >
      <div class="play-sheet-heading">
        <div>
          <p>JOY VOLLEY · ARENA TIFOSI</p>
          <h2 id="play-preferences-title">La tua prospettiva</h2>
        </div>
        <button aria-label="Chiudi impostazioni" @click="prefsDialog.close()">
          <Icon name="close" />
        </button>
      </div>
      <div class="play-sheet-body">
        <div
          class="play-tabs"
          role="tablist"
          aria-label="Viste e dettagli arena"
        >
          <button
            id="play-views-tab"
            role="tab"
            :aria-selected="prefsTab === 'views'"
            aria-controls="play-views-panel"
            :tabindex="prefsTab === 'views' ? 0 : -1"
            @click="prefsTab = 'views'"
            @keydown.right.prevent="moveTab($event)"
            @keydown.left.prevent="moveTab($event)"
          >
            <Icon name="view" />Viste</button
          ><button
            id="play-scene-tab"
            role="tab"
            :aria-selected="prefsTab === 'scene'"
            aria-controls="play-scene-panel"
            :tabindex="prefsTab === 'scene' ? 0 : -1"
            @click="prefsTab = 'scene'"
            @keydown.right.prevent="moveTab($event)"
            @keydown.left.prevent="moveTab($event)"
          >
            <Icon name="cube" />Scena
          </button>
        </div>
        <div
          v-show="prefsTab === 'views'"
          id="play-views-panel"
          role="tabpanel"
          aria-labelledby="play-views-tab"
          class="play-view-list"
        >
          <button
            v-for="(preset, index) in presets"
            :key="preset.id"
            :aria-pressed="view === preset.id"
            :disabled="!ready"
            @click="choose(preset.id)"
          >
            <small>0{{ index + 1 }}</small
            ><span>{{ preset.label }}</span
            ><Icon name="chevron" /></button
          ><button class="play-tour-launch" :disabled="!ready" @click="play">
            <Icon name="play" /><span>Avvia tour cinematico</span
            ><small>18s</small>
          </button>
        </div>
        <div
          v-show="prefsTab === 'scene'"
          id="play-scene-panel"
          role="tabpanel"
          aria-labelledby="play-scene-tab"
          class="settings-panel play-settings"
        >
          <label
            ><span
              ><strong>Comparse sugli spalti</strong
              ><small>280 tifosi di riempimento</small></span
            ><input v-model="fillers" type="checkbox" role="switch" /><i></i
          ></label>
          <label
            ><span
              ><strong>Solo profili attivi</strong
              ><small>Mostra i tifosi con il punto verde</small></span
            ><input v-model="onlyActive" type="checkbox" role="switch" /><i></i
          ></label>
          <label
            ><span
              ><strong>Tetto in legno</strong
              ><small>Copertura e travi del palazzetto</small></span
            ><input v-model="roof" type="checkbox" role="switch" /><i></i
          ></label>
          <label
            ><span
              ><strong>Barriere del campo</strong
              ><small>Ringhiere perimetrali blu</small></span
            ><input v-model="railings" type="checkbox" role="switch" /><i></i
          ></label>
          <label
            ><span
              ><strong>Sedie sul parquet</strong
              ><small>Allestimento della fotografia</small></span
            ><input v-model="chairs" type="checkbox" role="switch" /><i></i
          ></label>
        </div>
        <div class="play-utilities">
          <button :aria-pressed="portrait" @click="portrait = !portrait">
            <Icon name="portrait" />Formato 9:16</button
          ><button @click="toggleFullscreen">
            <Icon name="expand" />{{
              fullscreen ? "Esci da schermo intero" : "Schermo intero"
            }}</button
          ><button @click="showHelp"><Icon name="info" />Come giocare</button
          ><button @click="openPhoto">
            <Icon name="photo" />Foto originale
          </button>
        </div>
        <div class="play-menu-footer">
          <a href="/">Torna ad ABX<Icon name="arrow" /></a
          ><button
            @click="resetDemo"
            title="Cancella soltanto i dati della demo"
          >
            Reset demo
          </button>
        </div>
      </div>
    </dialog>
    <MvpDialog
      ref="mvpDialog"
      :fan="selectedFan"
      :voted-player="votedPlayer"
      :persistent="state.persistent"
      @vote="vote"
    />
    <dialog
      ref="photoDialog"
      class="photo-dialog"
      aria-label="Foto del palazzetto"
      @click="backdrop($event, 'photo')"
    >
      <div>
        <button aria-label="Chiudi foto" @click="photoDialog.close()">
          <Icon name="close" /></button
        ><img
          :src="photoUrl"
          alt="Fotografia del palazzetto Joy Volley a Gioia del Colle"
        />
        <p>
          <span>LA FOTO DI PARTENZA</span>La geometria del palazzetto è ispirata
          a questa fotografia e alle indicazioni ricevute.
        </p>
      </div>
    </dialog>
  </main>
</template>
