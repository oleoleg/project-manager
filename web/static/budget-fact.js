
// Веса для сортировки Типа операции и Вида работы
const TYPE_ORDER = {
    "Прибыль": 1, "Платежи по договору": 2, "Закупка по договору": 3, "Зарплата сотрудникам": 4,
    "Накладные": 5, "Командировки": 6, "Субподрядчики": 7, "Комиссия": 8, "Другое (расходы)": 9
};
const WORK_ORDER = { "ПИР": 1, "ППО": 2, "СМР": 3, "ПНР": 4, "Продажа": 5, "ТО": 6, "Ремонт": 7, "Сборка": 8, "Комплексный": 9 };


function recalculateTotals() {
    const table = document.getElementById('fact-table');
    if (!table) return;
    
    let totalIncome = 0;
    let totalExpense = 0;
    let confirmedIncome = 0;
    let confirmedExpense = 0;

    // Считаем суммы по всем строкам
    const rows = table.querySelectorAll('tr.fact-row');
    rows.forEach(row => {
        const incInput = row.querySelector('input[name="fact_income[]"]');
        const expInput = row.querySelector('input[name="fact_expense[]"]');
        
        const incVal = incInput ? parseMoneyText(incInput.value) : 0;
        const expVal = expInput ? parseMoneyText(expInput.value) : 0;
        
        totalIncome += incVal;
        totalExpense += expVal;

        const isConfirmed = row.classList.contains('confirmed');
        if (isConfirmed) {
            confirmedIncome += incVal;
            confirmedExpense += expVal;
        }
    });

    // Записываем в общие карточки
    const totalIncEl = document.getElementById('total-fact-income');
    const totalExpEl = document.getElementById('total-fact-expense');
    const totalBalEl = document.getElementById('total-fact-balance');

    if (totalIncEl) totalIncEl.textContent = formatMoney(totalIncome);
    if (totalExpEl) totalExpEl.textContent = formatMoney(totalExpense);
    if (totalBalEl) totalBalEl.textContent = formatMoney(totalIncome - totalExpense);

    const confIncEl = document.getElementById('confirmed-fact-income');
    const confExpEl = document.getElementById('confirmed-fact-expense');
    const confBalEl = document.getElementById('confirmed-fact-balance');

    if (confIncEl) confIncEl.textContent = 'Подтв.: ' + formatMoney(confirmedIncome);
    if (confExpEl) confExpEl.textContent = 'Подтв.: ' + formatMoney(confirmedExpense);
    if (confBalEl) confBalEl.textContent = 'Подтв.: ' + formatMoney(confirmedIncome - confirmedExpense);
}
 

window.switchTab = function(tabId, btn) {
    console.log("Switching to tab:", tabId);
    document.querySelectorAll('.tab-content').forEach(el => el.style.display = 'none');
    document.querySelectorAll('.tab-btn').forEach(el => el.classList.remove('active'));
    
    const tab = document.getElementById('tab-' + tabId);
    if (tab) tab.style.display = 'block';
    if (btn) btn.classList.add('active');
    
    // Сохраняем в URL
    const url = new URL(window.location.href);
    url.searchParams.set('tab', tabId);
    window.history.replaceState({}, '', url);

    // Если мы переключаемся на "Сводку", инициализируем временной график
    if (tabId === 'summary') {
        if (document.getElementById('factTimelineChart')) {
            // Небольшая задержка, чтобы div успел получить display: block
            setTimeout(() => {
                // Проверяем, не нарисован ли уже график, чтобы не перерисовывать зря
                if (!window.myFactTimelineChart) {
                    updateFactTimeline();
                }
            }, 100);
        }
    }
};



