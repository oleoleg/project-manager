// web/static/budget_plan.js

// ============ Вспомогательные функции ============
window.switchTab = function(tabId, btn) {
    // 1. Скрываем все вкладки, убираем активный класс с кнопок
    document.querySelectorAll('.tab-content').forEach(el => el.style.display = 'none');
    document.querySelectorAll('.tab-btn').forEach(el => el.classList.remove('active'));
    
    // 2. Показываем нужную
    const tab = document.getElementById('tab-' + tabId);
    if (tab) tab.style.display = 'block';
    if (btn) btn.classList.add('active');

    // 3. Обновляем URL без перезагрузки страницы (сохраняем текущий снимок)
    const url = new URL(window.location.href);
    url.searchParams.set('tab', tabId);
    window.history.replaceState({}, '', url);

    // 4. Обновляем скрытое поле в форме, чтобы при смене версии вкладка сохранилась
    const hiddenInput = document.getElementById('current-tab-input');
    if (hiddenInput) hiddenInput.value = tabId;
};

window.toggleSettings = function() {
    const body = document.getElementById('settings-body');
    const btn = document.querySelector('#section-settings .toggle-btn');
    if (!body) return;
    if (body.style.display === 'none') {
        body.style.display = 'block';
        if (btn) btn.textContent = 'Свернуть';
    } else {
        body.style.display = 'none';
        if (btn) btn.textContent = 'Развернуть';
    }
};

window.askNote = function(btn) {
    let note = prompt("Укажите причину генерации нового плана (например: 'Доп. соглашение №1'):");
    if (note === null) return false;
    const input = btn.closest('form').querySelector('.generate-note-input');
    if (input) input.value = note;
    return true;
};

