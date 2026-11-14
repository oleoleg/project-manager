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
});

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
 