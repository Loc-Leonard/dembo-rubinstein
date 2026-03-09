class DemboRubinsteinSurvey {
    constructor() {
        this.scales = [
            { id: 'health', title: 'Здоровье', description: 'тренировочная шкала' },
            { id: 'mind', title: 'Ум, способности', description: 'интеллектуальные возможности' },
            { id: 'character', title: 'Характер', description: 'особенности личности' },
            { id: 'authority', title: 'Авторитет', description: 'признание в коллективе' },
            { id: 'hands', title: 'Умелые руки', description: 'практические навыки' },
            { id: 'appearance', title: 'Внешность', description: 'оценка привлекательности' },
            { id: 'confidence', title: 'Уверенность в себе', description: 'вера в свои силы' }
        ];

        this.responses = {};
        this.activeSlider = null;
        this.init();
    }

    init() {
        this.renderScales();
        this.attachEventListeners();
        this.resetResponses();
        /*this.enhanceMobileExperience();*/
    }

    resetResponses() {
        this.scales.forEach(scale => {
            this.responses[`${scale.id}_now`] = 0;
            this.responses[`${scale.id}_ideal`] = 0;
        });
    }

    renderScales() {
        const container = document.querySelector('.scales-container');
        if (!container) return;
        container.innerHTML = '';

        this.scales.forEach(scale => {
            const card = document.createElement('div');
            card.className = 'scale-card';
            card.dataset.scaleId = scale.id;

            card.innerHTML = `
                <div class="scale-title">
                    ${scale.title}
                    <small style="display: block; font-size: 10px; color: #999; margin-top: 5px;">
                        ${scale.description}
                    </small>
                </div>
                <div class="vertical-scale">
                    <div class="slider-wrapper" data-label="Сейчас">
                        <input type="range" min="0" max="100" value="0" class="slider now-slider"
                               data-scale="${scale.id}" data-type="now" id="${scale.id}-now">
                    </div>
                    <div class="slider-wrapper" data-label="Идеал">
                        <input type="range" min="0" max="100" value="0" class="slider ideal-slider"
                               data-scale="${scale.id}" data-type="ideal" id="${scale.id}-ideal">
                    </div>
                </div>
            `;
            container.appendChild(card);
        });

        const sliders = document.querySelectorAll('.slider');
        sliders.forEach(slider => {
            this.attachGradient(slider);
        });
    }

    attachGradient(slider) {
        const color = slider.classList.contains('now-slider') ? '#2196F3' : '#FF1493';
        const updateGradient = () => {
            slider.style.background = `linear-gradient(to right, ${color} ${slider.value}%, #ccc ${slider.value}%)`;
        };
        slider.addEventListener('input', updateGradient);
        updateGradient();
    }

    attachEventListeners() {
        const sliders = document.querySelectorAll('.slider');
        sliders.forEach(slider => {
            slider.addEventListener('input', e => this.handleSliderChange(e));
        });

        const submitBtn = document.getElementById('submit-btn');
        if (submitBtn) submitBtn.addEventListener('click', () => this.showResults());
    }

// enhanceMobileExperience() {
//     if (!('ontouchstart' in window)) return;

//     // Ждём, пока DOM точно отрисует все слайдеры
//     setTimeout(() => {
//         const wrappers = document.querySelectorAll('.slider-wrapper');
//         if (!wrappers.length) return;

//         let active = false;

//         const disableScroll = () => {
//             document.body.dataset.prevOverflow = document.body.style.overflow || '';
//             document.body.style.overflow = 'hidden';
//         };

//         const enableScroll = () => {
//             document.body.style.overflow = document.body.dataset.prevOverflow || '';
//         };

//         wrappers.forEach(wrapper => {
//             wrapper.addEventListener('touchstart', (e) => {
//                 active = true;
//                 disableScroll();
//                 // не даём событию уйти в скролл, но внутри wrapper браузер сам обработает drag по input
//                 e.preventDefault();
//             }, { passive: false });

//             wrapper.addEventListener('touchmove', (e) => {
//                 if (!active) return;
//                 e.preventDefault(); // блокируем прокрутку страницы
//             }, { passive: false });

//             const endTouch = (e) => {
//                 if (!active) return;
//                 active = false;
//                 enableScroll();
//                 e.preventDefault();
//             };

//             wrapper.addEventListener('touchend', endTouch, { passive: false });
//             wrapper.addEventListener('touchcancel', endTouch, { passive: false });
//         });

//         const buttons = document.querySelectorAll('.btn');
//         buttons.forEach(btn => {
//             btn.style.padding = '15px 30px';
//             btn.style.minHeight = '50px';
//         });
//     }, 0);
// }


    handleSliderChange(event) {
        const slider = event.target;
        const scaleId = slider.dataset.scale;
        const type = slider.dataset.type;
        this.responses[`${scaleId}_${type}`] = parseInt(slider.value);
        this.checkCompletion();
    }

    checkCompletion() {
        const expectedCount = this.scales.length * 2;
        const completedCount = Object.values(this.responses).filter(v => v > 0).length;

        const submitBtn = document.getElementById('submit-btn');
        const validationMsg = document.getElementById('validation-message');

        if (submitBtn) {
            if (completedCount === expectedCount) {
                submitBtn.disabled = false;
                if (validationMsg) validationMsg.textContent = '';
            } else {
                submitBtn.disabled = true;
                if (validationMsg) {
                    validationMsg.textContent =
                        `Передвиньте все ползунки (осталось ${expectedCount - completedCount})`;
                }
            }
        }
    }

    showResults() {
        const data = {};
        this.scales.forEach(scale => {
            const nowSlider = document.getElementById(`${scale.id}-now`);
            const idealSlider = document.getElementById(`${scale.id}-ideal`);

            data[`${scale.id}_now`] = nowSlider ? parseInt(nowSlider.value) : 0;
            data[`${scale.id}_ideal`] = idealSlider ? parseInt(idealSlider.value) : 0;
        });

        fetch('/api/save', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        })
        .then(response => {
            if (!response.ok) {
                throw new Error('Ошибка сервера');
            }
            return response.json();
        })
        .then(data => {
            const fullUrl = `${window.location.origin}${data.share_url}`;
            window.location.href = fullUrl;
        })
        .catch(error => {
            alert('❌ Ошибка сохранения: ' + error.message);
            console.error('Save error:', error);
        });
    }

    calculateResults(data) {
        const scales = [
            { name: 'Здоровье', now: data.health_now, ideal: data.health_ideal },
            { name: 'Ум, способности', now: data.mind_now, ideal: data.mind_ideal },
            { name: 'Характер', now: data.character_now, ideal: data.character_ideal },
            { name: 'Авторитет у сверстников', now: data.authority_now, ideal: data.authority_ideal },
            { name: 'Умелые руки', now: data.hands_now, ideal: data.hands_ideal },
            { name: 'Внешность', now: data.appearance_now, ideal: data.appearance_ideal },
            { name: 'Уверенность в себе', now: data.confidence_now, ideal: data.confidence_ideal }
        ];

        const results = [];
        let sumNow = 0, sumIdeal = 0, sumDiff = 0;
        let validScalesCount = 0;

        scales.forEach((scale, index) => {
            const diff = scale.ideal - scale.now;

            if (index > 0) {
                sumNow += scale.now;
                sumIdeal += scale.ideal;
                sumDiff += diff;
                validScalesCount++;
            }

            results.push({
                name: scale.name,
                now: scale.now,
                ideal: scale.ideal,
                diff: diff,
                nowLevel: this.getSelfEsteemLevel(scale.now),
                idealLevel: this.getAspirationLevel(scale.ideal),
                isTraining: index === 0
            });
        });

        const averages = {
            now: validScalesCount > 0 ? Math.round(sumNow / validScalesCount) : 0,
            ideal: validScalesCount > 0 ? Math.round(sumIdeal / validScalesCount) : 0,
            diff: validScalesCount > 0 ? Math.round(sumDiff / validScalesCount) : 0
        };

        return {
            scales: results,
            averages: averages,
            interpretation: this.getFullInterpretation(averages)
        };
    }

    getSelfEsteemLevel(value) {
        if (value < 45) return 'Заниженная (группа риска)';
        if (value <= 74) return 'Адекватная (средняя и высокая)';
        return 'Завышенная';
    }

    getAspirationLevel(value) {
        if (value < 60) return 'Заниженный';
        if (value <= 89) return 'Оптимальный (60-89)';
        return 'Нереалистичный (90-100)';
    }

    getFullInterpretation(averages) {
        const lines = [];

        lines.push('📊 УРОВЕНЬ ПРИТЯЗАНИЙ');
        if (averages.ideal >= 90) {
            lines.push('• 75-100 баллов: нереалистический, некритический уровень притязаний');
        } else if (averages.ideal >= 75) {
            lines.push('• 75-89 баллов: оптимальный уровень, подтверждающий оптимальное представление о своих возможностях');
        } else if (averages.ideal >= 60) {
            lines.push('• 60-74 балла: реалистический уровень притязаний');
        } else {
            lines.push('• менее 60 баллов: заниженный уровень притязаний, индикатор неблагоприятного развития личности');
        }

        lines.push('');

        lines.push('📈 ВЫСОТА САМООЦЕНКИ');
        if (averages.now >= 75) {
            lines.push('• 75-100 баллов: завышенная самооценка');
            lines.push('  Указывает на определенные отклонения в формировании личности:');
            lines.push('  - личностная незрелость');
            lines.push('  - неумение правильно оценить результаты своей деятельности');
            lines.push('  - закрытость для опыта, нечувствительность к своим ошибкам');
        } else if (averages.now >= 45) {
            lines.push('• 45-74 балла: адекватная самооценка');
            lines.push('  Реалистическая оценка своих возможностей, важный фактор личностного развития');
        } else {
            lines.push('• менее 45 баллов: заниженная самооценка');
            lines.push('  Свидетельствует о крайнем неблагополучии в развитии личности (группа риска)');
            lines.push('  Может скрывать:');
            lines.push('  - подлинную неуверенность в себе');
            lines.push('  - защитную позицию (декларирование отсутствия способностей, чтобы не прилагать усилий)');
        }

        lines.push('');

        lines.push('📉 РАСХОЖДЕНИЕ МЕЖДУ УРОВНЕМ ПРИТЯЗАНИЙ И САМООЦЕНКОЙ');
        if (averages.diff > 15) {
            lines.push('• Большое расхождение: уровень притязаний значительно выше самооценки');
            lines.push('  Может указывать на нереалистичные цели или низкую самооценку');
        } else if (averages.diff > 5) {
            lines.push('• Умеренное расхождение: здоровое стремление к развитию');
        } else if (averages.diff >= 0) {
            lines.push('• Гармоничное соотношение: уровень притязаний немного выше самооценки');
        } else {
            lines.push('• Отрицательное расхождение: уровень притязаний ниже самооценки');
            lines.push('  Может указывать на защитное поведение или низкую мотивацию');
        }

        return lines;
    }

    renderResults(results) {
        const container = document.querySelector('.container');

        document.getElementById('survey-form').style.display = 'none';

        const resultsDiv = document.createElement('div');
        resultsDiv.id = 'results-view';
        resultsDiv.innerHTML = `
            <h2>Результаты диагностики</h2>
            <table class="results-table">
                <thead>
                    <tr>
                        <th>Шкала</th>
                        <th>Самооценка (сейчас)</th>
                        <th>Уровень притязаний (идеал)</th>
                        <th>Разница</th>
                        <th>Интерпретация</th>
                    </tr>
                </thead>
                <tbody>
                    ${results.scales.map(scale => `
                        <tr ${scale.isTraining ? 'style="opacity: 0.7;"' : ''}>
                            <td><strong>${scale.name}</strong></td>
                            <td>${scale.now}</td>
                            <td>${scale.ideal}</td>
                            <td class="${scale.diff >= 0 ? 'diff-positive' : 'diff-negative'}">
                                ${scale.diff >= 0 ? '+' : ''}${scale.diff}
                            </td>
                            <td>
                                <small>
                                    Самооценка: ${scale.nowLevel}<br>
                                    Притязания: ${scale.idealLevel}
                                </small>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
                <tfoot class="averages">
                    <tr>
                        <td><strong>СРЕДНИЕ ЗНАЧЕНИЯ*</strong></td>
                        <td><strong>${results.averages.now}</strong></td>
                        <td><strong>${results.averages.ideal}</strong></td>
                        <td><strong>${results.averages.diff >= 0 ? '+' : ''}${results.averages.diff}</strong></td>
                        <td><em>по 6 шкалам (без "Здоровье")</em></td>
                    </tr>
                </tfoot>
            </table>

            <div class="interpretation-box">
                <h3>🔍 Интерпретация по методике Дембо-Рубинштейн</h3>
                ${results.interpretation.map(line => {
                    if (line.startsWith('•')) {
                        return `<p style="margin-left: 20px;">${line}</p>`;
                    } else if (line.startsWith('  ')) {
                        return `<p style="margin-left: 40px; color: #666;">${line.trim()}</p>`;
                    } else {
                        return `<p><strong>${line}</strong></p>`;
                    }
                }).join('')}
            </div>

            <div class="interpretation-box" style="background: #e8f5e8;">
                <h3>📌 Важно</h3>
                <p>• Первая шкала ("Здоровье") является тренировочной и не учитывается в средних показателях</p>
                <p>• Для получения более точных результатов рекомендуется проходить методику в спокойной обстановке</p>
                <p>• Результаты носят диагностический характер и требуют обсуждения со специалистом</p>
            </div>

            <div class="actions">
                <button class="btn" onclick="location.reload()">Пройти заново</button>
                <button class="btn" style="background: #2196F3;" onclick="copyResultsToClipboard()">
                    📋 Копировать результаты
                </button>
            </div>
        `;

        container.appendChild(resultsDiv);

        window.copyResultsToClipboard = () => {
            const resultsText = resultsDiv.innerText;
            navigator.clipboard.writeText(resultsText).then(() => {
                alert('Результаты скопированы в буфер обмена!');
            });
        };
    }

    renderShareLink(shareUrl) {
        const form = document.getElementById('survey-form');
        if (form) form.style.display = 'none';

        const container = document.querySelector('.container');
        const shareDiv = document.createElement('div');
        shareDiv.className = 'share-results';
        shareDiv.innerHTML = `
            <div style="background: #e8f5e8; padding: 20px; border-radius: 8px; text-align: center;">
                <h2>✅ Результаты сохранены!</h2>
                <p>Скопируйте ссылку и отправьте проверяющему:</p>
                <div style="margin: 20px 0;">
                    <input id="share-link" value="${window.location.origin}${shareUrl}"
                           readonly style="width: 70%; padding: 10px; font-size: 14px;">
                    <button class="btn" onclick="navigator.clipboard.writeText(document.getElementById('share-link').value)"
                            style="width: 25%; padding: 10px; margin-left: 5px;">
                        📋 Копировать
                    </button>
                </div>
                <p style="font-size: 12px; color: #666;">
                    Проверяющий увидит те же результаты, что и вы
                </p>
                <button class="btn" onclick="location.reload()" style="background: #6c757d;">
                    Пройти заново
                </button>
            </div>
        `;
        container.appendChild(shareDiv);
    }
}

document.addEventListener('DOMContentLoaded', () => {
    new DemboRubinsteinSurvey();
});
