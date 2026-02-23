class ResultsViewer {
    constructor() {
        this.init();
    }

    init() {
        // Проверяем вариант А (данные в hash)
        if (window.location.hash) {
            const hashData = window.location.hash.substring(1); // убираем #
            const params = new URLSearchParams(hashData);
            
            if (params.has('d')) {
                this.decodeAndShowResults(params.get('d'));
                return;
            }
        }

        // Проверяем вариант Б (данные от сервера)
        const urlParams = new URLSearchParams(window.location.search);
        if (urlParams.has('v')) {
            // Данные уже отрендерены сервером
            document.querySelector('.loading').style.display = 'none';
            return;
        }

        // Если ничего нет
        this.showError('Результаты не найдены');
    }

    decodeAndShowResults(encodedData) {
        try {
            // Декодируем base64
            let base64 = encodedData.replace(/-/g, '+').replace(/_/g, '/');
            while (base64.length % 4) {
                base64 += '=';
            }
            
            const jsonStr = atob(base64);
            const data = JSON.parse(jsonStr);

            // Рассчитываем результаты
            const results = this.calculateResults(data);
            
            // Отображаем
            this.renderResults(results);
        } catch (error) {
            console.error('Ошибка декодирования:', error);
            this.showError('Ошибка при загрузке результатов');
        }
    }

    calculateResults(data) {
        const scales = [
            { name: 'Здоровье', now: data.health_now, ideal: data.health_ideal },
            { name: 'Ум/способности', now: data.mind_now, ideal: data.mind_ideal },
            { name: 'Характер', now: data.character_now, ideal: data.character_ideal },
            { name: 'Авторитет у сверстников', now: data.authority_now, ideal: data.authority_ideal },
            { name: 'Умелые руки', now: data.hands_now, ideal: data.hands_ideal },
            { name: 'Внешность', now: data.appearance_now, ideal: data.appearance_ideal },
            { name: 'Уверенность в себе', now: data.confidence_now, ideal: data.confidence_ideal }
        ];

        const results = [];
        let sumNow = 0, sumIdeal = 0, sumDiff = 0;
        let count = 0;

        scales.forEach((scale, index) => {
            const diff = scale.ideal - scale.now;
            
            // Для первой шкалы (здоровье) не считаем средние
            if (index > 0) {
                sumNow += scale.now;
                sumIdeal += scale.ideal;
                sumDiff += diff;
                count++;
            }

            results.push({
                name: scale.name,
                now: scale.now,
                ideal: scale.ideal,
                diff: diff,
                nowLevel: this.getSelfEsteemLevel(scale.now),
                idealLevel: this.getAspirationLevel(scale.ideal)
            });
        });

        return {
            scales: results,
            averages: {
                now: count > 0 ? (sumNow / count).toFixed(1) : 0,
                ideal: count > 0 ? (sumIdeal / count).toFixed(1) : 0,
                diff: count > 0 ? (sumDiff / count).toFixed(1) : 0
            }
        };
    }

    getSelfEsteemLevel(value) {
        if (value < 45) return 'Заниженная';
        if (value <= 74) return 'Адекватная';
        return 'Завышенная';
    }

    getAspirationLevel(value) {
        if (value < 60) return 'Заниженный';
        if (value <= 89) return 'Оптимальный';
        return 'Нереалистичный';
    }

    renderResults(results) {
        const container = document.getElementById('results-container');
        
        let html = `
            <table class="results-table">
                <thead>
                    <tr>
                        <th>Шкала</th>
                        <th>Сейчас</th>
                        <th>Уровень самооценки</th>
                        <th>Буду доволен</th>
                        <th>Уровень притязаний</th>
                        <th>Разница</th>
                    </tr>
                </thead>
                <tbody>
        `;

        results.scales.forEach(scale => {
            const diffClass = scale.diff >= 0 ? 'diff-positive' : 'diff-negative';
            html += `
                <tr>
                    <td><strong>${scale.name}</strong></td>
                    <td>${scale.now}</td>
                    <td>${scale.nowLevel}</td>
                    <td>${scale.ideal}</td>
                    <td>${scale.idealLevel}</td>
                    <td class="${diffClass}">${scale.diff >= 0 ? '+' : ''}${scale.diff}</td>
                </tr>
            `;
        });

        html += `
                </tbody>
                <tfoot class="averages">
                    <tr>
                        <td><strong>Средние значения (без шкалы "Здоровье")</strong></td>
                        <td>${results.averages.now}</td>
                        <td colspan="2">${results.averages.ideal}</td>
                        <td></td>
                        <td>${results.averages.diff}</td>
                    </tr>
                </tfoot>
            </table>
            
            <div class="interpretation">
                <h3>Интерпретация результатов:</h3>
                <p><strong>Самооценка:</strong> Средний балл ${results.averages.now} — 
                ${this.getSelfEsteemLevel(parseFloat(results.averages.now))}</p>
                <p><strong>Уровень притязаний:</strong> Средний балл ${results.averages.ideal} — 
                ${this.getAspirationLevel(parseFloat(results.averages.ideal))}</p>
                <p><strong>Средняя разница:</strong> ${results.averages.diff}</p>
            </div>
        `;

        container.innerHTML = html;
    }

    showError(message) {
        document.getElementById('results-container').innerHTML = `
            <div class="error-message" style="text-align: center; padding: 50px;">
                <h3>${message}</h3>
                <p>Пройдите тест заново, чтобы получить результаты.</p>
            </div>
        `;
    }
}

// Инициализация
document.addEventListener('DOMContentLoaded', () => {
    new ResultsViewer();
});