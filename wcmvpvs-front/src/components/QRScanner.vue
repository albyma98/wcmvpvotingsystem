<template>
  <div class="qr-scanner">
    <div class="qr-scanner__preview" :class="{ 'is-active': isActive }">
      <QrcodeStream
        v-if="isVisible"
        class="qr-scanner__camera"
        camera="rear"
        :track="drawOutline"
        :constraints="constraints"
        @decode="handleDecode"
        @init="handleInit"
        @error="handleStreamError"
        @camera-on="handleCameraOn"
        @camera-off="handleCameraOff"
      />
      <div v-else class="qr-scanner__placeholder">
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
import { computed, onBeforeUnmount, ref } from 'vue'
import { QrcodeStream } from 'vue-qrcode-reader'

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

const defaultInfoMessage = 'Premi "Avvia scansione" per utilizzare la fotocamera.'
const isVisible = ref(false)
const isActive = ref(false)
const infoMessage = ref(defaultInfoMessage)
const errorMessage = ref('')

let startPromise = null
let startResolve = null
let startReject = null
let lastValue = ''
let scanningFeedbackTimeout = null

function setInfo(message) {
  infoMessage.value = message
}

function resetLastValue() {
  lastValue = ''
}

function clearScanningFeedback() {
  if (scanningFeedbackTimeout) {
    clearTimeout(scanningFeedbackTimeout)
    scanningFeedbackTimeout = null
  }
}

const constraints = computed(() => {
  const desiredFacingMode = props.facingMode === 'user' ? 'user' : 'environment'
  return {
    audio: false,
    video: {
      facingMode: {
        ideal: desiredFacingMode,
      },
    },
  }
})

function normalizeError(error) {
  if (!error) {
    return ''
  }
  if (typeof error === 'string') {
    return error.toLowerCase()
  }
  if (typeof error?.message === 'string') {
    return error.message.toLowerCase()
  }
  if (typeof error?.name === 'string') {
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

  const isPermissionError =
    normalized.includes('notallowederror') || normalized.includes('permission')
  const isDeviceError =
    normalized.includes('notfounderror') ||
    normalized.includes('device') ||
    normalized.includes('camera')

  let eventEmitted = false

  if (isPermissionError) {
    errorMessage.value = 'Accesso alla fotocamera negato.'
    emit('permission-denied', err)
    eventEmitted = true
  } else if (isDeviceError) {
    errorMessage.value = 'Nessuna fotocamera disponibile sul dispositivo.'
    emit('permission-denied', err)
    eventEmitted = true
  } else {
    errorMessage.value = 'Impossibile avviare la fotocamera.'
  }

  setInfo('')

  const wasActive = isActive.value
  isVisible.value = false
  isActive.value = false
  resetLastValue()
  clearScanningFeedback()

  if (wasActive) {
    emit('state-change', { active: false })
  }

  if (!eventEmitted) {
    emit('error', err)
  }

  if (startReject) {
    startReject(err)
  }

  startPromise = null
  startResolve = null
  startReject = null
}

function handleDecode(value) {
  if (typeof value !== 'string') {
    return
  }

  const normalizedValue = value.trim()
  if (!normalizedValue || normalizedValue === lastValue) {
    return
  }

  lastValue = normalizedValue
  clearScanningFeedback()
  emit('detected', normalizedValue)

  if (props.stopOnDetection) {
    setInfo('QR code rilevato.')
    stop({ silent: true }).catch(() => {})
  }
}

async function handleInit(promise) {
  try {
    await promise
  } catch (error) {
    handleStartError(error)
  }
}

function handleStreamError(error) {
  handleStartError(error)
}

async function start() {
  if (startPromise) {
    return startPromise
  }

  if (isActive.value) {
    return
  }

  resetLastValue()
  errorMessage.value = ''
  setInfo('Attivazione della fotocamera…')

  startPromise = new Promise((resolve, reject) => {
    startResolve = resolve
    startReject = reject
  })

  isVisible.value = true

  try {
    await startPromise
  } finally {
    startPromise = null
    startResolve = null
    startReject = null
  }
}

async function stop({ silent = false } = {}) {
  if (startPromise && startReject) {
    startReject(new Error('scanner_start_aborted'))
    startPromise = null
    startResolve = null
    startReject = null
  }

  if (!isVisible.value && !isActive.value) {
    if (!silent && infoMessage.value === '') {
      setInfo(defaultInfoMessage)
    }
    return
  }

  const wasActive = isActive.value

  isVisible.value = false
  isActive.value = false
  resetLastValue()
  clearScanningFeedback()

  if (!silent) {
    setInfo('Scansione interrotta.')
  }

  if (wasActive) {
    emit('state-change', { active: false })
  }

  if (!isActive.value && !silent && infoMessage.value === '') {
    setInfo(defaultInfoMessage)
  }
}

function reset() {
  resetLastValue()
  errorMessage.value = ''
  if (!isActive.value) {
    setInfo(defaultInfoMessage)
  }
  clearScanningFeedback()
}

onBeforeUnmount(() => {
  stop({ silent: true }).catch(() => {})
  clearScanningFeedback()
})

function handleCameraOn() {
  isActive.value = true
  errorMessage.value = ''
  setInfo('Scansione in corso… inquadra il QR code del ticket.')
  emit('state-change', { active: true })

  clearScanningFeedback()
  scanningFeedbackTimeout = setTimeout(() => {
    if (isActive.value && !lastValue) {
      setInfo("Nessun QR rilevato ancora. Prova ad avvicinarti o migliora l'illuminazione.")
    }
  }, 4000)

  if (startResolve) {
    startResolve()
  }
}

function handleCameraOff() {
  const wasActive = isActive.value
  isActive.value = false
  clearScanningFeedback()

  if (wasActive) {
    emit('state-change', { active: false })
  }

  if (!isVisible.value && infoMessage.value === '') {
    setInfo(defaultInfoMessage)
  }
}

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

.qr-scanner__camera :deep(video) {
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


