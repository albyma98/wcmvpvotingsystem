<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

const props = defineProps({ photo: { type: Object, required: true } })
const emit = defineEmits(['close'])
const dialog = ref(null)
const failed = ref(false)
const title = computed(() => props.photo.title || props.photo.caption || 'Foto del club')

function closeOutside(event) {
  if (event.target !== dialog.value) return
  const rect = dialog.value.getBoundingClientRect()
  if (event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom) emit('close')
}

onMounted(() => dialog.value?.showModal())
onBeforeUnmount(() => dialog.value?.close())
</script>

<template>
  <dialog ref="dialog" class="club-lightbox" :aria-label="`Immagine ingrandita: ${title}`" @cancel.prevent="emit('close')" @click="closeOutside">
    <header class="lightbox-header">
      <h2>{{ title }}</h2>
      <button type="button" autofocus aria-label="Chiudi immagine ingrandita" @click="emit('close')">
        <svg viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m6 6 12 12M18 6 6 18" /></svg>
      </button>
    </header>
    <figure>
      <p v-if="failed" class="lightbox-error" role="alert">Impossibile caricare l’immagine. Chiudi e riprova.</p>
      <img v-else :src="photo.imageUrl" :alt="title" @error="failed = true" />
      <figcaption v-if="photo.caption || photo.teamName">{{ photo.caption || photo.teamName }}</figcaption>
    </figure>
  </dialog>
</template>

<style scoped>
.club-lightbox{box-sizing:border-box;width:min(1440px,calc(100vw - 48px));height:calc(100dvh - 48px);max-width:none;max-height:none;margin:auto;padding:0;border:1px solid #475569;border-radius:16px;background:#07101f;color:#f8fafc;overflow:hidden;font-family:var(--club-font-body,Inter,system-ui,sans-serif);box-shadow:0 24px 80px #0008}
.club-lightbox[open]{display:flex;flex-direction:column}.club-lightbox::backdrop{background:rgba(3,7,18,.9)}
.club-lightbox *{box-sizing:border-box}.lightbox-header{flex:none;display:flex;align-items:center;justify-content:space-between;gap:16px;padding:12px 16px;border-bottom:1px solid #263449}
.lightbox-header h2{min-width:0;margin:0;font-family:var(--club-font-title,inherit);font-size:16px;line-height:1.4;overflow-wrap:anywhere;display:-webkit-box;-webkit-line-clamp:2;-webkit-box-orient:vertical;overflow:hidden}
.club-lightbox .lightbox-header button{flex:none;display:grid;place-items:center;width:44px;height:44px;padding:0;border:1px solid #64748b;border-radius:10px;background:#172238;color:#fff;cursor:pointer}
.lightbox-header button svg{width:24px;height:24px;stroke:currentColor;stroke-width:2;stroke-linecap:round}.club-lightbox button:focus-visible{outline:3px solid #fbbf24;outline-offset:3px}
.club-lightbox figure{flex:1;min-height:0;min-width:0;display:flex;flex-direction:column;gap:12px;margin:0;padding:16px}
.club-lightbox img{display:block;flex:1;min-height:0;width:100%;height:100%;object-fit:contain}
.club-lightbox figcaption{flex:none;max-height:20dvh;overflow:auto;color:#cbd5e1;font-size:14px;line-height:1.5;text-align:center;white-space:pre-line;overflow-wrap:anywhere}
.lightbox-error{margin:auto;text-align:center;color:#fecdd3;font-size:14px;line-height:1.5}
@media(max-width:700px){.club-lightbox{width:100vw;height:100dvh;border:0;border-radius:0}.lightbox-header{padding:max(12px,env(safe-area-inset-top)) max(12px,env(safe-area-inset-right)) 12px max(12px,env(safe-area-inset-left))}.club-lightbox figure{padding:12px max(12px,env(safe-area-inset-right)) max(12px,env(safe-area-inset-bottom)) max(12px,env(safe-area-inset-left))}}
</style>
