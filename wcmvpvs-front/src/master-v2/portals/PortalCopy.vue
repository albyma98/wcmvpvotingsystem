<script setup>
import { ref } from 'vue'
import MasterIcon from '../ui/MasterIcon.vue'

const props = defineProps({
  value: { type: String, required: true },
  label: { type: String, default: 'Copia' },
})
const status = ref('')
const failed = ref(false)

async function copy() {
  try {
    await navigator.clipboard.writeText(props.value)
    failed.value = false
    status.value = 'Copiato negli appunti.'
  } catch {
    failed.value = true
    status.value = 'Copia manualmente il valore.'
  }
}
</script>

<template>
  <span class="portal-copy">
    <button class="m-btn m-subtle" type="button" @click="copy">
      <MasterIcon name="copy" :size="14" /> {{ label }}
    </button>
    <small v-if="status" :class="failed ? 'm-error' : 'm-muted-text'" role="status">{{ status }}</small>
  </span>
</template>

<style scoped>
.portal-copy { display: inline-flex; flex-wrap: wrap; align-items: center; gap: 6px; }
</style>
