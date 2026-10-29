<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from "vue";
import { apiClient, getFanSessionHeaders } from "../api";
import { getOrCreateDeviceId } from "../deviceId";
import { createArena } from "./scene/createArena.js";
import { organizationCrowd } from "./organizationCrowd.js";
import { FAN_ACTIONS } from "./actions/demoSession.js";
import { presets } from "./camera/cameraSystem.js";
import Icon from "./ArenaIcon.vue";
const props = defineProps({ slug: { type: String, required: true } });
const viewport = ref(null),
  loading = ref(true),
  error = ref(""),
  actors = ref([]),
  me = ref(null),
  inspected = ref(null),
  query = ref(""),
  panel = ref(false),
  view = ref("general"),
  total = ref(0),
  notice = ref(""),
  cooldowns = ref({}),
  clock = ref(Date.now());
const results = computed(() =>
  query.value.trim()
    ? actors.value
        .filter((a) =>
          a.profile.nickname
            .toLowerCase()
            .includes(query.value.trim().toLowerCase()),
        )
        .slice(0, 20)
    : [],
);
const home = computed(() => "/" + encodeURIComponent(props.slug) + "/new");
let arena,
  ticker,
  abort = new AbortController(),
  disposed = false;
function reload() {
  window.location.reload();
}
function focus(actor) {
  inspected.value = actor;
  panel.value = false;
  arena?.focusFan(actor.id);
}
function act(kind) {
  if (!me.value || clock.value < (cooldowns.value[kind] || 0)) return;
  arena?.fanAction(kind, me.value.id);
  cooldowns.value = {
    ...cooldowns.value,
    [kind]: Date.now() + FAN_ACTIONS[kind].cooldown,
  };
  notice.value = FAN_ACTIONS[kind].recent;
}
onMounted(async () => {
  try {
    const { data } = await apiClient.get("/fan/arena", {
      headers: {
        "X-Organization-Slug": props.slug,
        "X-Device-ID": getOrCreateDeviceId(),
        ...getFanSessionHeaders(props.slug),
      },
      signal: abort.signal,
    });
    if (disposed) return;
    const crowd = organizationCrowd(data.fans || [], data.organization_id);
    actors.value = crowd.actors;
    total.value = crowd.total;
    me.value = actors.value.find((a) => a.id === crowd.selectedId) || null;
    arena = createArena(
      viewport.value,
      {
        onView: (id) => (view.value = id),
        onError: (message) => (error.value = message),
        onSelectProfile: (id) => {
          const actor = actors.value.find((a) => a.id === id);
          if (actor) focus(actor);
        },
      },
      crowd,
    );
    arena.setChairs(false);
    ticker = setInterval(() => (clock.value = Date.now()), 250);
  } catch (e) {
    if (!disposed)
      error.value =
        e.response?.status === 404
          ? "Società non trovata."
          : "Impossibile caricare l’arena. Riprova tra poco.";
  } finally {
    loading.value = false;
  }
});
onBeforeUnmount(() => {
  disposed = true;
  abort.abort();
  clearInterval(ticker);
  arena?.dispose();
});
</script>

