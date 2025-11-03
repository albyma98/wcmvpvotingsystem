import { createApp } from 'vue';
import PrimeVue from 'primevue/config';
import Aura from 'primevue/themes/aura';
import ToastService from 'primevue/toastservice';
import './style.css';
import 'primeicons/primeicons.css';
import 'primeflex/primeflex.css';
import App from './App.vue';
import { QrcodeStream, QrcodeDropZone, QrcodeCapture } from 'vue-qrcode-reader';

const app = createApp(App);
app.use(PrimeVue, {
  theme: {
    preset: Aura,
    options: {
      darkModeSelector: 'body.dark-mode',
    },
  },
});
app.use(ToastService);

app.component('QrcodeStream', QrcodeStream);
app.component('QrcodeDropZone', QrcodeDropZone);
app.component('QrcodeCapture', QrcodeCapture);

app.mount('#app');
