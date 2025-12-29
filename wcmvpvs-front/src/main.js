import { createApp } from 'vue';
import './style.css';
import App from './App.vue';
import { QrcodeStream, QrcodeDropZone, QrcodeCapture } from 'vue-qrcode-reader';
import { createRouter } from './router';
import { routes } from './router/routes';

const router = createRouter({ routes });

const app = createApp(App);
app.component('QrcodeStream', QrcodeStream);
app.component('QrcodeDropZone', QrcodeDropZone);
app.component('QrcodeCapture', QrcodeCapture);
app.use(router);
app.mount('#app');