<template>
  <main class="organization-arena" aria-label="Arena della società">
    <div ref="viewport" class="organization-arena-canvas"></div>
    <header>
      <a :href="home" class="plate">← {{ slug }}</a
      ><strong class="brand"
        >Arena Tifosi <small>{{ total }} profili della società</small></strong
      ><button
        class="plate"
        @click="panel = !panel"
        :aria-expanded="panel"
        aria-label="Viste e ricerca tifosi"
      >
        <Icon name="settings" />
      </button>
    </header>
    <section
      v-if="panel"
      class="arena-menu plate"
      aria-label="Viste e ricerca tifosi"
    >
      <button class="close" @click="panel = false" aria-label="Chiudi menu">
        ×
      </button>
      <h2>La tua tribuna</h2>
      <label
        >Inquadratura<select
          :value="view"
          @change="arena?.view($event.target.value)"
        >
          <option v-for="preset in presets" :key="preset.id" :value="preset.id">
            {{ preset.label }}
          </option>
          <option v-if="view === 'custom'" value="custom">Vista libera</option>
        </select></label
      >
      <label
        >Cerca un tifoso<input
          v-model="query"
          type="search"
          placeholder="Nickname"
      /></label>
      <ul v-if="results.length">
        <li v-for="actor in results" :key="actor.id">
          <button @click="focus(actor)">
            {{ actor.profile.nickname }} <small>{{ actor.seat.id }}</small>
          </button>
        </li>
      </ul>
      <p v-else-if="query">Nessun tifoso visibile con questo nickname.</p>
      <p v-if="total > actors.length">
        {{ actors.length }} posti visibili su {{ total }} profili. La tribuna ha
        raggiunto la capienza.
      </p>
      <p>
        I profili rappresentano gli iscritti della società, non le presenze in
        tempo reale.
      </p>
    </section>
    <aside v-if="inspected" class="arena-person plate">
      <button @click="inspected = null" aria-label="Chiudi scheda tifoso">
        ×</button
      ><strong>{{ inspected.profile.nickname }}</strong
      ><small>Posto {{ inspected.seat.id }}</small
      ><span v-if="inspected.id === me?.id">Il tuo avatar</span>
    </aside>
    <footer v-if="!loading && !error">
      <button v-if="me" class="plate identity" @click="focus(me)">
        <Icon name="pin" /><span
          >{{ me.profile.nickname
          }}<small>Il mio posto · {{ me.seat.id }}</small></span
        >
      </button>
      <a v-else class="plate identity" :href="home"
        >Accedi o crea il tuo profilo</a
      >
      <nav class="plate" aria-label="Azioni del tifoso">
        <a :href="home"><Icon name="trophy" /><span>MVP</span></a
        ><button
          v-for="(action, kind) in FAN_ACTIONS"
          :key="kind"
          :disabled="!me || clock < (cooldowns[kind] || 0)"
          @click="act(kind)"
          :aria-label="action.label"
        >
          <Icon
            :name="
              kind === 'clap' ? 'clap' : kind === 'scarf' ? 'scarf' : 'wave'
            "
          /><span>{{
            kind === "clap" ? "Applauso" : kind === "scarf" ? "Sciarpa" : "Ola"
          }}</span
          ><small v-if="clock < (cooldowns[kind] || 0)"
            >{{ Math.ceil((cooldowns[kind] - clock) / 1000) }}s</small
          >
        </button>
      </nav>
    </footer>
    <p
      v-if="notice"
      role="status"
      class="arena-status plate"
      @click="notice = ''"
    >
      {{ notice }}
    </p>
    <div v-if="loading || error" class="arena-loading">
      <h1>{{ error || "Entra nella tua arena…" }}</h1>
      <template v-if="error"
        ><button @click="reload">Riprova</button
        ><a :href="home">Torna alla società</a></template
      >
    </div>
  </main>
</template>

