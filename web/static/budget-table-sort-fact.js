// web/static/budget-table-sort-fact.js

function sortFactTable(field, btnElement) {
    // Порядок видов работ (как в ФинПлане)
    const workOrder = { "ПИР": 1, "ППО": 2, "СМР": 3, "ПНР": 4, "Продажа": 5, "ТО": 6, "Ремонт": 7, "Сборка": 8, "Комплексный": 9 };

    sortTable('fact-table', field, btnElement, {
        labelColspan: 4,
        incIdx: 7,
        expIdx: 8,
        balIdx: -1,
        hasIncome: true,
        hasExpense: true,
        hasBalance: false,
        workOrder: workOrder
    });
         
    recalculateTotals();
}

function sortTable(tableId, field, btnElement, groupConfig) {
    const table = document.getElementById(tableId);
    if (!table) return;
    const tbody = table.querySelector('tbody');
    if (!tbody) return;

    if (btnElement && btnElement.parentNode) {
        btnElement.parentNode.querySelectorAll('button').forEach(btn => {
            btn.classList.remove('btn-primary');
            btn.style.backgroundColor = '';
            btn.style.color = '';
        });
        btnElement.classList.add('btn-primary');
        btnElement.style.backgroundColor = '#337ab7';
        btnElement.style.color = '#fff';
    }

    // Удаляем только динамические заголовки
    tbody.querySelectorAll('.group-header-row.dynamic').forEach(el => el.remove());
    
    // Собираем строки данных, ИСКЛЮЧАЯ заголовки и кнопки добавления
    const allRows = Array.from(tbody.querySelectorAll('tr.fact-row, tr:not(.group-header-row):not(.group-add-row)'));

    const typeOrder = {
        "Прибыль": 1, "Платежи по договору": 2, "Закупка по договору": 3, "Зарплата сотрудникам": 4,
        "Накладные": 5, "Командировки": 6, "Субподрядчики": 7, "Комиссия": 8, "Другое (расходы)": 9
    };
    const workOrder = groupConfig.workOrder || {};

    allRows.sort((a, b) => {
        if (field === 'stage') {
            const sA = parseInt(a.dataset.order) || 0;
            const sB = parseInt(b.dataset.order) || 0;
            if (sA !== sB) return sA - sB;
            return parseInt(a.dataset.date) - parseInt(b.dataset.date);
        }
        if (field === 'date') return parseInt(a.dataset.date) - parseInt(b.dataset.date);
        if (field === 'type') {
            const diff = (typeOrder[a.dataset.type] || 99) - (typeOrder[b.dataset.type] || 99);
            if (diff !== 0) return diff;
            const sA = parseInt(a.dataset.stage) || 0;
            const sB = parseInt(b.dataset.stage) || 0;
            if (sA !== sB) return sA - sB;
            return parseInt(a.dataset.date) - parseInt(b.dataset.date);
        }
        if (field === 'work') {
            const wA = workOrder[a.dataset.work] || 99;
            const wB = workOrder[b.dataset.work] || 99;
            if (wA !== wB) return wA - wB; // Сортируем по весу
            
            // Если виды работ одинаковые, сортируем по типу операции
            const tDiff = (typeOrder[a.dataset.type] || 99) - (typeOrder[b.dataset.type] || 99);
            if (tDiff !== 0) return tDiff;
            
            // Затем по этапу
            const sA = parseInt(a.dataset.order) || 0;
            const sB = parseInt(b.dataset.order) || 0;
            if (sA !== sB) return sA - sB;
            
            return parseInt(a.dataset.date) - parseInt(b.dataset.date);
        }
        return 0;
    });

    tbody.innerHTML = '';
    const months = ["Январь", "Февраль", "Март", "Апрель", "Май", "Июнь", "Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь"];

    let currentGroupValue = null;
    let currentGroupLabel = '';
    let currentGroupRows = [];
    let tableIncome = 0;
    let tableExpense = 0;
    let tableBalance = 0;

    function flushGroup() {
        if (currentGroupRows.length === 0) return;
        
        let gInc = 0, gExp = 0, gBal = 0;
        currentGroupRows.forEach(row => {
            const incInput = row.querySelector('input[name="fact_income[]"]');
            const expInput = row.querySelector('input[name="fact_expense[]"]');
            
            let inc = 0, exp = 0, bal = 0;
            
            if (incInput && expInput) {
                inc = parseMoneyText(incInput.value);
                exp = parseMoneyText(expInput.value);
            } else {
                const incCell = row.cells[groupConfig.incIdx];
                const expCell = row.cells[groupConfig.expIdx];
                if (incCell) inc = parseMoneyText(incCell.textContent);
                if (expCell) exp = parseMoneyText(expCell.textContent);
            }
            
            const balCell = row.cells[groupConfig.balIdx];
            if (balCell) bal = parseMoneyText(balCell.textContent);
            
            gInc += inc; gExp += exp; gBal += bal;
            tableIncome += inc; tableExpense += exp; tableBalance += bal;
        });

        if (field === 'stage') {
            if (currentGroupRows.length > 0) {
                const firstRow = currentGroupRows[0];
                if (firstRow.dataset.stageName) {
                    currentGroupLabel = firstRow.dataset.stageName;
                } else {
                    currentGroupLabel = 'Этап ' + (currentGroupValue || '—');
                }
            } else {
                currentGroupLabel = 'Этап ' + (currentGroupValue || '—');
            }
        }
        
        // Проверяем, есть ли в группе ТОЛЬКО строки ЗП/Накладные
        let hasOnlySalaryRows = true;
        currentGroupRows.forEach(row => {
            if (!row.classList.contains('salary-row')) {
                hasOnlySalaryRows = false; // Нашли обычную строку
            }
        });

        const headerRow = document.createElement('tr');
        headerRow.className = 'group-header-row dynamic';
        
        // Скрываем заголовок, ТОЛЬКО если в группе нет ничего, кроме ЗП
        if (hasOnlySalaryRows) {
            headerRow.className += ' salary-row';
            
            // Проверяем, включен ли чекбокс "Показать ЗП"
            const showSalaryCheckbox = document.getElementById('show-salary');
            const isShowSalary = showSalaryCheckbox ? showSalaryCheckbox.checked : false;
            
            // Показываем заголовок, только если чекбокс включен. Иначе прячем.
            headerRow.style.display = isShowSalary ? '' : 'none';
        }
        headerRow.setAttribute('data-type', field === 'type' ? currentGroupValue : '');
        
        headerRow.innerHTML = `
            <td colspan="${groupConfig.labelColspan}" style="background-color: var(--surface-alt); font-weight: bold; color: var(--accent); padding: 10px; border-top: 2px solid var(--border); border-bottom: 1px solid var(--border);">${currentGroupLabel}</td>
            <td colspan="${groupConfig.incIdx - groupConfig.labelColspan}" style="background-color: var(--surface-alt); border-top: 2px solid var(--border); border-bottom: 1px solid var(--border);"></td>
            ${groupConfig.hasIncome ? `<td class="num group-total-income" style="background-color: var(--surface-alt); color: var(--accent); font-weight: bold; padding: 10px; border-bottom: 1px solid var(--border);">${gInc !== 0 ? formatMoney(gInc) : ''}</td>` : ''}
            ${groupConfig.hasExpense ? `<td class="num group-total-expense" style="background-color: var(--surface-alt); color: var(--accent); font-weight: bold; padding: 10px; border-bottom: 1px solid var(--border);">${gExp !== 0 ? formatMoney(gExp) : ''}</td>` : ''}
            ${groupConfig.hasBalance ? `<td class="num group-total-balance" style="background-color: var(--surface-alt); color: var(--accent); font-weight: bold; padding: 10px; border-bottom: 1px solid var(--border);">${gBal !== 0 ? formatMoney(gBal) : ''}</td>` : ''}
            <td style="background-color: var(--surface-alt); border-top: 2px solid var(--border); border-bottom: 1px solid var(--border);"></td>
        `;
        
        tbody.appendChild(headerRow);
        currentGroupRows.forEach(row => tbody.appendChild(row));
        
        // Проверяем, есть ли в группе хотя бы одна строка, не относящаяся к ЗП/Накладным
        let hasVisibleRows = false;
        currentGroupRows.forEach(row => {
            if (!row.classList.contains('salary-row')) {
                hasVisibleRows = true;
            }
        });

        // Определяем, нужно ли показывать кнопку добавления
        let showAddButton = true;

        // Скрываем кнопку ТОЛЬКО если сортируем по Типу операции и это ЗП/Накладные
        if (field === 'type') {
            if (currentGroupValue === 'Зарплата сотрудникам' || currentGroupValue === 'Накладные') {
                showAddButton = false;
            }
        } else {
            // Если сортируем по дате/этапу/работе, проверяем, есть ли ВИДИМЫЕ строки в группе
            // (с учетом того, что ЗП могут быть скрыты)
            const hasVisibleRows = currentGroupRows.some(r => r.style.display !== 'none');
            if (!hasVisibleRows) {
                showAddButton = false;
            }
        }

        // Дополнительная проверка: если все строки в группе скрыты (ЗП), но мы не в режиме "По типу", 
        // все равно показываем кнопку, чтобы можно было добавить обычную операцию в этот месяц/этап
        if (showAddButton) {
            let groupOpType = 'Платежи по договору'; 
            
            if (field === 'type') {
                groupOpType = currentGroupValue;
            } else {
                // Берем тип из ПОСЛЕДНЕЙ ВИДИМОЙ строки (не ЗП), чтобы кнопка добавляла правильный тип
                let lastVisibleRow = null;
                for (let i = currentGroupRows.length - 1; i >= 0; i--) {
                    const r = currentGroupRows[i];
                    if (r.style.display !== 'none' && !r.classList.contains('salary-row')) {
                        lastVisibleRow = r;
                        break;
                    }
                }
                if (!lastVisibleRow) lastVisibleRow = currentGroupRows[0]; 

                if (lastVisibleRow) {
                    const select = lastVisibleRow.querySelector('.op-type-select');
                    if (select) {
                        groupOpType = select.value;
                    } else {
                        groupOpType = lastVisibleRow.dataset.type || 'Платежи по договору';
                    }
                }
            }
            
            const addRow = document.createElement('tr');
            addRow.className = 'group-add-row';
            // ВАЖНО: сохраняем данные группы в кнопку, чтобы addNewRow знал, откуда брать данные
            addRow.dataset.groupType = groupOpType;
            if (field === 'stage') addRow.dataset.groupStage = currentGroupValue;
            if (field === 'date') addRow.dataset.groupDate = currentGroupValue;
            if (field === 'work') addRow.dataset.groupWork = currentGroupValue;

            const totalCols = groupConfig.labelColspan + 
                              (groupConfig.incIdx - groupConfig.labelColspan) + 
                              (groupConfig.hasIncome ? 1 : 0) + 
                              (groupConfig.hasExpense ? 1 : 0) + 
                              (groupConfig.hasBalance ? 1 : 0) + 1;
            
            // Берем данные из ПОСЛЕДНЕЙ видимой строки в группе, ИГНОРИРУЯ ЗП и Накладные
            let lastRow = null;
            for (let i = currentGroupRows.length - 1; i >= 0; i--) {
                const r = currentGroupRows[i];
                // Пропускаем скрытые и ЗП/Накладные
                if (r.style.display !== 'none' && !r.classList.contains('salary-row')) {
                    lastRow = r;
                    break;
                }
            }
            // Если нашли только ЗП, берем любую строку (чтобы не упасть)
            if (!lastRow && currentGroupRows.length > 0) {
                lastRow = currentGroupRows[0];
            }

            // Берем отсрочку из последней строки
            let lastDelay = 0;
            if (lastRow) {
                const delayInput = lastRow.querySelector('input[name="delay_days[]"]');
                if (delayInput) lastDelay = delayInput.value;
            }

            // Определяем этап (если сортируем по этапам, берем из группы, иначе из последней строки)
            let groupStage = '';
            let groupStageName = '';
            if (field === 'stage') {
                groupStage = currentGroupValue;
                groupStageName = currentGroupLabel; // Например, "1 — Проектирование"
            } else if (lastRow) {
                // Берем из data-атрибута последней строки
                groupStage = lastRow.dataset.stage || '';
                groupStageName = lastRow.dataset.stageName || '';
            }

            // Определяем дату (если сортируем по дате, берем из группы, иначе из последней строки)
            let groupDate = '';
            if (field === 'date') {
                groupDate = currentGroupValue;
            } else if (lastRow) {
                groupDate = lastRow.dataset.date || '';
            }

            // Определяем вид работы (если сортируем по виду работ, берем из группы, иначе из последней строки)
            let groupWork = '';
            if (field === 'work') {
                groupWork = currentGroupValue;
            } else if (lastRow) {
                groupWork = lastRow.dataset.work || '';
            }
            
            // Ищем ПОСЛЕДНЮЮ видимую строку (не ЗП) в группе
            let sourceRow = null;
            for (let i = currentGroupRows.length - 1; i >= 0; i--) {
                const r = currentGroupRows[i];
                if (r.style.display !== 'none' && !r.classList.contains('salary-row')) {
                    sourceRow = r;
                    break;
                }
            }
            if (!sourceRow && currentGroupRows.length > 0) {
                sourceRow = currentGroupRows[0];
            }

            // Собираем данные для кнопки
            let groupData = {
                type: groupOpType, // Тип операции (текст кнопки)
                stage: sourceRow ? (sourceRow.dataset.stage || '') : '', 
                stageName: sourceRow ? (sourceRow.dataset.stageName || '') : '',
                work: sourceRow ? (sourceRow.dataset.work || '') : '',
                delay: sourceRow ? (sourceRow.querySelector('input[name="delay_days[]"]')?.value || '0') : '0',
                date: sourceRow ? (sourceRow.querySelector('input[name="operation_date[]"]')?.value || '') : '' // Берем дату из последней строки
            };
            
            const jsonStr = JSON.stringify(groupData).replace(/"/g, '&quot;');

            addRow.innerHTML = `
                <td colspan="${totalCols}" style="padding: 5px 5px;">
                    <button type="button" class="btn btn-sm" style="opacity: 0.7;" 
                            onclick="addNewRowFromGroup(this, ${jsonStr})">
                        + Добавить операцию
                    </button>
                </td>
            `;
            tbody.appendChild(addRow);
        }
        
        currentGroupRows = [];
    }

    allRows.forEach((row) => {
        let gv = '', gl = '';
        if (field === 'date') {
            const d = new Date(parseInt(row.dataset.date) * 1000);
            gv = `${d.getFullYear()}-${d.getMonth()}`; // Уникальный ключ для месяца
            gl = `${months[d.getMonth()]} ${d.getFullYear()}`;
        } else if (field === 'type') {
            gv = row.dataset.type; gl = row.dataset.type;
        } else if (field === 'work') {
            gv = row.dataset.work; 
            gl = row.cells[1].textContent.trim();
        } else if (field === 'stage') {
            gv = row.dataset.stage;
            gl = ''; // Оставляем пустым, сформируем в flushGroup
        }

        if (gv !== currentGroupValue) {
            flushGroup();
            currentGroupValue = gv;
            currentGroupLabel = gl;
        }
        currentGroupRows.push(row);
    });

    flushGroup();

    if (document.getElementById('total-income')) {
        document.getElementById('total-income').textContent = formatMoney(tableIncome);
        document.getElementById('total-expense').textContent = formatMoney(tableExpense);
        document.getElementById('total-balance').textContent = formatMoney(tableBalance);
    }
}

