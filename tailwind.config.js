/** @type {import('tailwindcss').Config} */
const { addDynamicIconSelectors } = require('@iconify/tailwind');
module.exports = {
    content: [
        "templates/*.gohtml",
    ],
    theme: {
        extend: {},
    },
    plugins: [require("daisyui"), addDynamicIconSelectors(),],
}

