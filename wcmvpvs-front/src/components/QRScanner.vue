<template>
  <div class="qr-scanner">
    <div class="qr-scanner__preview" :class="{ 'is-active': isActive }">
      <div :id="previewId" ref="preview" class="qr-scanner__camera"></div>
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
import { nextTick, onBeforeUnmount, ref } from 'vue'

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

const previewId = `qr-scanner-${Math.random().toString(36).slice(2, 10)}`
const preview = ref(null)
const isActive = ref(false)
const infoMessage = ref('Premi "Avvia scansione" per utilizzare la fotocamera.')
const errorMessage = ref('')

let html5QrCode = null
let startPromise = null
let stopPromise = null
let lastValue = ''

function setInfo(message) {
  infoMessage.value = message
}

function resetLastValue() {
  lastValue = ''
}

function ensureScanner() {
  if (html5QrCode) {
    return html5QrCode
  }
  if (typeof window === 'undefined' || !window.Html5Qrcode) {
    const err = new Error('html5_qrcode_unavailable')
    errorMessage.value = 'Libreria di scansione QR non disponibile nel browser.'
    setInfo('')
    emit('error', err)
    throw err
  }
  if (!preview.value) {
    const err = new Error('scanner_container_unavailable')
    emit('error', err)
    throw err
  }
  html5QrCode = new window.Html5Qrcode(previewId)
  return html5QrCode
}

function normalizeError(error) {
  if (!error) {
    return ''
  }
  if (typeof error === 'string') {
    return error.toLowerCase()
  }
  if (typeof error.message === 'string') {
    return error.message.toLowerCase()
  }
  if (typeof error.name === 'string') {
    return error.name.toLowerCase()
  }
  return String(error).toLowerCase()
}

function toError(error) {
  return error instanceof Error ? error : new Error(String(error || ''))
}

function handleStartError(error) {
  const normalized = normalizeError(error)
  const err = toError(error)

  if (normalized.includes('notallowederror') || normalized.includes('permission')) {
    errorMessage.value = 'Accesso alla fotocamera negato.'
    emit('permission-denied', err)
  } else if (normalized.includes('notfounderror') || normalized.includes('device') || normalized.includes('camera')) {
    errorMessage.value = 'Nessuna fotocamera disponibile sul dispositivo.'
    emit('permission-denied', err)
  } else {
    errorMessage.value = 'Impossibile avviare la fotocamera.'
    emit('error', err)
  }

  setInfo('')

  if (html5QrCode) {
    html5QrCode.stop?.().catch(() => {})
    html5QrCode.clear?.().catch(() => {})
    html5QrCode = null
  }
}

function handleScanSuccess(decodedText) {
  if (!decodedText || typeof decodedText !== 'string') {
    return
  }

  const value = decodedText.trim()
  if (!value || value === lastValue) {
    return
  }

  lastValue = value
  emit('detected', value)

  if (props.stopOnDetection) {
    setInfo('QR code rilevato.')
    stop({ silent: true }).catch(() => {})
  }
}

function handleScanFailure() {
  // Ignoriamo gli errori di scansione intermedi per evitare spam di notifiche.
}

async function start() {
  if (isActive.value || startPromise) {
    return startPromise
  }

  resetLastValue()
  errorMessage.value = ''
  setInfo('Attivazione della fotocamera…')

  startPromise = (async () => {
    await nextTick()
    const scanner = ensureScanner()

    const facingMode = props.facingMode || 'environment'
    const cameraConfig = { facingMode }

    const formats =
      typeof window !== 'undefined' && window.Html5QrcodeSupportedFormats
        ? window.Html5QrcodeSupportedFormats
        : null

    const configuration = {
      fps: 10,
      aspectRatio: 1.3333333333,
      disableFlip: facingMode === 'environment',
      qrbox(viewFinderWidth, viewFinderHeight) {
        const minEdge = Math.min(viewFinderWidth, viewFinderHeight)
        const size = Math.max(200, Math.round(minEdge * 0.65))
        return { width: size, height: size }
      },
    }

    if (formats && formats.QR_CODE) {
      configuration.formatsToSupport = [formats.QR_CODE]
    }

    try {
      await scanner.start(cameraConfig, configuration, handleScanSuccess, handleScanFailure)
      isActive.value = true
      setInfo('Inquadra il QR code del ticket.')
      emit('state-change', { active: true })
    } catch (error) {
      handleStartError(error)
      throw error
    }
  })()

  try {
    await startPromise
  } finally {
    startPromise = null
  }
}

async function stop({ silent = false } = {}) {
  if (stopPromise) {
    return stopPromise
  }

  stopPromise = (async () => {
    if (startPromise) {
      try {
        await startPromise
      } catch {
        // ignoriamo errori già gestiti nello start
      }
    }

    if (!html5QrCode) {
      if (isActive.value) {
        isActive.value = false
        emit('state-change', { active: false })
      }
      return
    }

    try {
      await html5QrCode.stop()
    } catch (error) {
      emit('error', toError(error))
    }

    try {
      await html5QrCode.clear()
    } catch (error) {
      emit('error', toError(error))
    }

    html5QrCode = null

    if (isActive.value) {
      isActive.value = false
      if (!silent) {
        setInfo('Scansione interrotta.')
      }
      emit('state-change', { active: false })
    }
  })()

  try {
    await stopPromise
  } finally {
    stopPromise = null
  }

  if (!isActive.value && !silent && infoMessage.value === '') {
    setInfo('Premi "Avvia scansione" per utilizzare la fotocamera.')
  }
}

function reset() {
  resetLastValue()
  errorMessage.value = ''
  if (!isActive.value) {
    setInfo('Premi "Avvia scansione" per utilizzare la fotocamera.')
  }
}

onBeforeUnmount(() => {
  stop({ silent: true }).catch(() => {})
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

.qr-scanner__camera {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
}

.qr-scanner__camera :deep(video),
.qr-scanner__camera :deep(canvas) {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: inherit;
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
  z-index: 2;
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
