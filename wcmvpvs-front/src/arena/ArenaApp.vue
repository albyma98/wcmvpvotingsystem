<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import Icon from "./ArenaIcon.vue";
import { createArena } from "./scene/createArena";
import { presets, TOUR_SECONDS } from "./camera/cameraSystem";

const viewport = ref(null),
  sceneFrame = ref(null),
  photoDialog = ref(null);
const ready = ref(false),
  error = ref(""),
  chairs = ref(true),
  roof = ref(true),
  railings = ref(true);
const tour = ref(false),
  progress = ref(0),
  view = ref("general"),
  portrait = ref(false);
const stats = ref({ calls: 0, chairs: 36 }),
  activeTab = ref("views"),
  photo = ref(false),
  fullscreen = ref(false);
const photoUrl = `${import.meta.env.BASE_URL}arena/reference.jpg`;
const currentView = computed(
  () =>
    presets.find((item) => item.id === view.value)?.label ||
    (tour.value ? "Reel orbit" : "Esplorazione libera"),
);
let arena;
watch(chairs, (value) => arena?.setChairs(value));
watch(roof, (value) => arena?.setRoof(value));
watch(railings, (value) => arena?.setRailings(value));
watch(photo, (value) => {
  if (value) {
    arena?.stopTour();
    photoDialog.value?.showModal();
  } else photoDialog.value?.close();
});
function choose(id) {
  arena?.view(id);
}
function play() {
  if (tour.value) arena?.stopTour();
  else arena?.startTour();
}
function changeMode(value) {
  chairs.value = value;
}
function reload() {
  window.location.reload();
}
async function toggleFullscreen() {
  try {
    if (document.fullscreenElement) await document.exitFullscreen();
    else if (sceneFrame.value?.requestFullscreen)
      await sceneFrame.value.requestFullscreen();
    else portrait.value = !portrait.value;
  } catch {
    error.value =
      "Schermo intero non disponibile in questo browser. Puoi continuare a esplorare la scena.";
  }
}
const syncFullscreen = () => {
  fullscreen.value = Boolean(document.fullscreenElement);
};
onMounted(async () => {
  document.addEventListener("fullscreenchange", syncFullscreen);
  // Let the shell paint before constructing the WebGL scene.
  await new Promise((resolve) => requestAnimationFrame(resolve));
  try {
    arena = createArena(viewport.value, {
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
    });
    ready.value = true;
  } catch {
    error.value =
      "Il browser non riesce ad avviare WebGL. Prova un browser con accelerazione grafica attiva.";
  }
});
onBeforeUnmount(() => {
  arena?.dispose();
  document.removeEventListener("fullscreenchange", syncFullscreen);
});
</script>

