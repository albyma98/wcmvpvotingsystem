/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './index.html',
    './src/**/*.{vue,js,ts,jsx,tsx}',
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ['"Poppins"', 'system-ui', 'sans-serif'],
      },
      boxShadow: {
        pitch: 'inset 0 8px 40px rgba(15, 118, 110, 0.35)',
        glow: '0 0 25px rgba(250, 204, 21, 0.65)',
      },
      colors: {
        pitch: {
          base: '#0b5d40',
          dark: '#084531',
          light: '#127a56',
        },
      },
      backgroundImage: {
        'court-grid': 'repeating-linear-gradient(0deg, rgba(255,255,255,0.06) 0, rgba(255,255,255,0.06) 6%, transparent 6%, transparent 12%)',
      },
    },
  },
  plugins: [],
};
