<template>
  <v-navigation-drawer
    width="260"
    permanent
    floating
    class="app-sidenav"
    :model-value="true"
  >
    <v-list nav density="comfortable">
      <template v-for="section in items" :key="section.header">
        <v-list-subheader class="text-uppercase text-body-2 text-medium-emphasis">
          {{ section.header }}
        </v-list-subheader>
        <v-list-item
          v-for="item in section.items"
          :key="item.key"
          :value="item.key"
          :active="item.key === activeKey"
          color="primary"
          rounded="lg"
          @click="$emit('navigate', item.key)"
        >
          <v-list-item-title>{{ item.label }}</v-list-item-title>
        </v-list-item>
      </template>
    </v-list>
  </v-navigation-drawer>
</template>

<script setup>
defineProps({
  items: {
    type: Array,
    default: () => [],
  },
  activeKey: {
    type: String,
    default: '',
  },
});

defineEmits(['navigate']);
</script>

<style scoped>
.app-sidenav {
  border-right: 1px solid rgba(255, 255, 255, 0.04);
}
</style>
