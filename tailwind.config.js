module.exports = {
  content: ["./templates/tmpls/*"],
  theme: {
    screens: {
      sm:'480px',
      md: '768px',
      lg: '976px',
      xl: '1440px'
    },
    extend: {
      colors:{
        light: '#e7eaf6',
        lighter: '#a2a8d3',
        darker: '#38598b',
        dark: '#113f67',

        dc: {
          900: '#202225',
          800: '#2f3136',
          700: '#36393f',
          600: '#4f545c',
          400: '#d4d7dc',
          300: '#e3e5e8',
          200: '#ebedef',
          100: '#f2f3f5',
        },
      }
    },
  },
  plugins: [],
}