document.addEventListener('DOMContentLoaded', function() {

    // Читаем JSON из скрытых полей
    const wkInput = document.getElementById('json-work-kinds');
    const swkInput = document.getElementById('json-stage-work-kinds');
    
    try {
        window.workKinds = wkInput && wkInput.value ? JSON.parse(wkInput.value) : [];
    } catch (e) {
        window.workKinds = [];
    }

    try {
        window.stageWorkKinds = swkInput && swkInput.value ? JSON.parse(swkInput.value) : {};
    } catch (e) {
        window.stageWorkKinds = {};
    }

    // 2. Формируем HTML для выбора этапа (используем номер как value)
    window.stageOptionsHTML = ``;
    const stageNodes = document.querySelectorAll('#stage-data div');
    stageNodes.forEach(node => {
        const name = node.getAttribute('data-name');
        const stageNum = name.split('—')[0].trim(); 
        const isSelected = node.hasAttribute('data-selected') ? 'selected' : '';
        window.stageOptionsHTML += `<option value="${stageNum}" data-stage-num="${stageNum}" ${isSelected}>${name}</option>`;
    });

    // 3. Включаем блокировку полей для уже подтвержденных строк и вешаем обработчики
    document.querySelectorAll('input[name="is_confirmed[]"]').forEach(cb => {
        if (cb.checked) {
            toggleRowInputs(cb);
        }
    });
    
    if (window.attachMoneyInput) {
        document.querySelectorAll('input.money-input').forEach(window.attachMoneyInput);
    }

    document.querySelectorAll('input[name="fact_income[]"], input[name="fact_expense[]"]').forEach(inp => {
        inp.addEventListener('input', recalculateTotals);
    });

    // 4. Считаем итоги
    recalculateTotals();

    // 5. И только теперь сортируем 
    const typeBtn = Array.from(document.querySelectorAll('.filter-bar button')).find(b => b.textContent.includes('По типу операции'));
    if (typeBtn) {
        sortFactTable('type', typeBtn);
    }


    // Восстанавливаем вкладку при загрузке
    const urlParams = new URLSearchParams(window.location.search);
    const activeTab = urlParams.get('tab') || 'details';
    const btn = document.querySelector(`.tab-btn[onclick*="'${activeTab}'"]`);
    if (btn) {
        switchTab(activeTab, btn);
    }

    // === Графики для ФинФакта ===
        // Получаем данные для графиков при загрузке страницы (если мы на вкладке Сводка)
    // Загружаем всегда, чтобы не тормозить при переключении
    const factMode = urlParams.get('fact_mode') || 'confirmed';
    
    const contractID = window.location.pathname.split('/')[2]; // /contracts/{id}/budget/fact

    fetch(`/contracts/${contractID}/budget/fact/charts.json?fact_mode=${factMode}`)
        .then(r => r.json())
        .then(data => {
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
                    // Считаем сумму только не-null и положительных значений
                    const total = data.datasets[0].data.reduce((a, b) => (b !== null && b > 0) ? a + b : a, 0);
                    if (total === 0) return; // Защита от деления на ноль
                    
                    chart.getDatasetMeta(0).data.forEach((datapoint, index) => {
                        const value = data.datasets[0].data[index];
                        if (!value || value <= 0 || (value / total) < 0.02) return; 
                        
                        const percent = ((value / total) * 100).toFixed(1) + '%';

                        // Вычисляем координаты для выноски (чуть дальше от центра)
                        const angle = (datapoint.startAngle + datapoint.endAngle) / 2;
                        const x = datapoint.x + Math.cos(angle) * (datapoint.outerRadius * 0.6);
                        const y = datapoint.y + Math.sin(angle) * (datapoint.outerRadius * 0.6);

                        ctx.save();
                        ctx.shadowColor = 'rgba(0,0,0,0.5)';
                        ctx.shadowBlur = 4;
                        ctx.fillStyle = '#ffffff';
                        ctx.font = 'bold 14px sans-serif';
                        ctx.textAlign = 'center';
                        ctx.textBaseline = 'middle';
                        ctx.fillText(percent, x, y);
                        ctx.restore();
                    });
                }
            };

            // Функция сортировки данных для легенды перед отрисовкой
            function sortLegendData(chartData, sortType) {
                const indices = chartData.labels.map((_, i) => i);
                
                indices.sort((a, b) => {
                    const labelA = chartData.labels[a];
                    const labelB = chartData.labels[b];
                    let weightA = 99;
                    let weightB = 99;

                    if (sortType === 'type') {
                        weightA = TYPE_ORDER[labelA] || 99;
                        weightB = TYPE_ORDER[labelB] || 99;
                    } else if (sortType === 'work') {
                        weightA = WORK_ORDER[labelA] || 99;
                        weightB = WORK_ORDER[labelB] || 99;
                    }

                    // Если веса разные — сортируем по ним
                    if (weightA !== weightB) {
                        return weightA - weightB;
                    }
                    // Если веса одинаковые (например, неизвестный тип) — сортируем по алфавиту для стабильности
                    return labelA.localeCompare(labelB);
                });

                // Возвращаем новые отсортированные массивы
                return {
                    labels: indices.map(i => chartData.labels[i]),
                    data: indices.map(i => chartData.data[i]),
                    backgroundColor: indices.map(i => chartData.backgroundColor[i])
                };
            }
             // 3. Функция для генерации кастомной HTML-легенды (в виде списка)
            function renderHtmlLegend(containerId, chartData) {
                const container = document.getElementById(containerId);
                if (!container) return;
                
                // Считаем сумму ТОЛЬКО положительных значений для корректного расчета процентов
                const positiveTotal = chartData.data.reduce((sum, val) => val > 0 ? sum + val : sum, 0);
                
                let html = '<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 8px; margin-top: 15px; font-size: 13px;">';
                
                chartData.labels.forEach((label, i) => {
                    const value = chartData.data[i];
                    const color = chartData.backgroundColor[i];
                    const formatted = new Intl.NumberFormat('ru-RU', { style: 'currency', currency: 'RUB', maximumFractionDigits: 0 }).format(value);
                    
                    let percentText = '';
                    let textColor = 'var(--text)';
                    let valueColor = 'var(--text-mid)';
                    
                    if (value < 0) {
                        // Отрицательное значение
                        percentText = '';
                        textColor = 'var(--error-text)';
                        valueColor = 'var(--error-text)';
                    } else {
                        // Положительное значение
                        const percent = positiveTotal > 0 ? ((value / positiveTotal) * 100).toFixed(1) : '0.0';
                        percentText = ` (${percent}%)`;
                    }

                    html += `
                        <div style="display: flex; align-items: center; gap: 8px; padding: 4px; background: var(--surface-alt); border-radius: 4px;">
                            <span style="display: inline-block; width: 12px; height: 12px; background-color: ${color}; border-radius: 2px; flex-shrink: 0;"></span>
                            <span style="flex: 1; color: ${textColor}; ${value < 0 ? 'font-weight: bold;' : ''}">${label}</span>
                            <span style="font-weight: bold; color: ${valueColor}; white-space: nowrap;">${formatted}${percentText}</span>
                        </div>
                    `;
                });
                html += '</div>';
                container.innerHTML = html;
            }

            // Функция для подсчета и вывода суммы под заголовком
            function updateTotalSum(containerId, dataArray) {
                const total = dataArray.reduce((a, b) => a + b, 0);
                const formatted = new Intl.NumberFormat('ru-RU', { style: 'currency', currency: 'RUB', maximumFractionDigits: 0 }).format(total);
                const el = document.getElementById(containerId);
                if (el) el.textContent = 'Итого: ' + formatted;
            }

            // --- График 1: Разбивка суммы Договора --- 
            const expDataRaw = {
                labels: data.expenses_chart.labels,
                data: data.expenses_chart.data,
                backgroundColor: getColors(data.expenses_chart.labels)
            };
            const expData = sortLegendData(expDataRaw, 'type');
            
            const chartExpData = expData.data.map(v => v >= 0 ? v : 0);

            const ctxExp = document.getElementById('factExpensesChart');
            if (ctxExp) {
                new Chart(ctxExp.getContext('2d'), {
                    type: 'pie',
                    data: {
                        labels: expData.labels,
                        datasets: [{ data: chartExpData, backgroundColor: expData.backgroundColor }]
                    },
                    options: { 
                        responsive: true, maintainAspectRatio: false,
                        plugins: { legend: { display: false } }
                    },
                    plugins: [percentPlugin]
                });
                renderHtmlLegend('fact-expenses-legend', expData);
                updateTotalSum('fact-expenses-total-sum', expData.data);
            }

            // --- График 2: Разбивка прибыли по Отделам ---
            const profitDataRaw = {
                labels: data.profit_chart.labels,
                data: data.profit_chart.data,
                backgroundColor: getColors(data.profit_chart.labels)
            };
            const profitData = sortLegendData(profitDataRaw, 'work');

            // Для отрисовки графика используем только положительные значения 
            const chartProfitData = profitData.data.map(v => v >= 0 ? v : 0);
            
            const ctxProfit = document.getElementById('factProfitChart');
            if (ctxProfit) {
                new Chart(ctxProfit.getContext('2d'), {
                    type: 'pie',
                    data: {
                        labels: profitData.labels,
                        datasets: [{ data: chartProfitData, backgroundColor: profitData.backgroundColor }]
                    },
                    options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false } } },
                    plugins: [percentPlugin]
                });
                renderHtmlLegend('fact-profit-legend', profitData);
                updateTotalSum('fact-profit-total-sum', profitData.data);
            }

            // --- График 3: Разбивка затрат по Отделам ---
            const deptDataRaw = {
                labels: data.departments_chart.labels,
                data: data.departments_chart.data,
                backgroundColor: getColors(data.departments_chart.labels)
            };
            const deptData = sortLegendData(deptDataRaw, 'work');

            const ctxDept = document.getElementById('factDepartmentsChart');
            
            if (ctxDept) {
                new Chart(ctxDept.getContext('2d'), {
                    type: 'pie',
                    data: {
                        labels: deptData.labels,
                        datasets: [{ data: deptData.data, backgroundColor: deptData.backgroundColor }]
                    },
                    options: { 
                        responsive: true, maintainAspectRatio: false,
                        plugins: { legend: { display: false } }
                    },
                    plugins: [percentPlugin]
                });
                renderHtmlLegend('fact-departments-legend', deptData);
                updateTotalSum('fact-departments-total-sum', deptData.data);
            }
        })
        .catch(err => console.error('Ошибка загрузки графиков факта:', err));

    // Если мы загрузились сразу на вкладку Сводка (например, после фильтра)
    if (activeTab === 'summary' && document.getElementById('factTimelineChart')) {
        setTimeout(() => {
            if (!window.myFactTimelineChart) {
                updateFactTimeline();
            }
        }, 100);
    }
});