// ============ Запуск при загрузке ============
document.addEventListener('DOMContentLoaded', function() {
    // 1. Восстанавливаем активную вкладку из URL (если есть)
    const urlParams = new URLSearchParams(window.location.search);
    const activeTab = urlParams.get('tab');
    
    if (activeTab) {
        // Ищем кнопку, которая вызывает switchTab с нужным ID
        const btn = document.querySelector(`.tab-btn[onclick*="'${activeTab}'"]`);
        if (btn) {
            switchTab(activeTab, btn);
        }
        
        // Заполняем скрытое поле формы, чтобы при смене версии вкладка сохранилась
        const hiddenInput = document.getElementById('current-tab-input');
        if (hiddenInput) hiddenInput.value = activeTab;
    }

    // 1. Сортировка таблицы плана
    const typeBtn = Array.from(document.querySelectorAll('.filter-bar button')).find(b => b.textContent.includes('По типу операции'));
    if (typeBtn && typeof sortTable === 'function') {
        sortTable('plan-table', 'type', typeBtn, {
            labelColspan: 3, incIdx: 6, expIdx: 7, balIdx: 8,
            hasIncome: true, hasExpense: true, hasBalance: true
        });
    }

    // 2. Отрисовка графиков Chart.js
    const contractID = window.location.pathname.split('/')[2]; // /contracts/{id}/budget
    const snapshotSelect = document.querySelector('select[name="snapshot"]');
    const snapshotID = snapshotSelect ? snapshotSelect.value : '';
    
    if (!snapshotID || typeof Chart === 'undefined') return;

    const ctxExp = document.getElementById('expensesChart');
    const ctxDept = document.getElementById('departmentsChart');
    
    if (!ctxExp || !ctxDept) return;

    fetch(`/contracts/${contractID}/budget/charts.json?snapshot=${snapshotID}`)
        .then(r => r.json())
        .then(data => {
            // 1. Фиксированные цвета
            const colorMap = {
                'Платежи по договору': '#36a2eb', 'Закупка по договору': '#ff6384',
                'Зарплата сотрудникам': '#ffce56', 'Накладные': '#4bc0c0',
                'Командировки': '#9966ff', 'Субподрядчики': '#ff9f40',
                'Комиссия': '#8e5ea2', 'Другое (расходы)': '#c9cbcf',
                'Прибыль': '#2ecc71',
                'ПИР': '#36a2eb', 'СМР': '#ff6384', 'ПНР': '#ffce56',
                'ТО': '#4bc0c0', 'Сборка': '#9966ff', 'Ремонт': '#ff9f40',
                'Продажа': '#8e5ea2', 'Комплексный': '#c9cbcf'
            };

            function getColors(labels) {
                const defaultColors = ['#36a2eb', '#ff6384', '#ffce56', '#4bc0c0', '#9966ff', '#ff9f40', '#c9cbcf'];
                return labels.map((label, i) => colorMap[label] || defaultColors[i % defaultColors.length]);
            }

            // 2. Плагин для отрисовки процентов (вынесены наружу, с контрастным фоном)
            const percentPlugin = {
                id: 'percentPlugin',
                afterDatasetsDraw(chart) {
                    const { ctx, data } = chart;
                    const total = data.datasets[0].data.reduce((a, b) => a + b, 0);
                    
                    chart.getDatasetMeta(0).data.forEach((datapoint, index) => {
                        const value = data.datasets[0].data[index];
                        if (value === 0 || (value / total) < 0.02) return; // Скрываем совсем мелкие
                        
                        const percent = ((value / total) * 100).toFixed(1) + '%';
                        
                        // Вычисляем координаты для выноски (чуть дальше от центра)
                        const angle = (datapoint.startAngle + datapoint.endAngle) / 2;
                        const radius = datapoint.outerRadius * 0.7; // Смещаем от центра к краю
                        const x = datapoint.x + Math.cos(angle) * (datapoint.outerRadius * 0.6);
                        const y = datapoint.y + Math.sin(angle) * (datapoint.outerRadius * 0.6);

                        ctx.save();
                        // Тень для контраста на светлом/темном фоне
                        ctx.shadowColor = 'rgba(0,0,0,0.5)';
                        ctx.shadowBlur = 4;
                        
                        ctx.fillStyle = '#ffffff'; // Белый текст всегда виден на цветных секторах
                        ctx.font = 'bold 14px sans-serif';
                        ctx.textAlign = 'center';
                        ctx.textBaseline = 'middle';
                        ctx.fillText(percent, x, y);
                        ctx.restore();
                    });
                }
            };

            // 3. Функция для генерации кастомной HTML-легенды (в виде списка)
            function renderHtmlLegend(chart, containerId, chartData) {
                const container = document.getElementById(containerId);
                if (!container) return;
                
                const total = chartData.data.reduce((a, b) => a + b, 0);
                let html = '<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 8px; margin-top: 15px; font-size: 13px;">';
                
                chartData.labels.forEach((label, i) => {
                    const value = chartData.data[i];
                    const color = chartData.backgroundColor[i];
                    const formatted = new Intl.NumberFormat('ru-RU', { style: 'currency', currency: 'RUB', maximumFractionDigits: 0 }).format(value);
                    const percent = total > 0 ? ((value / total) * 100).toFixed(1) : '0.0';
                    
                    html += `
                        <div style="display: flex; align-items: center; gap: 8px; padding: 4px; background: var(--surface-alt); border-radius: 4px;">
                            <span style="display: inline-block; width: 12px; height: 12px; background-color: ${color}; border-radius: 2px; flex-shrink: 0;"></span>
                            <span style="flex: 1; color: var(--text);">${label}</span>
                            <span style="font-weight: bold; color: var(--text-mid); white-space: nowrap;">${formatted} (${percent}%)</span>
                        </div>
                    `;
                });
                html += '</div>';
                container.innerHTML = html;
            }

            // Отрисовка первого графика
            const expData = {
                labels: data.expenses_chart.labels,
                data: data.expenses_chart.data,
                backgroundColor: getColors(data.expenses_chart.labels)
            };
            const ctxExp = document.getElementById('expensesChart').getContext('2d');
            const expChart = new Chart(ctxExp, {
                type: 'pie',
                data: {
                    labels: expData.labels,
                    datasets: [{ data: expData.data, backgroundColor: expData.backgroundColor }]
                },
                options: { 
                    responsive: true, maintainAspectRatio: false,
                    plugins: { legend: { display: false } } // Отключаем встроенную легенду
                },
                plugins: [percentPlugin]
            });
            renderHtmlLegend(expChart, 'expenses-legend', expData);

            // Отрисовка второго графика
            const deptData = {
                labels: data.departments_chart.labels,
                data: data.departments_chart.data,
                backgroundColor: getColors(data.departments_chart.labels)
            };
            const ctxDept = document.getElementById('departmentsChart').getContext('2d');
            const deptChart = new Chart(ctxDept, {
                type: 'pie',
                data: {
                    labels: deptData.labels,
                    datasets: [{ data: deptData.data, backgroundColor: deptData.backgroundColor }]
                },
                options: { 
                    responsive: true, maintainAspectRatio: false,
                    plugins: { legend: { display: false } } // Отключаем встроенную легенду
                },
                plugins: [percentPlugin]
            });
            renderHtmlLegend(deptChart, 'departments-legend', deptData);
        })
        .catch(err => console.error('Ошибка загрузки графиков:', err));



    // === Временной график ===
    let isSelecting = false;
    let selStartPixel = null;
    let selEndPixel = null;
    let selStartIndex = null;
    let selEndIndex = null;

    window.updateTimeline = function() {
        const groupEl = document.getElementById('timeline-group');
        const negativeEl = document.getElementById('timeline-negative-expense');
        if (!groupEl) return;
        
        const group = groupEl.value;
        const isNegative = negativeEl ? negativeEl.checked : false;
        const ctxTimeline = document.getElementById('timelineChart');
        if (!ctxTimeline) return;

        const isDark = document.documentElement.getAttribute('data-theme') === 'dark';
        const textColor = isDark ? '#e6e9ee' : '#2f3640';
        const gridColor = isDark ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.05)';
        
        const colorIncome = '#36a2eb';
        const colorExpense = '#ff6384';
        const colorBalance = '#2ecc71';
        const colorIncPeriod = isDark ? '#90caf9' : '#1976d2';
        const colorExpPeriod = isDark ? '#f48fb1' : '#c2185b';

        fetch(`/contracts/${contractID}/budget/timeline.json?snapshot=${snapshotID}&group=${group}`)
            .then(r => r.json())
            .then(data => {
                if (window.myTimelineChart) {
                    window.myTimelineChart.destroy();
                }
                
                // Сброс выделения при смене настроек
                isSelecting = false;
                selStartPixel = null;
                selEndPixel = null;
                selStartIndex = null;
                selEndIndex = null;

                const labels = data.map(d => d.period);
                const sign = isNegative ? -1 : 1;

                const incomeCum = data.map(d => d.income_cum);
                const expenseCum = data.map(d => d.expense_cum * sign);
                const balanceCum = data.map(d => d.income_cum - d.expense_cum);
                const incomeMonth = data.map(d => d.income_month);
                const expenseMonth = data.map(d => d.expense_month * sign);

                // Кастомный плагин для отрисовки выделения и курсора
                const selectionPlugin = {
                    id: 'selectionPlugin',
                    afterDatasetsDraw(chart) {
                        const { ctx, chartArea: { top, bottom } } = chart;
                        
                        // Если есть наведение мышью, рисуем вертикальную линию-курсор (если не выделяем)
                        if (!isSelecting && chart.tooltip._active && chart.tooltip._active.length > 0) {
                            const activePoint = chart.tooltip._active[0];
                            const x = activePoint.element.x;
                            ctx.save();
                            ctx.beginPath();
                            ctx.moveTo(x, top);
                            ctx.lineTo(x, bottom);
                            ctx.lineWidth = 1;
                            ctx.strokeStyle = isDark ? 'rgba(255,255,255,0.4)' : 'rgba(0,0,0,0.3)';
                            ctx.stroke();
                            ctx.restore();
                        }

                        // Если выделяем или уже выделили
                        if (selStartPixel !== null && selEndPixel !== null) {
                            const start = Math.min(selStartPixel, selEndPixel);
                            const end = Math.max(selStartPixel, selEndPixel);
                            
                            ctx.save();
                            // Заливка области
                            ctx.fillStyle = 'rgba(64, 115, 158, 0.2)';
                            ctx.fillRect(start, top, end - start, bottom - top);
                            
                            // Границы области
                            ctx.beginPath();
                            ctx.moveTo(start, top);
                            ctx.lineTo(start, bottom);
                            ctx.moveTo(end, top);
                            ctx.lineTo(end, bottom);
                            ctx.lineWidth = 2;
                            ctx.strokeStyle = '#40739e';
                            ctx.stroke();
                            ctx.restore();
                        }
                    }
                };

                window.myTimelineChart = new Chart(ctxTimeline.getContext('2d'), {
                    type: 'line',
                    data: {
                        labels: labels,
                        datasets: [
                            { label: 'Поступления (нараст.)', data: incomeCum, borderColor: colorIncome, backgroundColor: 'rgba(54, 162, 235, 0.1)', fill: false, tension: 0.2, pointRadius: 3 , hidden: true },
                            { label: 'Затраты (нараст.)', data: expenseCum, borderColor: colorExpense, backgroundColor: 'rgba(255, 99, 132, 0.1)', fill: false, tension: 0.2, pointRadius: 3 , hidden: true },
                            { label: 'Баланс (нараст.)', data: balanceCum, borderColor: colorBalance, backgroundColor: 'rgba(46, 204, 113, 0.1)', fill: true, tension: 0.2, pointRadius: 3 },
                            { label: 'Поступления (за период)', data: incomeMonth, borderColor: colorIncPeriod, borderDash: [5, 5], backgroundColor: 'transparent', fill: false },
                            { label: 'Затраты (за период)', data: expenseMonth, borderColor: colorExpPeriod, borderDash: [5, 5], backgroundColor: 'transparent', fill: false }
                        ]
                    },
                    options: {
                        responsive: true,
                        maintainAspectRatio: false,
                        plugins: {
                            legend: { display: false },
                            tooltip: { enabled: false } // Отключаем стандартный тултип, чтобы не мешал
                        },
                        scales: {
                            x: { ticks: { color: textColor }, grid: { color: gridColor } },
                            y: { ticks: { color: textColor }, grid: { color: gridColor } }
                        },
                        onHover: (event, elements) => {
                            if (!isSelecting && elements.length > 0) {
                                renderTimelineLegend(window.myTimelineChart, elements[0].index);
                            } else if (!isSelecting) {
                                renderTimelineLegend(window.myTimelineChart, null);
                            }
                        }
                    },
                    plugins: [selectionPlugin]
                });

                const chart = window.myTimelineChart;

                // Вспомогательная функция для получения пикселя и индекса
                const getPixelAndIndex = (e) => {
                    const rect = ctxTimeline.getBoundingClientRect();
                    const x = e.clientX - rect.left;
                    // Находим ближайшую точку по оси X
                    const chartArea = chart.chartArea;
                    const boundedX = Math.max(chartArea.left, Math.min(x, chartArea.right));
                    
                    // Переводим пиксель в значение по оси X
                    const xValue = chart.scales.x.getValueForPixel(boundedX);
                    if (xValue === undefined) return { x: boundedX, index: null };

                    // Находим ближайший индекс
                    let minDist = Infinity;
                    let closestIndex = 0;
                    
                    // Если метки строковые (формат 2026-10), Chart.js использует их как категории.
                    // Мы можем найти индекс по расстоянию на экране.
                    chart.data.labels.forEach((_, i) => {
                        const px = chart.scales.x.getPixelForValue(i);
                        const dist = Math.abs(px - boundedX);
                        if (dist < minDist) {
                            minDist = dist;
                            closestIndex = i;
                        }
                    });
                    
                    return { x: boundedX, index: closestIndex };
                };

                // Обработка мыши
                const startSelection = (e) => {
                    e.preventDefault();
                    const { x, index } = getPixelAndIndex(e);
                    if (index !== null) {
                        isSelecting = true;
                        selStartPixel = x;
                        selEndPixel = x;
                        selStartIndex = index;
                        selEndIndex = index;
                        chart.update();
                        renderTimelineLegend(chart, null);
                        
                        // Навешиваем глобальные события
                        document.addEventListener('mousemove', onMouseMove);
                        document.addEventListener('mouseup', onMouseUp);
                    }
                };

                const onMouseMove = (e) => {
                    if (!isSelecting) return;
                    const { x, index } = getPixelAndIndex(e);
                    if (index !== null) {
                        // Ограничиваем, чтобы не выйти за пределы
                        selEndPixel = x;
                        selEndIndex = index;
                        chart.update();
                        renderTimelineLegend(chart, null);
                    }
                };

                const onMouseUp = (e) => {
                    if (isSelecting) {
                        isSelecting = false;
                        document.removeEventListener('mousemove', onMouseMove);
                        document.removeEventListener('mouseup', onMouseUp);
                        renderTimelineLegend(chart, null);
                    }
                };

                // Привязываем событие mousedown к холсту
                ctxTimeline.onmousedown = startSelection;
                // Для тач-устройств
                ctxTimeline.ontouchstart = startSelection;


            })
            .catch(err => console.error('Ошибка загрузки графика динамики:', err));
    };

    // Функция отрисовки HTML-легенды
    function renderTimelineLegend(chart, hoverIndex = null) {
        const container = document.getElementById('timeline-legend');
        if (!container) return;

        const labels = chart.data.labels;
        const datasets = chart.data.datasets;
        
        let html = '';

        // 1. БЛОК ВЫБОРА ПЕРЬЕВ (КЛИКАБЕЛЬНЫЕ ЛИНИИ)
        html += '<div style="display: flex; flex-wrap: wrap; gap: 15px; justify-content: center; font-size: 14px; margin-bottom: 15px; padding-bottom: 10px; border-bottom: 1px solid var(--border-soft);">';
        datasets.forEach((ds, i) => {
            const isVisible = chart.isDatasetVisible(i);
            const opacity = isVisible ? '1' : '0.4';
            const textDecoration = isVisible ? 'none' : 'line-through';
            html += `
                <div style="display: flex; align-items: center; gap: 8px; cursor: pointer; opacity: ${opacity}; text-decoration: ${textDecoration};" 
                     onclick="toggleTimelineDataset(${i})">
                    <span style="display: inline-block; width: 20px; height: 3px; background-color: ${ds.borderColor}; border-radius: 2px;"></span>
                    <span>${ds.label}</span>
                </div>
            `;
        });
        html += '</div>';

        // 2. БЛОК ДАННЫХ
        if (selStartIndex !== null && selEndIndex !== null) {
            // ВЫДЕЛЕННЫЙ ДИАПАЗОН (от руки)
            const startIdx = Math.min(selStartIndex, selEndIndex);
            const endIdx = Math.max(selStartIndex, selEndIndex);
            const periodLabel = `${labels[startIdx]} — ${labels[endIdx]}`;
            
            html += `<div style="background: var(--accent-soft); padding: 12px; border-radius: 6px; border-left: 4px solid var(--accent);">
                <strong style="display: block; margin-bottom: 10px; color: var(--text);">Выделенный период: ${periodLabel}</strong>
                <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 10px;">`;
            
            datasets.forEach((ds, i) => {
                if (chart.isDatasetVisible(i)) {
                    let displayValue = 0;
                    
                    if (ds.label.includes('нараст')) {
                        // ДЛЯ НАРАСТАЮЩИХ: всегда берем последнее значение в диапазоне
                        displayValue = ds.data[endIdx];
                    } else {
                        // ДЛЯ ПЕРИОДИЧЕСКИХ: суммируем значения
                        for (let j = startIdx; j <= endIdx; j++) {
                            displayValue += ds.data[j] || 0;
                        }
                    }

                    if (displayValue !== 0) {
                        const formatted = new Intl.NumberFormat('ru-RU', { style: 'currency', currency: 'RUB', maximumFractionDigits: 0 }).format(Math.abs(displayValue));
                        const sign = displayValue < 0 ? '-' : '';
                        html += `<div style="display: flex; align-items: center; gap: 8px; color: var(--text-mid);">
                            <span style="display: inline-block; width: 10px; height: 10px; background-color: ${ds.borderColor}; border-radius: 50%;"></span>
                            <span>${ds.label}: <strong style="color: var(--text);">${sign}${formatted}</strong></span>
                        </div>`;
                    }
                }
            });
            html += `</div></div>`;
            
        } else if (hoverIndex !== null && labels[hoverIndex]) {
            // НАВЕДЕНИЕ НА ТОЧКУ
            html += `<div style="background: var(--surface-alt); padding: 12px; border-radius: 6px; border-left: 4px solid var(--border-soft);">
                <strong style="display: block; margin-bottom: 10px; color: var(--text);">Данные за период: ${labels[hoverIndex]}</strong>
                <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 10px;">`;
            
            datasets.forEach((ds, i) => {
                if (chart.isDatasetVisible(i)) {
                    const val = ds.data[hoverIndex];
                    if (val !== null && val !== undefined) {
                        const formatted = new Intl.NumberFormat('ru-RU', { style: 'currency', currency: 'RUB', maximumFractionDigits: 0 }).format(Math.abs(val));
                        html += `<div style="display: flex; align-items: center; gap: 8px; color: var(--text-mid);">
                            <span style="display: inline-block; width: 10px; height: 10px; background-color: ${ds.borderColor}; border-radius: 50%;"></span>
                            <span>${ds.label}: <strong style="color: var(--text);">${formatted}</strong></span>
                        </div>`;
                    }
                }
            });
            html += `</div></div>`;
        } else {
             html += `<div style="text-align: center; font-size: 13px; color: var(--text-muted); padding: 10px;">
                Наведите курсор на график или зажмите мышь, чтобы выделить период.
             </div>`;
        }

        container.innerHTML = html;
    }
    
    window.toggleTimelineDataset = function(index) {
        if (!window.myTimelineChart) return;
        const meta = window.myTimelineChart.getDatasetMeta(index);
        meta.hidden = meta.hidden === null ? !window.myTimelineChart.data.datasets[index].hidden : null;
        window.myTimelineChart.update();
        renderTimelineLegend(window.myTimelineChart, null);
    };

    if (document.getElementById('timelineChart')) {
        updateTimeline();
    }
 


});