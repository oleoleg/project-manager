// web/static/table-sort.js

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
        "Платежи по договору": 1, "Закупка по договору": 2, "Зарплата сотрудникам": 3,
        "Накладные": 4, "Командировки": 5, "Субподрядчики": 6, "Комиссия": 7, "Другое (расходы)": 8
    };
    const workOrder = { "pir": 1, "ppo": 2, "smr": 3, "pnr": 4, "sale": 5, "to": 6, "remont": 7, "complex": 8 };

    allRows.sort((a, b) => {
        if (field === 'stage') {
            const sA = parseInt(a.dataset.stage) || 0;
            const sB = parseInt(b.dataset.stage) || 0;
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
            const wDiff = (workOrder[a.dataset.work] || 99) - (workOrder[b.dataset.work] || 99);
            if (wDiff !== 0) return wDiff;
            const tDiff = (typeOrder[a.dataset.type] || 99) - (typeOrder[b.dataset.type] || 99);
            if (tDiff !== 0) return tDiff;
            return parseInt(a.dataset.date) - parseInt(b.dataset.date);
        }
        return 0;
    });

    // Очищаем tbody от всего
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
            const incCell = row.cells[groupConfig.incIdx];
            const expCell = row.cells[groupConfig.expIdx];
            const balCell = row.cells[groupConfig.balIdx];

            const inc = incCell ? parseMoneyText(incCell.textContent) : 0;
            const exp = expCell ? parseMoneyText(expCell.textContent) : 0;
            const bal = balCell ? parseMoneyText(balCell.textContent) : 0;
            
            gInc += inc; gExp += exp; gBal += bal;
            tableIncome += inc; tableExpense += exp; tableBalance += bal;
        });

        const headerRow = document.createElement('tr');
        headerRow.className = 'group-header-row dynamic';
        headerRow.setAttribute('data-type', field === 'type' ? currentGroupValue : '');
        headerRow.innerHTML = `
            <td colspan="${groupConfig.labelColspan}" style="background-color: var(--surface-alt); font-weight: bold; color: var(--accent); padding: 10px; border-top: 2px solid var(--border); border-bottom: 1px solid var(--border);">${currentGroupLabel}</td>
            <td colspan="${groupConfig.incIdx - groupConfig.labelColspan}" style="background-color: var(--surface-alt); border-top: 2px solid var(--border); border-bottom: 1px solid var(--border);"></td>
            ${groupConfig.hasIncome ? `<td class="num group-total-income" style="background-color: var(--surface-alt); color: var(--accent); font-weight: bold; padding: 10px; border-bottom: 1px solid var(--border);"></td>` : ''}
            ${groupConfig.hasExpense ? `<td class="num group-total-expense" style="background-color: var(--surface-alt); color: var(--accent); font-weight: bold; padding: 10px; border-bottom: 1px solid var(--border);"></td>` : ''}
            ${groupConfig.hasBalance ? `<td class="num group-total-balance" style="background-color: var(--surface-alt); color: var(--accent); font-weight: bold; padding: 10px; border-bottom: 1px solid var(--border);"></td>` : ''}
            <td style="background-color: var(--surface-alt); border-top: 2px solid var(--border); border-bottom: 1px solid var(--border);"></td>
        `;
        tbody.appendChild(headerRow);
        currentGroupRows.forEach(row => tbody.appendChild(row));
        
         // ... заполнение ячеек
        const incDiv = headerRow.querySelector('.group-total-income');
        const expDiv = headerRow.querySelector('.group-total-expense');
        const balDiv = headerRow.querySelector('.group-total-balance');
        if (incDiv) incDiv.textContent = formatMoney(gInc);
        if (expDiv) expDiv.textContent = formatMoney(gExp);
        if (balDiv) balDiv.textContent = formatMoney(gBal);
        tbody.appendChild(headerRow);
        currentGroupRows.forEach(row => tbody.appendChild(row));

        // Возвращаем кнопку
        const isSalaryGroup = currentGroupLabel === 'Зарплата сотрудникам' || currentGroupLabel === 'Накладные';
        if (field === 'type' && !isSalaryGroup) {
            const addRow = document.createElement('tr');
            addRow.className = 'group-add-row';
            addRow.innerHTML = `
                <td colspan="10" style="padding: 5px 10px;">
                    <button type="button" class="btn btn-sm" style="opacity: 0.7;" onclick="addNewRow('${currentGroupLabel}', event)">+ Добавить операцию</button>
                </td>
            `;
            tbody.appendChild(addRow);
        }
        
        currentGroupRows = [];
    }

    allRows.forEach((row) => {
        let gv = '', gl = '';
        if (field === 'date') {
            gv = row.dataset.date; 
            const d = new Date(parseInt(row.dataset.date) * 1000);
            gl = `${months[d.getMonth()]} ${d.getFullYear()}`;
        } else if (field === 'type') {
            gv = row.dataset.type; gl = row.dataset.type;
        } else if (field === 'work') {
            gv = row.dataset.work; 
            gl = row.cells[1].textContent.trim();
        } else if (field === 'stage') {
            gv = row.dataset.stage;
            gl = 'Этап ' + (row.dataset.stage || '—');
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

// Вспомогательные функции
function parseMoneyText(text) {
    if (!text) return 0;
    // Заменяем неразрывные пробелы (\u00A0) на обычные
    const clean = text.replace(/\u00A0/g, ' ').replace(/[^\d.,-]/g, '').replace(',', '.');
    const val = parseFloat(clean);
    return isNaN(val) ? 0 : val;
}

function formatMoney(val) {
    if (!val && val !== 0) return '';
    const s = Math.abs(val).toFixed(2);
    const formatted = s.replace(/\B(?=(\d{3})+(?!\d))/g, '\u00A0');
    return (val < 0 ? '-' : '') + formatted + ' ₽';
}