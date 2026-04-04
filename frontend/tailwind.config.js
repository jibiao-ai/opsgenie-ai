/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          DEFAULT: '#513CC8',
          50: '#EEE9FB',
          100: '#DDD3F7',
          200: '#BBB0EF',
          300: '#998CE7',
          400: '#7769DF',
          500: '#513CC8',
          600: '#4230A6',
          700: '#322584',
          800: '#221962',
          900: '#120D40',
        },
      },
    },
  },
  plugins: [],
}
