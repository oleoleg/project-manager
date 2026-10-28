

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
    // 2. Формируем HTML для выбора этапа
    window.stageOptionsHTML = ``;
    const stageNodes = document.querySelectorAll('#stage-data div');
    stageNodes.forEach(node => {
        const guid = node.getAttribute('data-guid');
        const name = node.getAttribute('data-name');
        const stageNum = name.split('—')[0].trim(); 
        const isSelected = node.hasAttribute('data-selected') ? 'selected' : '';
        window.stageOptionsHTML += `<option value="${guid}" data-stage-num="${stageNum}" ${isSelected}>${name}</option>`;
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

    // 5. И только теперь сортируем (по типу операции)
    const typeBtn = Array.from(document.querySelectorAll('.filter-bar button')).find(b => b.textContent.includes('По типу операции'));
    if (typeBtn) {
        sortFactTable('type', typeBtn);
    }
});

function formatMoney(val) {
    if (!val && val !== 0) return '';
    const s = Math.abs(val).toFixed(2);
    const formatted = s.replace(/\B(?=(\d{3})+(?!\d))/g, '\u00A0');
    return (val < 0 ? '-' : '') + formatted + ' ₽';
}

function parseMoneyText(text) {
    if (!text) return 0;
    const clean = text.replace(/[^\d.,-]/g, '').replace(',', '.');
    const val = parseFloat(clean);
    return isNaN(val) ? 0 : val;
}

function recalculateTotals() {
    const table = document.getElementById('fact-table');
    if (!table) return;
    
    let totalIncome = 0;
    let totalExpense = 0;
    let confirmedIncome = 0;
    let confirmedExpense = 0;

    const groupHeaders = table.querySelectorAll('.group-header-row');
    
    groupHeaders.forEach(header => {
        const type = header.getAttribute('data-type');
        if (!type) return;
        
        let groupInc = 0;
        let groupExp = 0;
        
        const rows = table.querySelectorAll(`tr.fact-row`);
        rows.forEach(row => {
            const opTypeInput = row.querySelector('input[name="operation_type[]"]');
            if (opTypeInput && opTypeInput.value === type) {
                const incInput = row.querySelector('input[name="fact_income[]"]');
                const expInput = row.querySelector('input[name="fact_expense[]"]');
                
                const incVal = incInput ? parseMoneyText(incInput.value) : 0;
                const expVal = expInput ? parseMoneyText(expInput.value) : 0;
                
                groupInc += incVal;
                groupExp += expVal;

                const isConfirmed = row.classList.contains('confirmed');
                if (isConfirmed) {
                    confirmedIncome += incVal;
                    confirmedExpense += expVal;
                }
            }
        });

        const incCell = header.querySelector('.group-total-income');
        const expCell = header.querySelector('.group-total-expense');
        if (incCell) incCell.textContent = formatMoney(groupInc);
        if (expCell) expCell.textContent = formatMoney(groupExp);

        totalIncome += groupInc;
        totalExpense += groupExp;
    });

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

function sortFactTable(field, btnElement) {
    // Для факта колонка с названием — 2 (индекс 2), но т.к. перед ней есть еще № и Вид работы, 
    // мы передаем colspan 3 (первые 3 колонки: №, Вид, Наименование).
    // hasBalance: false, так как в факте нет колонки Баланс.
    sortTable('fact-table', field, btnElement, {
        labelColspan: 3,
        incIdx: 6, // Факт (Приход)
        expIdx: 7, // Факт (Расход)
        balIdx: -1,
        hasIncome: true,
        hasExpense: true,
        hasBalance: false
    });
         
        recalculateTotals();
    
}
