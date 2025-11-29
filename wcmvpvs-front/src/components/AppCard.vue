<template>
  <Card :class="['app-card', tone && `tone-${tone}`]">
    <template #title>
      <div class="card-title">
        <i v-if="icon" :class="['pi', icon]"></i>
        <div class="card-title__text">
          <span>{{ title }}</span>
          <small v-if="subtitle">{{ subtitle }}</small>
        </div>
        <strong v-if="value" class="card-value">{{ value }}</strong>
      </div>
    </template>
    <template v-if="$slots.subtitle" #subtitle>
      <slot name="subtitle" />
    </template>
    <template #content>
      <slot name="content">
        <slot />
      </slot>
    </template>
  </Card>
</template>

<script setup>
import { defineProps } from 'vue';

const props = defineProps({
  title: { type: String, default: '' },
  subtitle: { type: String, default: '' },
  icon: { type: String, default: '' },
  value: { type: [String, Number], default: '' },
  tone: { type: String, default: '' },
});
</script>

<style scoped>
.app-card :deep(.p-card) {
  border-radius: 14px;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  color: #0f172a;
}

.card-title__text {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.card-title__text small {
  color: #64748b;
}

.card-title i {
  color: #6366f1;
}

.card-value {
  margin-left: auto;
  font-size: 1.25rem;
  color: #0f172a;
}

.tone-success :deep(.p-card) {
  background: linear-gradient(135deg, #ecfeff, #e0f2fe);
}

.tone-warning :deep(.p-card) {
  background: linear-gradient(135deg, #fef9c3, #fffbeb);
}

.tone-danger :deep(.p-card) {
  background: linear-gradient(135deg, #fee2e2, #fef2f2);
}
</style>
