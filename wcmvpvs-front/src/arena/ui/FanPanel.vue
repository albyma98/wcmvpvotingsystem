<script setup>
import { computed, ref } from "vue";
import Icon from "../ArenaIcon.vue";
const props = defineProps({
  profiles: Array,
  fan: Object,
  ready: Boolean,
});
const emit = defineEmits(["select", "find"]);
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
