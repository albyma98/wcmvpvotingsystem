<script setup>
// Gallery del torneo: le miniature delle foto pubblicate dai tifosi + un pill
// acceso "SCATTA FOTO" che apre la fotocamera e pubblica. Le nuove foto degli
// altri compaiono da sole (SSE, gestito dal parent che ricarica su tick).
import { ref } from 'vue'

const props = defineProps({
  slug: { type: String, required: true },
  photos: { type: Array, default: () => [] }
})
const emit = defineEmits(['uploaded'])

const uploading = ref(false)
const error = ref('')
const viewer = ref(null) // id della foto aperta a tutto schermo
const fileInput = ref(null)

const imgUrl = id => `/api/v1/tournaments/${props.slug}/gallery/${id}/image`

// Ridimensiona la foto lato client (lato lungo max 1440px) → data-URL leggera.
// WebP quando supportato, JPEG di fallback (per le foto va benissimo).
function filePhotoToDataURL (file, maxSize = 1440) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    const img = new Image()
    reader.onload = () => { img.src = reader.result }
    reader.onerror = () => reject(new Error('read'))
    img.onerror = () => reject(new Error('decode'))
    img.onload = () => {
      const scale = Math.min(1, maxSize / Math.max(img.width, img.height))
      const w = Math.max(1, Math.round(img.width * scale))
      const h = Math.max(1, Math.round(img.height * scale))
      const canvas = document.createElement('canvas')
      canvas.width = w; canvas.height = h
      canvas.getContext('2d').drawImage(img, 0, 0, w, h)
      let out = canvas.toDataURL('image/webp', 0.82)
      if (!out.startsWith('data:image/webp')) out = canvas.toDataURL('image/jpeg', 0.85)
      resolve(out)
    }
    reader.readAsDataURL(file)
  })
}

function pickPhoto () { fileInput.value?.click() }

async function onPhotoPick (e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file) return
  if (!file.type.startsWith('image/')) { error.value = 'Seleziona un\'immagine.'; return }
  uploading.value = true
  error.value = ''
  try {
    const image = await filePhotoToDataURL(file)
    const res = await fetch(`/api/v1/tournaments/${props.slug}/gallery`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ image })
    })
    if (!res.ok) {
      const err = (await res.json().catch(() => ({}))).error
      error.value = err === 'image_too_large' ? 'Foto troppo pesante.' : 'Pubblicazione non riuscita.'
      return
    }
    emit('uploaded')
  } catch {
    error.value = 'Foto non valida.'
  } finally {
    uploading.value = false
  }
}
</script>

<template>
  <div class="gallery">
    <p v-if="!photos.length" class="empty">
      Ancora nessuna foto. Scatta la prima del torneo! 📸
    </p>
    <div v-else class="grid">
      <button
        v-for="p in photos" :key="p.id" class="thumb" @click="viewer = p.id"
        aria-label="Apri foto"
      >
        <img :src="imgUrl(p.id)" alt="Foto del torneo" loading="lazy" />
      </button>
    </div>

    <!-- Pill acceso: scatta e pubblica -->
    <div class="shoot-bar">
      <button class="shoot" :disabled="uploading" @click="pickPhoto">
        <span class="ico">📷</span>{{ uploading ? 'PUBBLICO…' : 'SCATTA FOTO' }}
      </button>
      <p v-if="error" class="err">{{ error }}</p>
    </div>

    <input
      ref="fileInput" class="hidden-file" type="file"
      accept="image/*" capture="environment" @change="onPhotoPick"
    />

    <!-- Visore a tutto schermo -->
    <div v-if="viewer" class="viewer" @click="viewer = null">
      <button class="close" aria-label="Chiudi" @click="viewer = null">✕</button>
      <img :src="imgUrl(viewer)" alt="Foto del torneo" />
    </div>
  </div>
</template>

<style scoped>
.gallery { padding-bottom: 92px; } /* spazio per la shoot-bar fissa */
.empty { color: rgba(255,255,255,.55); text-align: center; padding: 40px 20px; font-size: 14px; }

.grid {
  display: grid; grid-template-columns: repeat(3, 1fr); gap: 4px;
}
.thumb {
  padding: 0; border: none; background: #15151b; cursor: pointer;
  aspect-ratio: 1; overflow: hidden; border-radius: 4px;
}
.thumb img { width: 100%; height: 100%; object-fit: cover; display: block; }

/* Pill "SCATTA FOTO": fissa in basso, colore acceso, sopra la safe-area */
.shoot-bar {
  position: fixed; left: 0; right: 0; z-index: 6;
  bottom: 0; display: flex; flex-direction: column; align-items: center; gap: 6px;
  padding: 12px 16px calc(14px + env(safe-area-inset-bottom));
  background: linear-gradient(180deg, rgba(10,10,14,0), rgba(10,10,14,.92) 40%);
  pointer-events: none;
}
.shoot {
  pointer-events: auto;
  display: inline-flex; align-items: center; gap: 10px;
  border: none; border-radius: 999px; cursor: pointer;
  padding: 15px 34px; font-size: 16px; font-weight: 900; letter-spacing: .6px;
  color: #14110A;
  background: linear-gradient(135deg, #FFD23F, #FF7A18);
  box-shadow: 0 8px 22px rgba(255,122,24,.45);
}
.shoot:disabled { opacity: .7; cursor: default; }
.shoot .ico { font-size: 18px; }
.err { pointer-events: auto; color: #fca5a5; font-size: 12px; margin: 0; }
.hidden-file { display: none; }

/* Visore */
.viewer {
  position: fixed; inset: 0; z-index: 20; background: rgba(0,0,0,.94);
  display: flex; align-items: center; justify-content: center; padding: 16px;
}
.viewer img { max-width: 100%; max-height: 100%; object-fit: contain; border-radius: 6px; }
.close {
  position: absolute; top: calc(env(safe-area-inset-top,0px) + 12px); right: 14px;
  width: 40px; height: 40px; border-radius: 50%; border: none; cursor: pointer;
  background: rgba(255,255,255,.14); color: #fff; font-size: 18px;
}
</style>
