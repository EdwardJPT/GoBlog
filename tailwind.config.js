/** @type {import('tailwindcss').Config} */
module.exports = {
    content: [ "./web/templates/**/*.html" ], // This is where your HTML templates
    theme: {
        extend: {
            colors: {
                background: "#100340",
                lighter_background: "#0a144b",
                title_color: "#CEDAE7",
                hover: "#283264",
            },
            fontSize: {
                title: "2.5rem",
            },
            fontFamily: {
                "sans": ["Montserrat", "ui-sans-serif", "system-ui", "sans-serif", "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol", "Noto Color Emoji"],
            },
            width: {
                content: "1200px",
                article: "650px",
                max_width: "1550px",
            },
            height: {
                blogOption: "550px",
            },
            letterSpacing: {
                title: "5px",
            },
        },
    },
    plugins: [],
};
