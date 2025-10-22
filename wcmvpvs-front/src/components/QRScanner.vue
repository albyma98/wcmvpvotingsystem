<template>
  <div class="container center-align">
    <h4>Scansione QR Code</h4>
    <video
      ref="video"
      playsinline
      muted
      class="responsive-video"
      @click="startScanner"
    ></video>
    <p v-if="scannedCode" class="green-text text-darken-2">
      Codice rilevato: {{ scannedCode }}
    </p>
    <p v-if="errorMessage" class="error red-text text-darken-2">
      {{ errorMessage }}
    </p>
  </div>
</template>

<script setup>
import { onUnmounted, ref } from 'vue'
import { BrowserMultiFormatReader, NotFoundException } from '@zxing/browser'

const video = ref(null)
const scannedCode = ref('')
const errorMessage = ref('')
const isScannerActive = ref(false)

const supportsBarcodeDetector = typeof window !== 'undefined' && 'BarcodeDetector' in window

let stream
let detector
let animationFrameId
let zxingReader
let zxingControls

async function startScanner() {
  if (isScannerActive.value) {
    return
  }
  errorMessage.value = ''
  scannedCode.value = ''

  if (typeof navigator === 'undefined' || !navigator.mediaDevices?.getUserMedia) {
    errorMessage.value = 'Questo dispositivo non supporta la scansione'
    return
  }

  try {
    if (supportsBarcodeDetector) {
      detector = detector || new window.BarcodeDetector({ formats: ['qr_code'] })
      stream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: { ideal: 'environment' } },
      })
      if (video.value) {
        video.value.srcObject = stream
        await video.value.play().catch(() => {})
      }
      isScannerActive.value = true
      animationFrameId = requestAnimationFrame(scanFrame)
    } else {
      if (!zxingReader) {
        zxingReader = new BrowserMultiFormatReader()
      }
      zxingControls = await zxingReader.decodeFromConstraints(
        { audio: false, video: { facingMode: { ideal: 'environment' } } },
        video.value,
        (result, err) => {
          if (result) {
            const text = result.getText()
            if (text) {
              stopScanner()
              scannedCode.value = text
            }
          }
          if (err && !(err instanceof NotFoundException)) {
            console.error('ZXing error', err)
            errorMessage.value = 'Errore durante la scansione'
          }
        },
      )
      if (video.value) {
        await video.value.play().catch(() => {})
      }
      isScannerActive.value = true
    }
  } catch (error) {
    console.error('Impossibile accedere alla fotocamera', error)
    errorMessage.value = 'Impossibile accedere alla fotocamera'
    stopScanner()
  }
}

function stopScanner() {
  if (animationFrameId) {
    cancelAnimationFrame(animationFrameId)
    animationFrameId = undefined
  }
  if (zxingControls) {
    zxingControls.stop()
    zxingControls = undefined
  }
  if (stream) {
    stream.getTracks().forEach(track => track.stop())
    stream = undefined
  }
  if (video.value) {
    const mediaStream = video.value.srcObject
    if (mediaStream) {
      mediaStream.getTracks().forEach(track => track.stop())
    }
    video.value.pause?.()
    video.value.srcObject = null
  }
  if (zxingReader) {
    zxingReader.reset()
  }
  isScannerActive.value = false
}

async function scanFrame() {
  if (!isScannerActive.value || !detector || !video.value) {
    return
  }
  if (video.value.readyState >= HTMLMediaElement.HAVE_ENOUGH_DATA) {
    try {
      const barcodes = await detector.detect(video.value)
      if (barcodes.length > 0) {
        stopScanner()
        scannedCode.value = barcodes[0].rawValue || ''
        return
      }
    } catch (error) {
      console.error('Barcode detection error', error)
    }
  }
  animationFrameId = requestAnimationFrame(scanFrame)
}

onUnmounted(() => {
  stopScanner()
})
</script>

<style scoped>
video {
  width: 100%;
  max-width: 300px;
  border: 1px solid #ccc;
  border-radius: 8px;
}
.error {
  margin-top: 0.5rem;
}
</style>
