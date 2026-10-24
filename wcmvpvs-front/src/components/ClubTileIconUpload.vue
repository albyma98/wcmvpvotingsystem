<script setup>
import { onBeforeUnmount, ref } from 'vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  label: { type: String, required: true },
  fallbackImage: { type: String, default: '' },
  fallbackIcon: { type: String, default: '🏐' },
})
const emit = defineEmits(['update:modelValue', 'uploading'])
const loading = ref(false)
const error = ref('')
let disposed = false
onBeforeUnmount(() => {
  disposed = true
  if (loading.value) emit('uploading', false)
})

async function upload(event) {
  const input = event.target
  const file = input.files?.[0]
  if (!file) return
  input.value = ''
  error.value = ''
  if (!file.type.startsWith('image/')) { error.value = 'Seleziona un file immagine.'; return }
  if (file.size > 5 * 1024 * 1024) { error.value = 'L’immagine deve pesare al massimo 5 MB.'; return }
  loading.value = true
  emit('uploading', true)
  const objectUrl = URL.createObjectURL(file)
  try {
    const img = new Image()
    await new Promise((resolve, reject) => {
      img.onload = resolve
      img.onerror = () => reject(new Error('Impossibile leggere questa immagine.'))
      img.src = objectUrl
    })
    const canvas = document.createElement('canvas')
    const context = canvas.getContext('2d')
    if (!context) throw new Error('Caricamento immagine non disponibile.')
    let dataUrl = ''
    for (const maxSide of [256, 160, 96]) {
      const scale = Math.min(1, maxSide / Math.max(img.naturalWidth, img.naturalHeight))
      canvas.width = Math.max(1, Math.round(img.naturalWidth * scale))
      canvas.height = Math.max(1, Math.round(img.naturalHeight * scale))
      context.drawImage(img, 0, 0, canvas.width, canvas.height)
      dataUrl = canvas.toDataURL('image/webp', 0.85)
      if (dataUrl.length <= 160 * 1024) break
    }
    if (!/^data:image\/(webp|png);base64,/.test(dataUrl) || dataUrl.length > 160 * 1024) throw new Error('Immagine troppo complessa: scegli un file più piccolo.')
    if (!disposed) emit('update:modelValue', dataUrl)
  } catch (e) {
    if (!disposed) error.value = e.message || 'Caricamento immagine fallito.'
  } finally {
    URL.revokeObjectURL(objectUrl)
    loading.value = false
    if (!disposed) emit('uploading', false)
  }
}
</script>

<template>
  <article class="tile-icon-editor">
    <div class="icon-preview" aria-hidden="true">
      <img v-if="modelValue || fallbackImage" :src="modelValue || fallbackImage" alt="" />
      <span v-else>{{ fallbackIcon }}</span>
    </div>
    <div class="icon-controls">
      <strong>{{ label }}</strong>
      <label class="upload-label">{{ loading ? 'Caricamento…' : 'Carica immagine' }}
        <input type="file" accept="image/*" :aria-label="`Carica icona ${label}`" :disabled="loading" @change="upload" />
      </label>
      <button v-if="modelValue" type="button" :disabled="loading" :aria-label="`Ripristina icona ${label}`" @click="emit('update:modelValue', '')">Ripristina predefinita</button>
      <p v-if="error" role="alert">{{ error }}</p>
    </div>
  </article>
</template>

<style scoped>
.tile-icon-editor{display:flex;align-items:flex-start;gap:14px;padding:16px;border:1px solid #263449;border-radius:14px;min-width:0;background:#0b1526}.icon-preview{width:58px;height:58px;flex:none;border-radius:14px;background:#fff;display:grid;place-items:center;color:#172554;font-size:26px;padding:7px}.icon-preview img{width:100%;height:100%;object-fit:contain}.icon-controls{min-width:0;display:flex;align-items:flex-start;flex-direction:column;gap:8px}.icon-controls strong{font-size:14px;color:#f8fafc;overflow-wrap:anywhere}.upload-label{position:relative;overflow:hidden;padding:7px 10px;border-radius:7px;background:#24334b;color:#e2e8f0;font-size:12px;font-weight:700;cursor:pointer}.upload-label:focus-within{outline:2px solid #f59e0b;outline-offset:2px}.upload-label input{position:absolute;inset:0;opacity:0;cursor:pointer;width:100%;height:100%}.icon-controls button{padding:0;border:0;background:none;color:#94a3b8;font-size:11px;cursor:pointer;text-decoration:underline}.icon-controls button:disabled{opacity:.5;cursor:wait}.icon-controls p{margin:0;color:#fca5a5;font-size:12px}
</style>
