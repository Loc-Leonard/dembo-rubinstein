describe('Dembo-Rub survey flow',() => {
    it('fills all sliders and shows result page', () => {
        cy.visit('/');

        cy.get('#submit-btn').should('be.disabled');

        const scales = ['health', 'mind', 'character', 'authority', 'hands', 'appearance', 'confidence'];

        scales.forEach(id => {
            cy.get(`#${id}-now`)
              .invoke('val', 50)
              .trigger('input');

            cy.get(`#${id}-ideal`)
              .invoke('val', 70)
              .trigger('input');
        });

        cy.get('#submit-btn').should('not.be.disabled');
        cy.get('#submit-btn').click();
        cy.url().should('match', /\/result\?v=.*/);
        cy.get('h1').contains('Результаты самооценки по методике Дембо-Рубинштейн').should('be.visible');
        cy.get('table.results-table').should('be.visible');
    });
});