<template>
  <div class="arena-lab">
    <header class="lab-header">
      <a class="lab-brand" href="/arena-demo/" aria-label="Arena Lab, inizio"
        ><span class="brand-symbol"><Icon name="arena" /></span
        ><span
          >ARENA<span class="brand-light">BOOST</span><b>X</b
          ><small>SPATIAL EXPERIENCES</small></span
        ></a
      >
      <div class="header-divider"></div>
      <span class="header-label"
        >Arena Lab <span>/</span> <strong>Joy Volley</strong></span
      >
      <span class="prototype-badge"><i></i> PROTOTIPO INTERATTIVO</span>
      <a class="header-link" href="/">Torna ad ABX <Icon name="arrow" /></a>
    </header>

    <main class="lab-main">
      <section class="scene-column" aria-label="Esplora il palazzetto">
        <div class="scene-intro">
          <div>
            <p class="eyebrow">
              GIOIA DEL COLLE, PUGLIA <span>41° N · 16° E</span>
            </p>
            <h1>Il tuo palazzetto.<br /><em>Una nuova prospettiva.</em></h1>
          </div>
          <span class="edition">01 <span>/ ARENA STUDIES</span></span>
        </div>

        <div
          ref="sceneFrame"
          class="scene-frame"
          :class="{ 'is-portrait': portrait, 'is-fullscreen': fullscreen }"
        >
          <div class="scene-topbar">
            <span class="scene-live"
              ><i :class="{ loading: !ready }"></i
              >{{ ready ? "SCENA LIVE" : "CARICAMENTO 3D" }}</span
            ><span class="scene-view-name">{{ currentView }}</span
            ><span class="scene-dimension">{{
              portrait ? "9:16" : "3D VIEW"
            }}</span>
          </div>
          <div class="canvas-wrap">
            <div ref="viewport" class="arena-viewport"></div>
          </div>
          <div v-if="error" class="scene-error" role="alert">
            <Icon name="info" />
            <p>{{ error }}</p>
            <button @click="reload">Ricarica demo</button>
          </div>
          <div v-else-if="!ready" class="scene-loading">
            <span></span>
            <p>Entriamo nel palazzetto…</p>
          </div>
          <div class="scene-caption">
            <span class="caption-index">01—</span>
            <div>
              <strong>Joy Volley Arena</strong
              ><span>{{
                chairs
                  ? "Allestimento della foto · sedie sul parquet"
                  : "Pronta per l’evento · campo libero"
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
              title="Ripristina vista"
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
            <div><i :style="{ width: `${progress * 100}%` }"></i></div>
            <button @click="play">Interrompi tour</button>
          </div>
        </div>
        <div class="scene-footer">
          <span><Icon name="mouse" /><b>Trascina</b> per ruotare</span
          ><span><Icon name="zoom" /><b>Scroll / pinch</b> per zoomare</span
          ><span class="footer-engine"
            ><i></i> WEBGL · {{ stats.calls }} DRAW CALLS</span
          >
        </div>
        <div class="study-note">
          <span>UNA FOTOGRAFIA, UNO SPAZIO DA ESPLORARE.</span>
          <p>
            Il legno del soffitto, il giallo delle sedute, il blu delle
            ringhiere. Un piccolo studio digitale sull’identità di un luogo.
          </p>
        </div>
      </section>

      <aside class="control-panel" aria-label="Controlli della scena">
        <div class="panel-heading">
          <span class="eyebrow">IL PALAZZETTO, IN 3D</span><Icon name="cube" />
          <h2>Joy Volley<span>Gioia del Colle</span></h2>
          <p>Riconosci i dettagli.<br />Esplora ogni prospettiva.</p>
        </div>
        <button class="reference-card" @click="photo = true">
          <img
            :src="photoUrl"
            alt="Foto originale: tribune grigie e gialle, ringhiere blu e tetto in legno del palazzetto"
          /><span
            ><Icon name="photo" /> La foto di partenza <Icon name="expand"
          /></span>
        </button>
        <div class="panel-tabs" role="tablist" aria-label="Impostazioni">
          <button
            id="views-tab"
            role="tab"
            :aria-selected="activeTab === 'views'"
            aria-controls="views-panel"
            @click="activeTab = 'views'"
            @keydown.right.prevent="
              activeTab = 'scene';
              $refs.sceneTab.focus();
            "
            @keydown.left.prevent="
              activeTab = 'scene';
              $refs.sceneTab.focus();
            "
            ref="viewsTab"
            :tabindex="activeTab === 'views' ? 0 : -1"
          >
            <Icon name="view" /> Prospettive <span>05</span></button
          ><button
            id="scene-tab"
            role="tab"
            :aria-selected="activeTab === 'scene'"
            aria-controls="scene-panel"
            @click="activeTab = 'scene'"
            @keydown.left.prevent="
              activeTab = 'views';
              $refs.viewsTab.focus();
            "
            @keydown.right.prevent="
              activeTab = 'views';
              $refs.viewsTab.focus();
            "
            ref="sceneTab"
            :tabindex="activeTab === 'scene' ? 0 : -1"
          >
            <Icon name="cube" /> Scena
          </button>
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
            @click="choose(preset.id)"
            :disabled="!ready"
          >
            <span class="view-number">0{{ index + 1 }}</span
            ><span
              ><strong>{{ preset.label }}</strong
              ><small>{{ preset.subtitle }}</small></span
            ><Icon name="chevron" />
          </button>
        </div>
        <div
          v-show="activeTab === 'scene'"
          id="scene-panel"
          class="settings-panel"
          role="tabpanel"
          aria-labelledby="scene-tab"
        >
          <p class="setting-hint">
            I dettagli che rendono riconoscibile questo spazio.
          </p>
          <label
            ><span
              ><strong>Sedie sul parquet</strong
              ><small>{{ stats.chairs }} sedie · layer opzionale</small></span
            ><input
              type="checkbox"
              v-model="chairs"
              role="switch" /><i></i></label
          ><label
            ><span
              ><strong>Tetto in legno</strong
              ><small>Travi e copertura a falde</small></span
            ><input
              type="checkbox"
              v-model="roof"
              role="switch" /><i></i></label
          ><label
            ><span
              ><strong>Barriere del campo</strong
              ><small>Ringhiere perimetrali blu</small></span
            ><input type="checkbox" v-model="railings" role="switch" /><i></i
          ></label>
          <div class="cutaway-note">
            <Icon name="info" />
            <p>
              Le superfici esterne si aprono alla vista per lasciarti esplorare
              l’interno.
            </p>
          </div>
        </div>
        <div class="mode-control">
          <span class="eyebrow">ALLESTIMENTO</span>
          <div>
            <button
              :class="{ active: chairs }"
              :aria-pressed="chairs"
              @click="changeMode(true)"
            >
              <i></i> Come nella foto</button
            ><button
              :class="{ active: !chairs }"
              :aria-pressed="!chairs"
              @click="changeMode(false)"
            >
              Giorno evento
            </button>
          </div>
        </div>
        <div class="reel-card">
          <div>
            <span class="eyebrow">LASCIATI GUIDARE</span
            ><span class="reel-duration">{{ TOUR_SECONDS }} SEC</span>
          </div>
          <h3>Un giro. Mille prospettive.</h3>
          <p>Dal parquet alle tribune, in un unico<br />movimento di camera.</p>
          <button class="tour-button" @click="play" :disabled="!ready">
            <Icon :name="tour ? 'stop' : 'play'" /><span>{{
              tour ? "Interrompi tour" : "Avvia tour cinematico"
            }}</span
            ><Icon name="arrow" /></button
          ><span class="reel-hint">{{
            tour
              ? "Tour in corso · puoi interromperlo in ogni momento"
              : "Prova il formato 9:16 per una presentazione verticale"
          }}</span>
        </div>
        <div class="panel-footnote">
          <span class="small-cross">+</span> Un primo spazio per le prossime<br />esperienze
          ArenaBoostX.
        </div>
      </aside>
    </main>
    <footer class="lab-footer">
      <span>ARENA LAB <b>—</b> ESPERIENZE CHE PRENDONO SPAZIO.</span
      ><span
        >Ricostruzione interpretativa da singola fotografia
        <i>© ABX 2026</i></span
      >
    </footer>
    <dialog
      ref="photoDialog"
      class="photo-dialog"
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
          alt="Fotografia di riferimento del palazzetto Joy Volley a Gioia del Colle"
        />
        <p>
          <span>01 / LA FOTO DI PARTENZA</span> Proporzioni, colori e dettagli
          della ricostruzione derivano da questa immagine.
        </p>
      </div>
    </dialog>
  </div>
</template>
