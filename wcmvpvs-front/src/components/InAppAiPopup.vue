<template>
  <Transition name="ai-popup-fade">
    <div v-if="modelValue && popup" class="fixed inset-0 z-[260] flex items-end justify-center bg-slate-950/45 px-4 pb-6 pt-10 sm:items-center" @click.self="dismiss">
      <div class="w-full max-w-md overflow-hidden rounded-[28px] border border-white/10 bg-slate-950 text-white shadow-[0_28px_90px_rgba(15,23,42,0.65)]">
        <div class="bg-gradient-to-r from-amber-400 to-orange-500 px-5 py-3 text-xs font-black uppercase tracking-[0.28em] text-slate-950">
          {{ popup.tone || 'Ai nudge' }}
        </div>
        <div class="space-y-4 px-5 py-5">
          <div class="flex items-start justify-between gap-3">
            <div>
              <p class="text-2xl font-black leading-tight">{{ popup.popup_title }}</p>
              <p class="mt-2 text-sm leading-6 text-slate-200">{{ popup.popup_body }}</p>
            </div>
            <button type="button" class="rounded-full border border-white/15 px-3 py-1 text-xs font-bold uppercase text-slate-300" @click="dismiss">Chiudi</button>
          </div>
          <div class="flex items-center justify-between gap-3">
            <span class="rounded-full bg-white/10 px-3 py-1 text-[0.7rem] font-bold uppercase tracking-[0.18em] text-amber-300">
              {{ popup.urgency_level || 'medium' }}
            </span>
            <button type="button" class="rounded-2xl bg-amber-400 px-4 py-3 text-sm font-black uppercase tracking-wide text-slate-950" @click="ctaClick">
              {{ popup.cta_text || 'Apri' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup>
const props = defineProps({
  modelValue: { type: Boolean, default: false },
  popup: { type: Object, default: null },
});

const emit = defineEmits(['update:modelValue', 'cta', 'dismiss']);

function dismiss() {
  emit('dismiss');
  emit('update:modelValue', false);
}

function ctaClick() {
  emit('cta');
  emit('update:modelValue', false);
}
</script>

<style scoped>
.ai-popup-fade-enter-active,
.ai-popup-fade-leave-active {
  transition: opacity 0.2s ease;
}
.ai-popup-fade-enter-from,
.ai-popup-fade-leave-to {
  opacity: 0;
}
</style>
