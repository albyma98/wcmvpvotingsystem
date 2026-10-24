<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  label: { type: String, required: true },
  required: Boolean,
  disabled: Boolean,
  maxSide: { type: Number, default: 1600 },
})
const emit = defineEmits(['update:modelValue', 'uploading'])
const loading = ref(false)
const error = ref('')
const previewFailed = ref(false)
const fileInput = ref(null)
const linkInput = ref(null)
const remoteValue = computed(() => props.modelValue.startsWith('data:') ? '' : props.modelValue)
let disposed = false
watch(() => props.modelValue, () => { previewFailed.value = false; error.value = ''; linkInput.value?.setCustomValidity('') })
onBeforeUnmount(() => {
  disposed = true
  if (loading.value) emit('uploading', false)
})

function setLink(value, input) {
  const link = value.trim()
  // Images can be hosted externally or within the existing site assets.
  if (link && !/^https?:\/\/\S+$/i.test(link) && !/^\/(?!\/)\S+$/.test(link)) {
    error.value = 'Usa un collegamento https://, http:// oppure un percorso del sito che inizi con /.'
    input?.setCustomValidity(error.value)
    return
  }
  error.value = ''
  input?.setCustomValidity('')
  emit('update:modelValue', link)
}

async function upload(event) {
  const input = event.target
  const file = input.files?.[0]
  input.value = ''
  if (!file || loading.value || props.disabled) return
  error.value = ''
  if (!['image/jpeg', 'image/png', 'image/webp', 'image/gif'].includes(file.type)) {
    error.value = 'Scegli un’immagine JPG, PNG, WebP o GIF.'
    return
  }
  if (file.size > 10 * 1024 * 1024) { error.value = 'Scegli un’immagine fino a 10 MB.'; return }
  loading.value = true
  emit('uploading', true)
  let objectUrl
  try {
    objectUrl = URL.createObjectURL(file)
    const img = new Image()
    await new Promise((resolve, reject) => {
      img.onload = resolve
      img.onerror = () => reject(new Error('Impossibile leggere l’immagine. Prova con un altro file.'))
      img.src = objectUrl
    })
    const canvas = document.createElement('canvas')
    const context = canvas.getContext('2d')
    if (!context || !img.naturalWidth || !img.naturalHeight) throw new Error('Elaborazione immagine non disponibile.')
    let encoded = ''
    for (const maxSide of [props.maxSide, Math.round(props.maxSide * .75), Math.round(props.maxSide * .5)]) {
      const scale = Math.min(1, maxSide / Math.max(img.naturalWidth, img.naturalHeight))
      canvas.width = Math.max(1, Math.round(img.naturalWidth * scale))
      canvas.height = Math.max(1, Math.round(img.naturalHeight * scale))
      context.clearRect(0, 0, canvas.width, canvas.height)
      context.drawImage(img, 0, 0, canvas.width, canvas.height)
      encoded = canvas.toDataURL('image/webp', .82)
      if (encoded.length <= 300 * 1024) break
    }
    if (!/^data:image\/(webp|png);base64,/.test(encoded) || encoded.length > 300 * 1024) {
      throw new Error('L’immagine è troppo complessa. Scegli una versione più piccola.')
    }
    if (!disposed) emit('update:modelValue', encoded)
  } catch (e) {
    if (!disposed) error.value = e.message || 'Caricamento non riuscito. Riprova.'
  } finally {
    if (objectUrl) URL.revokeObjectURL(objectUrl)
    loading.value = false
    if (!disposed) emit('uploading', false)
  }
}
</script>

<template>
  <div class="media-field" :aria-busy="loading">
    <span class="media-label">{{ label }}{{ required ? ' *' : '' }}</span>
    <div class="media-dropzone">
      <div v-if="modelValue" class="media-thumbnail">
        <img v-if="!previewFailed" :src="modelValue" :alt="`Anteprima ${label}`" @error="previewFailed = true" />
        <span v-else>Anteprima non disponibile. Verifica il collegamento.</span>
      </div>
      <div class="media-controls">
        <button type="button" :disabled="disabled || loading" @click="fileInput?.click()">{{ loading ? 'Elaborazione…' : modelValue ? 'Sostituisci immagine' : 'Carica immagine' }}</button>
        <input ref="fileInput" class="file-input" type="file" accept="image/jpeg,image/png,image/webp,image/gif" :aria-label="`Carica ${label}`" :disabled="disabled || loading" :required="required && !modelValue" @change="upload" />
        <span>JPG, PNG, WebP o GIF · fino a 10 MB<br />Dimensioni ottimizzate automaticamente. Le GIF vengono salvate come immagini statiche.</span>
        <button v-if="modelValue" class="remove-media" type="button" :disabled="disabled || loading" :aria-label="`Rimuovi ${label}`" @click="emit('update:modelValue', '')">Rimuovi immagine</button>
      </div>
    </div>
    <details class="media-link"><summary>Oppure usa un collegamento</summary><label>Collegamento · {{ label }}<input ref="linkInput" :value="remoteValue" type="text" :disabled="disabled || loading" placeholder="https://… oppure /immagini/foto.jpg" @invalid="$event.target.closest('details').open = true" @change="setLink($event.target.value, $event.target)" /></label></details>
    <p v-if="error" class="media-error" role="alert">{{ error }}</p>
    <p v-else-if="modelValue && required" class="media-ready">Immagine pronta. Premi Salva per applicarla.</p>
  </div>
</template>

<style scoped>
.media-field{min-width:0;display:grid;gap:10px;color:#bdcbdd;font-size:13px}.media-label{font-weight:600}.media-dropzone{position:relative;display:flex;align-items:center;gap:18px;padding:18px;border:1px dashed #40516b;border-radius:10px;background:#091322}.media-thumbnail{width:108px;height:90px;flex:none;display:grid;place-items:center;overflow:hidden;border-radius:8px;background:#17243a}.media-thumbnail img{width:100%;height:100%;object-fit:contain}.media-thumbnail>span{padding:8px;font-size:10px;text-align:center;color:#fda4af}.media-controls{min-width:0;display:grid;justify-items:start;gap:9px}.media-controls>span{color:#9aabc1;font-size:11px;line-height:1.6}.media-field button{border:1px solid #40516b;border-radius:8px;background:#1c304b;color:#f1f5f9;padding:9px 12px;font:inherit;font-weight:650;cursor:pointer}.media-field .remove-media{border:0;background:none;padding:0;font-size:11px;color:#fda4af}.file-input{position:absolute;width:1px;height:1px;opacity:0;bottom:12px;left:18px}.media-link summary{cursor:pointer;color:#a5b4c9;font-size:12px}.media-link label{display:grid;gap:8px;padding-top:12px;font-size:12px}.media-link input{width:100%;min-width:0;padding:10px 12px;border:1px solid #40516b;border-radius:8px;background:#091322;color:#f8fafc;font:inherit;font-size:14px}.media-error{color:#fda4af;font-size:12px;margin:0}.media-ready{color:#86efac;font-size:11px;margin:0}.media-field button:disabled{opacity:.5;cursor:not-allowed}.media-field button:focus-visible,.media-link summary:focus-visible,.media-link input:focus-visible{outline:3px solid #fbbf24;outline-offset:3px}@media(max-width:600px){.media-dropzone{align-items:flex-start;flex-direction:column}.media-thumbnail{width:100%;height:140px}}
</style>
