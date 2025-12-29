import { createApp } from 'vue';
import './style.css';
import '@mdi/font/css/materialdesignicons.css';
import App from './App.vue';
import vuetify from './plugins/vuetify';
import { QrcodeStream, QrcodeDropZone, QrcodeCapture } from 'vue-qrcode-reader';

const app = createApp(App);
app.use(vuetify);
app.component('QrcodeStream', QrcodeStream);
app.component('QrcodeDropZone', QrcodeDropZone);
app.component('QrcodeCapture', QrcodeCapture);
app.mount('#app');
