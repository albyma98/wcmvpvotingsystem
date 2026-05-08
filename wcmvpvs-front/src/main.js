import { createApp } from 'vue';
import './style.css';
import './assets/admin-theme.css';
import App from './App.vue';
import { QrcodeStream, QrcodeDropZone, QrcodeCapture } from 'vue-qrcode-reader'
import { initPostHog } from './lib/posthog';

initPostHog();

const app = createApp(App)
app.component('QrcodeStream', QrcodeStream)
app.component('QrcodeDropZone', QrcodeDropZone)
app.component('QrcodeCapture', QrcodeCapture)
app.mount('#app');