<style scoped>
.organization-arena {
  position: fixed;
  inset: 0;
  height: 100dvh;
  overflow: hidden;
  background: #e9e8e0;
  color: #263b2d;
  font:
    14px Arial,
    sans-serif;
  z-index: 50;
}
.organization-arena * {
  box-sizing: border-box;
}
.organization-arena button,
.organization-arena a,
.organization-arena input,
.organization-arena select {
  font: inherit;
  color: inherit;
}
.organization-arena button {
  cursor: pointer;
  border: 0;
}
.organization-arena a {
  text-decoration: none;
}
.organization-arena button:focus-visible,
.organization-arena a:focus-visible {
  outline: 3px solid #648638;
}
.organization-arena button:disabled {
  opacity: 0.4;
  cursor: default;
}
.organization-arena-canvas {
  position: absolute;
  inset: 0;
  touch-action: none;
}
.organization-arena-canvas :deep(canvas) {
  width: 100%;
  height: 100%;
  display: block;
}
.plate {
  background: #f5f8e9ed;
  border: 1px solid #ffffffb3;
  border-radius: 16px;
  padding: 12px 16px;
  box-shadow: 0 4px 24px #243b1926;
  backdrop-filter: blur(14px);
}
header {
  position: absolute;
  inset: max(16px, env(safe-area-inset-top)) 16px auto;
  display: flex;
  align-items: center;
  gap: 16px;
  pointer-events: none;
}
header > * {
  pointer-events: auto;
}
.brand {
  flex: 1;
  text-shadow: 0 1px 8px #fff;
  font-size: 18px;
}
.brand small {
  display: block;
  font-size: 11px;
  margin-top: 4px;
}
svg {
  width: 22px;
  height: 22px;
}
.arena-menu {
  position: absolute;
  right: 16px;
  top: 82px;
  width: 320px;
  max-width: calc(100% - 32px);
  max-height: 70dvh;
  overflow: auto;
}
.arena-menu h2 {
  font-size: 20px;
  margin: 0 0 20px;
}
.arena-menu label {
  display: block;
  margin-top: 16px;
}
.arena-menu select,
.arena-menu input {
  display: block;
  width: 100%;
  min-height: 44px;
  margin-top: 7px;
  background: #fffef3;
  border: 1px solid #c9d7b6;
  border-radius: 8px;
  padding: 8px;
}
.arena-menu p {
  font-size: 12px;
  line-height: 1.6;
  margin-top: 18px;
}
.arena-menu ul {
  list-style: none;
  padding: 0;
}
.arena-menu li button {
  background: transparent;
  width: 100%;
  text-align: left;
  padding: 12px 0;
  display: flex;
  justify-content: space-between;
}
.close,
.arena-person > button {
  float: right;
  background: transparent;
  font-size: 24px;
}
.arena-person {
  position: absolute;
  top: 92px;
  left: 16px;
  max-width: 260px;
}
.arena-person strong,
.arena-person small,
.arena-person span {
  display: block;
  margin: 8px;
}
.arena-person small {
  font-size: 12px;
}
footer {
  position: absolute;
  bottom: max(20px, env(safe-area-inset-bottom));
  left: 20px;
  right: 20px;
  display: flex;
  justify-content: space-between;
  align-items: end;
  gap: 16px;
  pointer-events: none;
}
footer > * {
  pointer-events: auto;
}
.identity {
  display: flex;
  align-items: center;
  gap: 8px;
}
.identity small {
  display: block;
  margin-top: 6px;
  font-size: 11px;
}
nav {
  display: flex;
  gap: 4px;
}
nav button,
nav a {
  position: relative;
  background: transparent;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 7px;
  width: 60px;
  min-height: 60px;
  border-radius: 12px;
}
nav a {
  background: #d5e99f;
}
nav span {
  font-size: 10px;
}
nav small {
  position: absolute;
  top: 0;
  right: 0;
}
nav button:hover {
  background: #dce8bf;
}
.arena-status {
  position: absolute;
  top: 85px;
  left: 50%;
  transform: translateX(-50%);
  font-size: 12px;
}
.arena-loading {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 20px;
  background: #eff1e6;
  padding: 24px;
  text-align: center;
}
.arena-loading h1 {
  font-size: 24px;
}
.arena-loading button {
  padding: 12px 20px;
  border-radius: 8px;
  background: #d5e99f;
}
@media (max-width: 650px) {
  header {
    gap: 10px;
  }
  .brand {
    font-size: 15px;
  }
  .brand small {
    font-size: 10px;
  }
  header > a {
    max-width: 135px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  footer {
    flex-direction: column;
    align-items: center;
    gap: 16px;
    left: 14px;
    right: 14px;
  }
  footer .identity {
    align-self: flex-start;
    order: 2;
  }
  footer nav {
    order: 1;
  }
  nav button,
  nav a {
    width: 51px;
    min-height: 52px;
  }
  .arena-menu {
    top: 80px;
  }
  .arena-person {
    top: 80px;
  }
  .arena-status {
    top: 140px;
  }
}
</style>
