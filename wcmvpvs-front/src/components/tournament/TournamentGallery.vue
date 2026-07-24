<script setup>
// Gallery del torneo: le miniature delle foto pubblicate dai tifosi + un pill
// acceso "SCATTA FOTO" che apre la fotocamera e pubblica. Le nuove foto degli
// altri compaiono da sole (SSE, gestito dal parent che ricarica su tick).
import { ref } from 'vue'
import { track as posthogTrack, EVENTS as PH_EVENTS } from '@/lib/track'

const props = defineProps({
  slug: { type: String, required: true },
  photos: { type: Array, default: () => [] },
  started: { type: Boolean, default: true }
})
const emit = defineEmits(['uploaded'])

const uploading = ref(false)
const error = ref('')
const viewer = ref(null)        // id della foto aperta a tutto schermo
const viewerLoading = ref(false) // la full si sta caricando nel visore
const fileInput = ref(null)
const showConsent = ref(false)  // avviso liberatoria prima di aprire la fotocamera

// Griglia = miniatura leggera (veloce anche con decine di foto); visore = full.
const thumbUrl = id => `/api/v1/tournaments/${props.slug}/gallery/${id}/thumb`
const imgUrl = id => `/api/v1/tournaments/${props.slug}/gallery/${id}/image`

// Genera dalla foto scelta due versioni (una decodifica, due render):
//  - full: qualità alta per l'ingrandimento (lato lungo 2560px, così resta sotto
//    il limite di 6MB del server senza rovinare la qualità sugli schermi phone)
//  - thumb: miniatura piccola per la griglia (lato lungo 400px)
// WebP quando supportato, JPEG di fallback (per le foto va benissimo).
function makeImages (file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    const img = new Image()
    reader.onload = () => { img.src = reader.result }
    reader.onerror = () => reject(new Error('read'))
    img.onerror = () => reject(new Error('decode'))
    img.onload = () => {
      const render = (maxSize, quality) => {
        const scale = Math.min(1, maxSize / Math.max(img.width, img.height))
        const w = Math.max(1, Math.round(img.width * scale))
        const h = Math.max(1, Math.round(img.height * scale))
        const canvas = document.createElement('canvas')
        canvas.width = w; canvas.height = h
        canvas.getContext('2d').drawImage(img, 0, 0, w, h)
        let out = canvas.toDataURL('image/webp', quality)
        if (!out.startsWith('data:image/webp')) out = canvas.toDataURL('image/jpeg', quality)
        return out
      }
      resolve({ full: render(2560, 0.9), thumb: render(400, 0.72) })
    }
    reader.readAsDataURL(file)
  })
}

function openViewer (id) {
  viewer.value = id; viewerLoading.value = true
  posthogTrack(PH_EVENTS.TOURNAMENT_GALLERY_PHOTO_OPENED, {
    tournament_slug: props.slug,
    surface: 'tournament',
    photo_id: id,
  })
}

// Prima di aprire la fotocamera mostra la liberatoria: premere "SCATTA FOTO"
// nell'avviso vale come accettazione (e apre la fotocamera nel gesto utente).
function pickPhoto () {
  if (!props.started) return
  showConsent.value = true
}
function cancelShoot () { showConsent.value = false }
function confirmShoot () {
  showConsent.value = false
  fileInput.value?.click()
}

