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
const stats = ref({ calls: 0, chairs: 36 }),
  activeTab = ref("fans"),
  photo = ref(false),
  marker = ref({ visible: false });
const now = ref(Date.now()),
  notice = ref(null),
  lastAction = ref("");
const photoUrl = import.meta.env.BASE_URL + "arena/reference.jpg";
const tabs = [
  { id: "fans", label: "Tifosi", icon: "fans" },
  { id: "views", label: "Viste", icon: "view" },
  { id: "scene", label: "Scena", icon: "cube" },
];
const currentView = computed(
  () =>
    presets.find((preset) => preset.id === view.value)?.label ||
    (view.value === "fan"
      ? "Il tuo posto"
      : tour.value
        ? "Reel orbit"
        : "Esplorazione libera"),
);
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
let arena,
  ticker,
  noticeTimer,
  actionTimer,
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
  }, 6000);
}
function revealScene() {
  const bounds = sceneFrame.value?.getBoundingClientRect();
  if (window.innerWidth < 801 || (bounds && bounds.bottom < 220))
    sceneFrame.value?.scrollIntoView({
      behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches
        ? "auto"
        : "smooth",
      block: "start",
    });
}
function selectFan(id) {
  if (session.select(id)) {
    refresh();
    arena?.selectFan(id);
    arena?.focusFan(id);
    revealScene();
  }
}
function findFan() {
  arena?.focusFan(state.value.selectedId);
  revealScene();
  notify("Ecco il posto di " + selectedFan.value.nickname + ".", "pin");
}
function choose(id) {
  arena?.view(id);
}
function play() {
  if (tour.value) arena?.stopTour();
  else arena?.startTour();
}
function showAction(kind) {
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
  refresh();
  arena?.fanAction(kind, result.fanId);
  showAction(kind);
  revealScene();
  notify(
    kind === "clap"
      ? "Il tuo applauso ha coinvolto i tifosi vicini."
      : kind === "scarf"
        ? "Sciarpe in alto, insieme al tuo settore."
        : "L’ola sta attraversando le tribune.",
    kind,
  );
}
function openVote() {
  if (!ready.value) return;
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
  revealScene();
  notify(
    "Hai votato l’MVP: " +
      DEMO_PLAYERS.find((player) => player.id === playerId).name +
      ".",
    "trophy",
  );
}
function resetDemo() {
  if (
    !window.confirm(
      "Ricominciare la demo? Saranno cancellati solo i voti e le interazioni demo salvati su questo dispositivo. I posti rimarranno gli stessi.",
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
  arena?.view("general");
  lastAction.value = "";
  notify("Arena demo ripristinata.");
}
function moveTab(event, index, delta) {
  const next = (index + delta + tabs.length) % tabs.length;
  activeTab.value = tabs[next].id;
  event.currentTarget.parentElement
    .querySelectorAll("[role=tab]")
    [next]?.focus();
}
watch(chairs, (value) => arena?.setChairs(value));
watch(roof, (value) => arena?.setRoof(value));
watch(railings, (value) => arena?.setRailings(value));
watch([fillers, onlyActive], () =>
  arena?.setCrowdFilters({
    fillers: fillers.value,
    onlyActive: onlyActive.value,
  }),
);
watch(photo, (value) => {
  if (value) {
    arena?.stopTour();
    photoDialog.value?.showModal();
  } else photoDialog.value?.close();
});
async function toggleFullscreen() {
  try {
    if (document.fullscreenElement) await document.exitFullscreen();
    else if (sceneFrame.value?.requestFullscreen)
      await sceneFrame.value.requestFullscreen();
    else portrait.value = !portrait.value;
  } catch {
    notify("Schermo intero non disponibile in questo browser.", "info");
  }
}
const syncFullscreen = () => {
  fullscreen.value = Boolean(document.fullscreenElement);
};
onMounted(async () => {
  document.addEventListener("fullscreenchange", syncFullscreen);
  ticker = setInterval(() => {
    now.value = Date.now();
  }, 250);
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
        },
        onStats: (value) => {
          stats.value = value;
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
      "Il browser non riesce ad avviare la scena WebGL. Prova un browser con accelerazione grafica attiva.";
  }
});
onBeforeUnmount(() => {
  alive = false;
  arena?.dispose();
  clearInterval(ticker);
  clearTimeout(noticeTimer);
  clearTimeout(actionTimer);
  document.removeEventListener("fullscreenchange", syncFullscreen);
});
</script>

<template>
  <div class="arena-lab arena-fans">
    <header class="lab-header">
      <a class="lab-brand" href="/arena-demo/" aria-label="Arena Tifosi, inizio"
        ><span class="brand-symbol"><Icon name="arena" /></span
        ><span
          >ARENA<span class="brand-light">BOOST</span><b>X</b
          ><small>SPATIAL EXPERIENCES</small></span
        ></a
      >
      <div class="header-divider"></div>
      <span class="header-label"
        >Joy Volley <span>/</span> <strong>Arena Tifosi</strong></span
      >
      <span class="prototype-badge"><i></i> ESPERIENZA DEMO</span
      ><a class="header-link" href="/">Torna ad ABX <Icon name="arrow" /></a>
    </header>
    <main class="lab-main">
      <section class="scene-column" aria-label="Arena Tifosi interattiva">
        <div class="scene-intro">
          <div>
            <p class="eyebrow">
              JOY VOLLEY · GIOIA DEL COLLE
              <span>LA COMMUNITY PRENDE POSTO</span>
            </p>
            <h1>La tua gente.<br /><em>La tua arena.</em></h1>
          </div>
          <span class="edition">02 <span>/ FAN EXPERIENCE</span></span>
        </div>
        <div
          ref="sceneFrame"
          class="scene-frame"
          :class="{ 'is-portrait': portrait, 'is-fullscreen': fullscreen }"
        >
          <div class="scene-topbar">
            <span class="scene-live"
              ><i :class="{ loading: !ready }"></i
              >{{ ready ? "ARENA TIFOSI" : "CARICAMENTO 3D" }}</span
            ><span class="scene-view-name">{{ currentView }}</span
            ><span class="scene-dimension">{{
              portrait ? "9:16" : "JOY VOLLEY"
            }}</span>
          </div>
          <div class="canvas-wrap">
            <div ref="viewport" class="arena-viewport"></div>
            <button
              v-if="ready && marker.visible"
              class="fan-world-label"
              :style="{ left: marker.x + '%', top: marker.y + '%' }"
              @click="findFan"
              :aria-label="'Il mio avatar: ' + selectedFan.nickname"
            >
              <i></i>{{ selectedFan.nickname }}<small>TU SEI QUI</small>
            </button>
          </div>
          <div v-if="error" class="scene-error" role="alert">
            <Icon name="info" />
            <p>{{ error }}</p>
            <button @click="reload">Ricarica demo</button>
          </div>
          <div v-else-if="!ready" class="scene-loading">
            <span></span>
            <p>La tua tribuna sta prendendo vita…</p>
          </div>
          <div class="scene-caption">
            <span class="caption-index"><Icon name="fans" /></span>
            <div>
              <strong>{{ visibleFans }} tifosi in arena</strong
              ><span>{{
                onlyActive
                  ? "Solo profili demo attivi"
                  : fillers
                    ? "60 profili demo · 280 comparse"
                    : "60 profili demo · comparse nascoste"
              }}</span>
            </div>
          </div>
          <div class="viewport-toolbar">
            <button
              :class="{ selected: portrait }"
              :aria-pressed="portrait"
              aria-label="Formato verticale 9:16"
              title="Formato verticale 9:16"
              @click="portrait = !portrait"
            >
              <Icon name="portrait" /></button
            ><button
              aria-label="Ripristina vista generale"
              title="Ripristina vista generale"
              @click="choose('general')"
            >
              <Icon name="reset" /></button
            ><span></span
            ><button
              :aria-label="
                fullscreen ? 'Esci da schermo intero' : 'Schermo intero'
              "
              title="Schermo intero"
              @click="toggleFullscreen"
            >
              <Icon name="expand" />
            </button>
          </div>
          <div v-if="tour" class="tour-overlay">
            <span><i></i> REEL ORBIT</span
            ><span
              >{{
                Math.min(TOUR_SECONDS, Math.floor(progress * TOUR_SECONDS))
                  .toString()
                  .padStart(2, "0")
              }}
              / {{ TOUR_SECONDS }} s</span
            >
            <div><i :style="{ width: progress * 100 + '%' }"></i></div>
            <button @click="play">Interrompi tour</button>
          </div>
          <Transition name="fan-notice"
            ><div v-if="notice" class="arena-notice" role="status">
              <span><Icon :name="notice.kind" /></span>
              <p>{{ notice.text }}</p>
              <button aria-label="Chiudi messaggio" @click="notice = null">
                <Icon name="close" />
              </button></div
          ></Transition>
        </div>
        <div class="scene-footer">
          <span><Icon name="mouse" /><b>Trascina</b> per ruotare</span
          ><span><Icon name="zoom" /><b>Scroll / pinch</b> per zoomare</span
          ><span class="footer-engine fan-active-legend"
            ><i></i> PROFILI DEMO ATTIVI</span
          >
        </div>
        <div class="arena-quick-actions" aria-label="Azioni rapide in arena">
          <button class="quick-mvp" :disabled="!ready" @click="openVote">
            <Icon name="trophy" /><span>Vota MVP</span></button
          ><button
            v-for="(action, kind) in FAN_ACTIONS"
            :key="kind"
            :disabled="!ready || cooldowns[kind] > 0"
            @click="act(kind)"
            :aria-label="action.label"
          >
            <Icon
              :name="
                kind === 'clap' ? 'clap' : kind === 'scarf' ? 'scarf' : 'wave'
              "
            /><span>{{
              kind === "clap"
                ? "Applaudi"
                : kind === "scarf"
                  ? "Alza la sciarpa"
                  : "Coreografia / Ola"
            }}</span
            ><small v-if="cooldowns[kind]">{{ cooldowns[kind] }}s</small>
          </button>
        </div>
        <div class="study-note">
          <span>UN POSTO TUTTO TUO.<br />UNA TRIBUNA DA VIVERE.</span>
          <p>
            I punti verdi indicano i profili demo attivi. Trova il tuo avatar,
            scegli il tuo MVP e coinvolgi i tifosi intorno a te.
          </p>
        </div>
      </section>
      <aside class="control-panel" aria-label="Pannello Arena Tifosi">
        <div class="panel-heading">
          <span class="eyebrow">BENVENUTO SUGLI SPALTI</span
          ><Icon name="fans" />
          <h2>Arena Tifosi<span>Joy Volley · Gioia del Colle</span></h2>
        </div>
        <div class="fan-presence">
          <div>
            <strong>{{ visibleFans }}</strong
            ><span>in arena</span>
          </div>
          <div>
            <strong><i></i>{{ activeFans }}</strong
            ><span>profili attivi demo</span>
          </div>
        </div>
        <div
          class="panel-tabs"
          role="tablist"
          aria-label="Arena Tifosi e impostazioni"
        >
          <button
            v-for="(tab, index) in tabs"
            :key="tab.id"
            :id="tab.id + '-tab'"
            role="tab"
            :aria-selected="activeTab === tab.id"
            :aria-controls="tab.id + '-panel'"
            :tabindex="activeTab === tab.id ? 0 : -1"
            @click="activeTab = tab.id"
            @keydown.right.prevent="moveTab($event, index, 1)"
            @keydown.left.prevent="moveTab($event, index, -1)"
          >
            <Icon :name="tab.icon" />{{ tab.label }}
          </button>
        </div>
        <div
          v-show="activeTab === 'fans'"
          id="fans-panel"
          role="tabpanel"
          aria-labelledby="fans-tab"
        >
          <FanPanel
            :profiles="state.profiles"
            :fan="selectedFan"
            :ready="ready"
            :cooldowns="cooldowns"
            :voted="Boolean(votedPlayer)"
            :last-action="lastAction"
            @select="selectFan"
            @find="findFan"
            @action="act"
            @vote="openVote"
          />
          <p v-if="!state.persistent" class="storage-warning" role="status">
            Salvataggio locale non disponibile: i dati resteranno in questa
            sessione.
          </p>
        </div>
        <div
          v-show="activeTab === 'views'"
          id="views-panel"
          class="views-panel"
          role="tabpanel"
          aria-labelledby="views-tab"
        >
          <button
            v-for="(preset, index) in presets"
            :key="preset.id"
            :class="{ active: view === preset.id }"
            :aria-pressed="view === preset.id"
            :disabled="!ready"
            @click="choose(preset.id)"
          >
            <span class="view-number">0{{ index + 1 }}</span
            ><span
              ><strong>{{ preset.label }}</strong
              ><small>{{ preset.subtitle }}</small></span
            ><Icon name="chevron" /></button
          ><button class="find-view" :disabled="!ready" @click="findFan">
            <span class="view-number"><Icon name="pin" /></span
            ><span
              ><strong>Il mio avatar</strong
              ><small
                >{{ selectedFan.nickname }} · {{ selectedFan.seatId }}</small
              ></span
            ><Icon name="chevron" />
          </button>
          <div class="reel-card">
            <div>
              <span class="eyebrow">LA TRIBUNA SI RACCONTA</span
              ><span class="reel-duration">{{ TOUR_SECONDS }} SEC</span>
            </div>
            <h3>Un giro nell’Arena Tifosi.</h3>
            <p>Una sequenza tra campo, tifosi e tribune.</p>
            <button class="tour-button" :disabled="!ready" @click="play">
              <Icon :name="tour ? 'stop' : 'play'" /><span>{{
                tour ? "Interrompi tour" : "Avvia tour cinematico"
              }}</span
              ><Icon name="arrow" /></button
            ><span class="reel-hint"
              >Disponibile anche con inquadratura 9:16.</span
            >
          </div>
        </div>
        <div
          v-show="activeTab === 'scene'"
          id="scene-panel"
          class="settings-panel"
          role="tabpanel"
          aria-labelledby="scene-tab"
        >
          <p class="setting-hint">
            Scegli chi vedere sugli spalti e i dettagli del palazzetto.
          </p>
          <label
            ><span
              ><strong>Comparse sugli spalti</strong
              ><small>280 avatar di riempimento</small></span
            ><input type="checkbox" v-model="fillers" role="switch" /><i></i
          ></label>
          <label
            ><span
              ><strong>Solo profili attivi</strong
              ><small>Mostra i tifosi con il punto verde</small></span
            ><input type="checkbox" v-model="onlyActive" role="switch" /><i></i
          ></label>
          <label
            ><span
              ><strong>Sedie sul parquet</strong
              ><small
                >{{ stats.chairs }} sedie della foto originale</small
              ></span
            ><input type="checkbox" v-model="chairs" role="switch" /><i></i
          ></label>
          <label
            ><span
              ><strong>Tetto in legno</strong
              ><small>Travi e copertura a falde</small></span
            ><input type="checkbox" v-model="roof" role="switch" /><i></i
          ></label>
          <label
            ><span
              ><strong>Barriere del campo</strong
              ><small>Ringhiere perimetrali blu</small></span
            ><input type="checkbox" v-model="railings" role="switch" /><i></i
          ></label>
          <button class="reference-card" @click="photo = true">
            <img
              :src="photoUrl"
              alt="Foto originale del palazzetto di Gioia del Colle"
            /><span
              ><Icon name="photo" />La foto di partenza<Icon name="expand"
            /></span>
          </button>
          <div class="cutaway-note">
            <Icon name="info" />
            <p>
              Le superfici esterne si aprono alla vista per lasciarti esplorare
              l’interno.
            </p>
          </div>
          <button class="reset-fan-demo" @click="resetDemo">
            <Icon name="reset" />Reset dati demo
          </button>
        </div>
        <div class="panel-footnote">
          <span class="small-cross">+</span>Profili e giocatori dimostrativi. Un
          primo passo verso la community ABX.
        </div>
      </aside>
    </main>
    <footer class="lab-footer">
      <span>ARENA TIFOSI <b>—</b> IL TUO POSTO NELLA COMMUNITY.</span
      ><span>Esperienza demo locale <i>© ABX 2026</i></span>
    </footer>
    <MvpDialog
      ref="mvpDialog"
      :fan="selectedFan"
      :voted-player="votedPlayer"
      @vote="vote"
      @open="arena?.stopTour()"
    />
    <dialog
      ref="photoDialog"
      class="photo-dialog"
      aria-label="Foto del palazzetto"
      @close="photo = false"
      @click="
        (event) => {
          if (event.target === photoDialog) photo = false;
        }
      "
    >
      <div>
        <button aria-label="Chiudi foto" @click="photo = false">
          <Icon name="close" /></button
        ><img
          :src="photoUrl"
          alt="Foto di riferimento del palazzetto Joy Volley"
        />
        <p>
          <span>01 / LA FOTO DI PARTENZA</span>La geometria del palazzetto è
          ispirata a questa fotografia e alle indicazioni ricevute.
        </p>
      </div>
    </dialog>
  </div>
</template>
