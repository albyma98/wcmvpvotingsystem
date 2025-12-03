import { createApp } from 'vue';
import { MotionPlugin } from '@vueuse/motion';
import './style.css';
import App from './App.vue';
import { QrcodeStream, QrcodeDropZone, QrcodeCapture } from 'vue-qrcode-reader'

const app = createApp(App)
app.use(MotionPlugin)
app.component('QrcodeStream', QrcodeStream)
app.component('QrcodeDropZone', QrcodeDropZone)
app.component('QrcodeCapture', QrcodeCapture)
app.mount('#app');