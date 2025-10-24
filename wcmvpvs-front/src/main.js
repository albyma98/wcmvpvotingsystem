import { createApp } from 'vue';
import './style.css';
import App from './App.vue';
import VueQrcodeReader from 'vue-qrcode-reader';

const app = createApp(App);
app.use(VueQrcodeReader);
app.mount('#app');