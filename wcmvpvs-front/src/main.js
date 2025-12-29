import { createApp } from 'vue';
import PrimeVue from 'primevue/config';
import 'primevue/resources/themes/aura-light-noir/theme.css';
import 'primevue/resources/primevue.min.css';
import 'primeicons/primeicons.css';
import './style.css';
import App from './App.vue';
import { QrcodeStream, QrcodeDropZone, QrcodeCapture } from 'vue-qrcode-reader';

const app = createApp(App);
app.use(PrimeVue);
app.component('QrcodeStream', QrcodeStream);
app.component('QrcodeDropZone', QrcodeDropZone);
app.component('QrcodeCapture', QrcodeCapture);
app.mount('#app');
