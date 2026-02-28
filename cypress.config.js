const { defineConfig } = require('cypress')

module.exports = defineConfig({
    e2e: {
        baseUrl: 'http://app:8080',
        specPattern: 'cypress/e2e/**/*.cy.js',
        supportFile: false,

    },
});
