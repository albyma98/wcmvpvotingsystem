import 'vuetify/styles';
import { createVuetify } from 'vuetify';
import { aliases, mdi } from 'vuetify/iconsets/mdi';

const light = {
  dark: false,
  colors: {
    background: '#f6f8fb',
    surface: '#ffffff',
    primary: '#1976d2',
  },
};

const dark = {
  dark: true,
  colors: {
    background: '#0f172a',
    surface: '#0b1324',
    primary: '#7cacf8',
  },
};

export default createVuetify({
  theme: {
    defaultTheme: 'dark',
    themes: { light, dark },
  },
  icons: {
    defaultSet: 'mdi',
    aliases,
    sets: { mdi },
  },
});
