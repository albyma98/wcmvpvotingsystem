<template>
  <v-app-bar flat density="comfortable" height="64" class="app-topbar">
    <div class="d-flex align-center gap-3">
      <div class="d-flex align-center gap-2">
        <v-chip color="primary" variant="flat" size="small" class="font-weight-bold">
          {{ badge }}
        </v-chip>
        <div>
          <div class="text-subtitle-1 font-weight-bold">{{ title }}</div>
          <div v-if="subtitle" class="text-body-2 text-medium-emphasis">
            {{ subtitle }}
          </div>
        </div>
      </div>
    </div>
    <v-spacer />
    <v-btn
      variant="text"
      color="primary"
      :icon="isDark ? 'mdi-weather-sunny' : 'mdi-moon-waning-crescent'"
      @click="toggleTheme"
    />
  </v-app-bar>
</template>

<script setup>
import { computed } from 'vue';
import { useTheme } from 'vuetify';

const props = defineProps({
  title: {
    type: String,
    required: true,
  },
  subtitle: {
    type: String,
    default: '',
  },
  badge: {
    type: String,
    default: '',
  },
});

const theme = useTheme();
const isDark = computed(() => theme.global.current.value.dark);

function toggleTheme() {
  theme.global.name.value = isDark.value ? 'light' : 'dark';
}
</script>

<style scoped>
.app-topbar {
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  backdrop-filter: blur(8px);
}
</style>
