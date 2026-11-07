<script setup>
import { onBeforeUnmount, onMounted, ref, useId } from 'vue';
import MasterIcon from './MasterIcon.vue';

const props = defineProps({ title: { type: String, required: true }, description: String, busy: Boolean });
const emit = defineEmits(['close']);
const dialog = ref(null);
const titleId = useId();
const descriptionId = useId();

onMounted(() => dialog.value.showModal());
onBeforeUnmount(() => dialog.value?.close());

function dismiss() {
  if (!props.busy) emit('close');
}
</script>

<template>
  <dialog ref="dialog" class="m-dialog" :aria-labelledby="titleId" :aria-describedby="description ? descriptionId : undefined" @cancel.prevent="dismiss">
    <div class="m-dialog-head">
      <div>
        <h2 :id="titleId">{{ title }}</h2>
        <p v-if="description" :id="descriptionId">{{ description }}</p>
      </div>
      <button type="button" class="m-btn m-subtle m-icon-btn" aria-label="Chiudi finestra" :disabled="busy" @click="dismiss"><MasterIcon name="close" /></button>
    </div>
    <div v-if="$slots.default" class="m-dialog-body"><slot /></div>
    <div v-if="$slots.actions" class="m-dialog-actions"><slot name="actions" /></div>
  </dialog>
</template>