async function onPhotoPick (e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file) return
  if (!file.type.startsWith('image/')) { error.value = 'Seleziona un\'immagine.'; return }
  uploading.value = true
  error.value = ''
  try {
    const { full, thumb } = await makeImages(file)
    const res = await fetch(`/api/v1/tournaments/${props.slug}/gallery`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ image: full, thumb })
    })
    if (!res.ok) {
      const err = (await res.json().catch(() => ({}))).error
      if (err === 'tournament_not_started') error.value = 'Il torneo non è ancora iniziato.'
      else error.value = err === 'image_too_large' ? 'Foto troppo pesante.' : 'Pubblicazione non riuscita.'
      return
    }
    posthogTrack(PH_EVENTS.TOURNAMENT_GALLERY_UPLOAD, {
      tournament_slug: props.slug,
      surface: 'tournament',
    })
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
      {{ started ? 'Ancora nessuna foto. Scatta la prima del torneo! 📸' : 'Le foto saranno disponibili quando inizierà il torneo.' }}
    </p>
    <div v-else class="grid">
      <button
        v-for="p in photos" :key="p.id" class="thumb" @click="openViewer(p.id)"
        aria-label="Apri foto"
      >
        <img :src="thumbUrl(p.id)" alt="Foto del torneo" loading="lazy" />
      </button>
    </div>

    <!-- Pill acceso: scatta e pubblica -->
    <div class="shoot-bar">
      <p v-if="!started" class="locked">🔒 Il torneo non è ancora iniziato</p>
      <button class="shoot" :disabled="uploading || !started" @click="pickPhoto">
        <span class="ico">📷</span>{{ uploading ? 'PUBBLICO…' : 'SCATTA FOTO' }}
      </button>
      <p v-if="error" class="err">{{ error }}</p>
    </div>

    <input
      ref="fileInput" class="hidden-file" type="file"
      accept="image/*" capture="environment" @change="onPhotoPick"
    />

    <!-- Liberatoria: consenso prima di scattare -->
    <div v-if="showConsent" class="consent-scrim" @click.self="cancelShoot">
      <div class="consent-card" role="dialog" aria-modal="true">
        <p class="consent-text">
          <span class="consent-check" aria-hidden="true">☑</span>
          Dichiaro di avere il diritto di pubblicare questa foto e di aver ottenuto, ove necessario,
          il consenso delle persone riconoscibili presenti nell'immagine. Autorizzo gli organizzatori
          a mostrare la foto all'interno della piattaforma e nelle pagine social del torneo.
        </p>
        <div class="consent-actions">
          <button class="consent-later" @click="cancelShoot">Non ora</button>
          <button class="consent-shoot" @click="confirmShoot">📷 SCATTA FOTO</button>
        </div>
      </div>
    </div>

    <!-- Visore a tutto schermo: miniatura sfocata subito, poi la full nitida -->
    <div v-if="viewer" class="viewer" @click="viewer = null">
      <button class="close" aria-label="Chiudi" @click="viewer = null">✕</button>
      <div class="stage" @click.stop>
        <img class="blur" :src="thumbUrl(viewer)" alt="" aria-hidden="true" />
        <img
          class="full" :class="{ ready: !viewerLoading }" :src="imgUrl(viewer)"
          alt="Foto del torneo" @load="viewerLoading = false"
        />
        <span v-if="viewerLoading" class="spinner" aria-label="Caricamento"></span>
      </div>
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
.locked { pointer-events: auto; color: #FFD66B; font-size: 12px; font-weight: 800; margin: 0; }
.err { pointer-events: auto; color: #fca5a5; font-size: 12px; margin: 0; }
.hidden-file { display: none; }

/* Liberatoria (consenso pubblicazione foto) */
.consent-scrim {
  position: fixed; inset: 0; z-index: 30; background: rgba(0,0,0,.72);
  display: flex; align-items: center; justify-content: center; padding: 20px;
  backdrop-filter: blur(2px);
}
.consent-card {
  width: 100%; max-width: 380px; background: #17171F;
  border: 1px solid rgba(255,255,255,.1); border-radius: 16px;
  padding: 20px 18px 16px; box-shadow: 0 18px 44px rgba(0,0,0,.5);
}
.consent-text {
  margin: 0 0 16px; font-size: 13px; line-height: 1.55;
  color: rgba(255,255,255,.82); font-weight: 500;
}
.consent-check { color: #00C897; font-weight: 900; margin-right: 4px; }
.consent-actions { display: flex; gap: 10px; justify-content: flex-end; align-items: center; }
.consent-later {
  border: none; background: transparent; color: rgba(255,255,255,.6);
  font-size: 14px; font-weight: 700; padding: 10px 14px; cursor: pointer;
}
.consent-shoot {
  border: none; border-radius: 999px; cursor: pointer;
  padding: 12px 22px; font-size: 14.5px; font-weight: 900; letter-spacing: .4px;
  color: #14110A; background: linear-gradient(135deg, #FFD23F, #FF7A18);
  box-shadow: 0 6px 16px rgba(255,122,24,.4);
}

/* Visore */
.viewer {
  position: fixed; inset: 0; z-index: 20; background: rgba(0,0,0,.94);
  display: flex; align-items: center; justify-content: center; padding: 16px;
}
.stage { position: relative; max-width: 100%; max-height: 100%; display: flex; }
.stage img { max-width: 100%; max-height: calc(100dvh - 32px); object-fit: contain; border-radius: 6px; display: block; }
/* miniatura ingrandita = placeholder istantaneo, leggermente sfocato */
.stage .blur { position: absolute; inset: 0; width: 100%; height: 100%; filter: blur(14px); transform: scale(1.02); }
/* la full sopra: appare in dissolvenza quando è caricata */
.stage .full { position: relative; opacity: 0; transition: opacity .25s ease; }
.stage .full.ready { opacity: 1; }
.spinner {
  position: absolute; top: 50%; left: 50%; width: 34px; height: 34px; margin: -17px 0 0 -17px;
  border: 3px solid rgba(255,255,255,.25); border-top-color: #fff; border-radius: 50%;
  animation: gal-spin .8s linear infinite;
}
@keyframes gal-spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { .spinner { animation: none; } }
.close {
  position: absolute; top: calc(env(safe-area-inset-top,0px) + 12px); right: 14px;
  width: 40px; height: 40px; border-radius: 50%; border: none; cursor: pointer;
  background: rgba(255,255,255,.14); color: #fff; font-size: 18px;
}
</style>
