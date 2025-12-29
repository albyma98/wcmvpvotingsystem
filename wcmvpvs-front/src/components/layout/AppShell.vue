<template>
  <v-app>
    <v-layout>
      <TopBar :title="title" :subtitle="subtitle" :badge="badge" />
      <SideNav
        :items="navItems"
        :active-key="activeKey"
        :dense="denseNav"
        @navigate="$emit('navigate', $event)"
      />
      <v-main class="app-shell__main">
        <div class="app-shell__content">
          <slot />
        </div>
      </v-main>
    </v-layout>
  </v-app>
</template>

<script setup>
import TopBar from './TopBar.vue';
import SideNav from './SideNav.vue';

defineProps({
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
  navItems: {
    type: Array,
    default: () => [],
  },
  activeKey: {
    type: String,
    default: '',
  },
  denseNav: {
    type: Boolean,
    default: false,
  },
});

defineEmits(['navigate']);
</script>

<style scoped>
.app-shell__main {
  background: linear-gradient(120deg, rgba(124, 172, 248, 0.14), rgba(22, 29, 43, 0.6));
  min-height: 100vh;
}

.app-shell__content {
  padding: 24px;
  max-width: 1600px;
  margin: 0 auto;
}
</style>
