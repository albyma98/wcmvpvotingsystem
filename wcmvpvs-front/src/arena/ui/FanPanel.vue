<script setup>
import { computed, ref } from "vue";
import Icon from "../ArenaIcon.vue";
import { FAN_ACTIONS } from "../actions/demoSession.js";
const props = defineProps({
  profiles: Array,
  fan: Object,
  ready: Boolean,
  cooldowns: Object,
  voted: Boolean,
  lastAction: String,
});
const emit = defineEmits(["select", "find", "action", "vote"]);
const query = ref(""),
  searchError = ref("");
const results = computed(() =>
  query.value.trim()
    ? props.profiles
        .filter((fan) =>
          fan.nickname
            .toLocaleLowerCase("it")
            .includes(query.value.trim().toLocaleLowerCase("it")),
        )
        .slice(0, 5)
    : [],
);
function select(id) {
  emit("select", id);
  query.value = "";
  searchError.value = "";
}
function search() {
  const exact = results.value.find(
    (fan) => fan.nickname.toLowerCase() === query.value.trim().toLowerCase(),
  );
  if (exact || results.value.length === 1)
    select((exact || results.value[0]).id);
  else
    searchError.value = results.value.length
      ? "Scegli un tifoso dai risultati."
      : "Nessun profilo demo trovato.";
}
const initials = computed(
  () =>
    props.fan.nickname[0] +
    (props.fan.nickname.match(/[A-Z]/g)?.[1] || props.fan.nickname[1]),
);
</script>
<template>
  <div class="fan-panel">
    <label class="profile-picker"
      >IL TUO PROFILO DEMO<select
        :value="fan.id"
        @change="select($event.target.value)"
      >
        <option
          v-for="profile in profiles"
          :key="profile.id"
          :value="profile.id"
        >
          {{ profile.nickname }}
        </option>
      </select></label
    >
    <form class="fan-search" @submit.prevent="search">
      <Icon name="search" /><input
        v-model="query"
        aria-label="Cerca nickname demo"
        placeholder="Cerca un nickname…"
        autocomplete="off"
        @input="searchError = ''"
      /><button
        type="submit"
        aria-label="Cerca tifoso"
        :disabled="!query.trim()"
      >
        <Icon name="arrow" />
      </button>
    </form>
    <ul
      v-if="results.length"
      class="fan-search-results"
      aria-label="Risultati ricerca tifosi"
    >
      <li v-for="result in results" :key="result.id">
        <button @click="select(result.id)">
          <span>{{ result.nickname }}</span
          ><small>{{ result.seat.sector }}</small
          ><Icon name="chevron" />
        </button>
      </li>
    </ul>
    <p v-if="searchError" class="fan-search-error" role="status">
      {{ searchError }}
    </p>
    <div class="fan-identity">
      <span class="fan-monogram">{{ initials }}<i></i></span>
      <div>
        <strong>{{ fan.nickname }}</strong
        ><span>Livello {{ fan.level }} · Tifoso demo</span>
      </div>
      <span class="fan-level">L{{ fan.level }}</span>
    </div>
    <div class="fan-seat-address">
      <Icon name="pin" />
      <div>
        <strong>Settore {{ fan.seat.sector }}</strong
        ><span
          >Fila {{ String(fan.seat.row).padStart(2, "0") }} <i>·</i> Posto
          {{ String(fan.seat.number).padStart(2, "0") }}</span
        >
      </div>
      <code>{{ fan.seatId }}</code>
    </div>
    <button class="find-fan-button" :disabled="!ready" @click="emit('find')">
      <Icon name="pin" />Trova il mio posto<Icon name="arrow" />
    </button>
    <div class="fan-actions-heading">
      <span class="eyebrow">FAI SENTIRE LA TUA PRESENZA</span
      ><span class="action-live-dot"></span>
    </div>
    <button class="mvp-entry" :disabled="!ready" @click="emit('vote')">
      <span class="mvp-entry-icon"><Icon name="trophy" /></span
      ><span
        ><strong>Vota MVP</strong
        ><small>{{
          voted ? "Il tuo voto è registrato" : "Il tuo voto accende la tribuna"
        }}</small></span
      ><Icon :name="voted ? 'check' : 'arrow'" />
    </button>
    <div class="fan-actions-grid">
      <button
        v-for="(action, kind) in FAN_ACTIONS"
        :key="kind"
        :disabled="!ready || cooldowns[kind] > 0"
        :class="{ reacting: lastAction === kind }"
        @click="emit('action', kind)"
      >
        <Icon
          :name="kind === 'clap' ? 'clap' : kind === 'scarf' ? 'scarf' : 'wave'"
        /><strong>{{ action.label }}</strong
        ><span>{{
          cooldowns[kind] > 0
            ? `Tra ${cooldowns[kind]} s`
            : kind === "wave"
              ? "Tutta l’arena"
              : "Insieme al tuo settore"
        }}</span>
      </button>
    </div>
    <div class="fan-history">
      <span>{{ fan.matches }} partite demo</span
      ><span>{{ fan.interactions }} interazioni</span>
    </div>
    <p class="fan-last-action"><i></i>{{ fan.recentAction }}</p>
    <p class="fan-demo-note">
      Seleziona uno dei 60 profili o tocca un avatar con il punto verde. Posti e
      voti restano su questo dispositivo.
    </p>
  </div>
</template>
