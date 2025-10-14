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
        court: 'inset 0 18px 60px rgba(86, 52, 20, 0.32)',
        glow: '0 0 25px rgba(250, 204, 21, 0.65)',
      },
      colors: {
        court: {
          base: '#c7955b',
          dark: '#945f2d',
          light: '#e2bc84',
        },
      },
      backgroundImage: {
        'court-wood-planks':
          'repeating-linear-gradient(90deg, rgba(122, 81, 38, 0.28) 0, rgba(122, 81, 38, 0.28) 2px, transparent 2px, transparent 88px)',
        'court-wood-grain':
          'linear-gradient(140deg, rgba(255, 242, 224, 0.12) 0%, rgba(121, 70, 23, 0.18) 45%, rgba(0, 0, 0, 0.18) 100%)',
      },
    },
  },
  plugins: [],
};
