<template>
  <div class="container center-align">
    <h4>Scansione QR Code</h4>
    <video ref="video" autoplay playsinline class="responsive-video"></video>
    <p v-if="scannedCode" class="green-text text-darken-2">
      Codice rilevato: {{ scannedCode }}
    </p>
    <p v-if="errorMessage" class="error red-text text-darken-2">
      {{ errorMessage }}
    </p>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

const video = ref(null)
const scannedCode = ref('')
const errorMessage = ref('')

let stream
let detector

onMounted(async () => {
  if ('BarcodeDetector' in window) {
    detector = new BarcodeDetector({ formats: ['qr_code'] })
    try {
      stream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: 'environment' } })
      video.value.srcObject = stream
      requestAnimationFrame(scan)
    } catch (e) {
      errorMessage.value = 'Impossibile accedere alla fotocamera'
    }
  } else {
    errorMessage.value = 'BarcodeDetector API non supportata'
  }
})

onUnmounted(() => {
  if (stream) {
    stream.getTracks().forEach(t => t.stop())
  }
})

async function scan() {
  if (video.value && video.value.readyState === video.value.HAVE_ENOUGH_DATA) {
    try {
      const barcodes = await detector.detect(video.value)
      if (barcodes.length > 0) {
        scannedCode.value = barcodes[0].rawValue
      }
    } catch (e) {
      console.error(e)
    }
  }
  requestAnimationFrame(scan)
}
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
