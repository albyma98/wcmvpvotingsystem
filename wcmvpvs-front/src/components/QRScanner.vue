<template>
  <div class="qr-scanner">
    <div class="qr-scanner__preview" :class="{ 'is-active': isActive }">
      <video ref="video" autoplay playsinline muted></video>
      <div v-if="!isActive" class="qr-scanner__placeholder">
        <slot name="placeholder">
          <span>Premi "Avvia scansione" per utilizzare la fotocamera.</span>
        </slot>
      </div>
    </div>
    <p v-if="infoMessage" class="qr-scanner__info">{{ infoMessage }}</p>
    <p v-if="errorMessage" class="qr-scanner__error">{{ errorMessage }}</p>
  </div>
</template>

<script setup>
import { onBeforeUnmount, ref } from 'vue'

const props = defineProps({
  facingMode: {
    type: String,
    default: 'environment',
  },
  stopOnDetection: {
    type: Boolean,
    default: true,
  },
})

const emit = defineEmits(['detected', 'error', 'permission-denied', 'state-change'])

const video = ref(null)
const isActive = ref(false)
const infoMessage = ref('Premi "Avvia scansione" per utilizzare la fotocamera.')
const errorMessage = ref('')

let detector
let stream
let frameHandle = 0
let lastValue = ''

function setInfo(message) {
  infoMessage.value = message
}

function resetLastValue() {
  lastValue = ''
}

function stopStream() {
  if (stream) {
    stream.getTracks().forEach((track) => track.stop())
    stream = null
  }
}

async function ensureDetector() {
  if (detector) {
    return detector
  }
  if (!('BarcodeDetector' in window)) {
    const err = new Error('barcode_detector_unsupported')
    errorMessage.value = 'BarcodeDetector API non supportata dal browser.'
    setInfo('')
    emit('error', err)
    throw err
  }
  detector = new BarcodeDetector({ formats: ['qr_code'] })
  return detector
}

async function start() {
  if (isActive.value) {
    return
  }

  resetLastValue()
  errorMessage.value = ''
  setInfo('Attivazione della fotocamera…')

  try {
    await ensureDetector()
  } catch (err) {
    throw err
  }

  try {
    stream = await navigator.mediaDevices.getUserMedia({
      video: {
        facingMode: props.facingMode || 'environment',
      },
    })
  } catch (err) {
    if (err && err.name === 'NotAllowedError') {
      errorMessage.value = 'Accesso alla fotocamera negato.'
      emit('permission-denied', err)
    } else if (err && err.name === 'NotFoundError') {
      errorMessage.value = 'Nessuna fotocamera disponibile sul dispositivo.'
      emit('permission-denied', err)
    } else {
      errorMessage.value = 'Impossibile accedere alla fotocamera.'
      emit('error', err)
    }
    setInfo('')
    stopStream()
    throw err
  }

  if (!video.value) {
    stopStream()
    setInfo('')
    return
  }

  video.value.srcObject = stream
  isActive.value = true
  setInfo('Inquadra il QR code del ticket.')
  emit('state-change', { active: true })
  frameHandle = requestAnimationFrame(scan)
}

function stop() {
  if (frameHandle) {
    cancelAnimationFrame(frameHandle)
    frameHandle = 0
  }
  if (!isActive.value && !stream) {
    return
  }
  stopStream()
  if (isActive.value) {
    setInfo('Scansione interrotta.')
  }
  isActive.value = false
  emit('state-change', { active: false })
}

function reset() {
  resetLastValue()
  errorMessage.value = ''
  if (!isActive.value) {
    setInfo('Premi "Avvia scansione" per utilizzare la fotocamera.')
  }
}

async function scan() {
  if (!isActive.value || !video.value || !detector) {
    return
  }

  try {
    const barcodes = await detector.detect(video.value)
    if (Array.isArray(barcodes) && barcodes.length > 0) {
      const rawValue = (barcodes[0].rawValue || '').trim()
      if (rawValue && rawValue !== lastValue) {
        lastValue = rawValue
        emit('detected', rawValue)
        if (props.stopOnDetection) {
          setInfo('QR code rilevato.')
          stop()
          return
        }
      }
    }
  } catch (err) {
    errorMessage.value = 'Errore durante la scansione del QR code.'
    emit('error', err)
  }

  frameHandle = requestAnimationFrame(scan)
}

onBeforeUnmount(() => {
  stop()
})

defineExpose({
  start,
  stop,
  reset,
  isActive,
})
</script>

<style scoped>
.qr-scanner {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  align-items: center;
}

.qr-scanner__preview {
  position: relative;
  width: 100%;
  max-width: 360px;
  aspect-ratio: 3 / 4;
  border-radius: 1rem;
  background: rgba(15, 23, 42, 0.1);
  overflow: hidden;
  border: 2px solid rgba(59, 130, 246, 0.25);
  display: flex;
  align-items: center;
  justify-content: center;
}

.qr-scanner__preview video {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.qr-scanner__preview.is-active {
  border-color: rgba(34, 197, 94, 0.45);
  box-shadow: 0 12px 32px rgba(34, 197, 94, 0.18);
}

.qr-scanner__placeholder {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.25rem;
  text-align: center;
  background: linear-gradient(135deg, rgba(15, 23, 42, 0.75), rgba(59, 130, 246, 0.45));
  color: #f8fafc;
  font-weight: 600;
}

.qr-scanner__info {
  margin: 0;
  font-size: 0.95rem;
  color: #1e293b;
  text-align: center;
}

.qr-scanner__error {
  margin: 0;
  font-size: 0.95rem;
  color: #dc2626;
  text-align: center;
}
</style>