// === Временной график ФинФакта ===
let factIsSelecting = false;
let factSelStartPixel = null;
let factSelEndPixel = null;
let factSelStartIndex = null;
let factSelEndIndex = null;

window.updateFactTimeline = function() {
    const groupEl = document.getElementById('fact-timeline-group');
    const negativeEl = document.getElementById('fact-timeline-negative-expense');
    if (!groupEl) return;
    
    const group = groupEl.value;
    const isNegative = negativeEl ? negativeEl.checked : false;
    const ctxTimeline = document.getElementById('factTimelineChart');
    if (!ctxTimeline) return;

    const urlParams = new URLSearchParams(window.location.search);
    const factMode = urlParams.get('fact_mode') || 'confirmed';

    const contractID = window.location.pathname.split('/')[2];

    const isDark = document.documentElement.getAttribute('data-theme') === 'dark';
    const textColor = isDark ? '#e6e9ee' : '#2f3640';
    const gridColor = isDark ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.05)';
    
    const colorIncome = '#36a2eb';
    const colorExpense = '#ff6384';
    const colorBalance = '#2ecc71';
    const colorIncPeriod = isDark ? '#90caf9' : '#1976d2';
    const colorExpPeriod = isDark ? '#f48fb1' : '#c2185b';

    fetch(`/contracts/${contractID}/budget/fact/timeline.json?group=${group}&fact_mode=${factMode}`)
        .then(r => r.json())
        .then(data => {
            if (window.myFactTimelineChart) {
                window.myFactTimelineChart.destroy();
            }
            
            factIsSelecting = false;
            factSelStartPixel = null;
            factSelEndPixel = null;
            factSelStartIndex = null;
            factSelEndIndex = null;

            const labels = data.map(d => d.period);
            const sign = isNegative ? -1 : 1;

            const incomeCum = data.map(d => d.income_cum);
            const expenseCum = data.map(d => d.expense_cum * sign);
            const balanceCum = data.map(d => d.income_cum - d.expense_cum);
            const incomeMonth = data.map(d => d.income_month);
            const expenseMonth = data.map(d => d.expense_month * sign);

            const selectionPlugin = {
                id: 'selectionPlugin',
                afterDatasetsDraw(chart) {
                    const { ctx, chartArea: { top, bottom } } = chart;
                    
                    if (!factIsSelecting && chart.tooltip._active && chart.tooltip._active.length > 0) {
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

                    if (factSelStartPixel !== null && factSelEndPixel !== null) {
                        const start = Math.min(factSelStartPixel, factSelEndPixel);
                        const end = Math.max(factSelStartPixel, factSelEndPixel);
                        
                        ctx.save();
                        ctx.fillStyle = 'rgba(64, 115, 158, 0.2)';
                        ctx.fillRect(start, top, end - start, bottom - top);
                        
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

            window.myFactTimelineChart = new Chart(ctxTimeline.getContext('2d'), {
                type: 'line',
                data: {
                    labels: labels,
                    datasets: [
                        { label: 'Поступления (нараст.)', data: incomeCum, borderColor: colorIncome, backgroundColor: 'rgba(54, 162, 235, 0.1)', fill: false, tension: 0.2, pointRadius: 3, hidden: true },
                        { label: 'Затраты (нараст.)', data: expenseCum, borderColor: colorExpense, backgroundColor: 'rgba(255, 99, 132, 0.1)', fill: false, tension: 0.2, pointRadius: 3, hidden: true },
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
                        tooltip: { enabled: false }
                    },
                    scales: {
                        x: { ticks: { color: textColor }, grid: { color: gridColor } },
                        y: { ticks: { color: textColor }, grid: { color: gridColor } }
                    },
                    onHover: (event, elements) => {
                        if (!factIsSelecting && elements.length > 0) {
                            renderFactTimelineLegend(window.myFactTimelineChart, elements[0].index);
                        } else if (!factIsSelecting) {
                            renderFactTimelineLegend(window.myFactTimelineChart, null);
                        }
                    }
                },
                plugins: [selectionPlugin]
            });

            const chart = window.myFactTimelineChart;

            const getPixelAndIndex = (e) => {
                const rect = ctxTimeline.getBoundingClientRect();
                const x = e.clientX - rect.left;
                const chartArea = chart.chartArea;
                const boundedX = Math.max(chartArea.left, Math.min(x, chartArea.right));
                
                const xValue = chart.scales.x.getValueForPixel(boundedX);
                if (xValue === undefined) return { x: boundedX, index: null };

                let minDist = Infinity;
                let closestIndex = 0;
                
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

            const startSelection = (e) => {
                e.preventDefault();
                const { x, index } = getPixelAndIndex(e);
                if (index !== null) {
                    factIsSelecting = true;
                    factSelStartPixel = x;
                    factSelEndPixel = x;
                    factSelStartIndex = index;
                    factSelEndIndex = index;
                    chart.update();
                    renderFactTimelineLegend(chart, null);
                    
                    document.addEventListener('mousemove', onMouseMove);
                    document.addEventListener('mouseup', onMouseUp);
                }
            };

            const onMouseMove = (e) => {
                if (!factIsSelecting) return;
                const { x, index } = getPixelAndIndex(e);
                if (index !== null) {
                    factSelEndPixel = x;
                    factSelEndIndex = index;
                    chart.update();
                    renderFactTimelineLegend(chart, null);
                }
            };

            const onMouseUp = (e) => {
                if (factIsSelecting) {
                    factIsSelecting = false;
                    document.removeEventListener('mousemove', onMouseMove);
                    document.removeEventListener('mouseup', onMouseUp);
                    renderFactTimelineLegend(chart, null);
                }
            };

            ctxTimeline.onmousedown = startSelection;
            ctxTimeline.ontouchstart = startSelection;

            renderFactTimelineLegend(chart, null);
        })
        .catch(err => console.error('Ошибка загрузки графика динамики факта:', err));
};

function renderFactTimelineLegend(chart, hoverIndex = null) {
    const container = document.getElementById('fact-timeline-legend');
    if (!container) return;

    const labels = chart.data.labels;
    const datasets = chart.data.datasets;
    
    let html = '';

    // 1. БЛОК ВЫБОРА ПЕРЬЕВ
    html += '<div style="display: flex; flex-wrap: wrap; gap: 15px; justify-content: center; font-size: 14px; margin-bottom: 15px; padding-bottom: 10px; border-bottom: 1px solid var(--border-soft);">';
    datasets.forEach((ds, i) => {
        const isVisible = chart.isDatasetVisible(i);
        const opacity = isVisible ? '1' : '0.4';
        const textDecoration = isVisible ? 'none' : 'line-through';
        html += `
            <div style="display: flex; align-items: center; gap: 8px; cursor: pointer; opacity: ${opacity}; text-decoration: ${textDecoration};" 
                 onclick="toggleFactTimelineDataset(${i})">
                <span style="display: inline-block; width: 20px; height: 3px; background-color: ${ds.borderColor}; border-radius: 2px;"></span>
                <span>${ds.label}</span>
            </div>
        `;
    });
    html += '</div>';

    // 2. БЛОК ДАННЫХ
    if (factSelStartIndex !== null && factSelEndIndex !== null) {
        const startIdx = Math.min(factSelStartIndex, factSelEndIndex);
        const endIdx = Math.max(factSelStartIndex, factSelEndIndex);
        const periodLabel = `${labels[startIdx]} — ${labels[endIdx]}`;
        
        html += `<div style="background: var(--accent-soft); padding: 12px; border-radius: 6px; border-left: 4px solid var(--accent);">
            <strong style="display: block; margin-bottom: 10px; color: var(--text);">Выделенный период: ${periodLabel}</strong>
            <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 10px;">`;
        
        datasets.forEach((ds, i) => {
            if (chart.isDatasetVisible(i)) {
                let displayValue = 0;
                if (ds.label.includes('нараст')) {
                    displayValue = ds.data[endIdx];
                } else {
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

window.toggleFactTimelineDataset = function(index) {
    if (!window.myFactTimelineChart) return;
    const meta = window.myFactTimelineChart.getDatasetMeta(index);
    meta.hidden = meta.hidden === null ? !window.myFactTimelineChart.data.datasets[index].hidden : null;
    window.myFactTimelineChart.update();
    renderFactTimelineLegend(window.myFactTimelineChart, null);
};

if (document.getElementById('FactTimelineChart')) {
    updateTimeline();
}