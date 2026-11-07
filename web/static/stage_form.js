// 1. Скрытие столбцов
(function () {
    const toggle = document.getElementById('costs-compact-toggle');
    const table = document.getElementById('costs-table');
    if (toggle && table) {
        toggle.addEventListener('change', () => {
            table.classList.toggle('compact', toggle.checked);
        });
    }
})();

// 2. Инициализация замочков
(function () {
    const body = document.getElementById('costs-body');
    if (body) {
        BudgetChecks.initLocks(body);
    }
})();

// 3. Пересчет и проверки
(function () {
    const body = document.getElementById('costs-body');
    if (!body) return;

    function checkWarnings() {
        const stageCostEl = document.querySelector('input[name="cost"]');
        const stageCost = BudgetChecks.num(stageCostEl ? stageCostEl.value : 0);
        
        let totalSale = 0;
        let totalExpenses = 0;

        body.querySelectorAll('.cost-row').forEach(r => {
            totalSale += 
                BudgetChecks.num(r.querySelector('[name="cost_amount"]').value) +
                BudgetChecks.num(r.querySelector('[name="cost_equipment_sale"]').value);
            
            totalExpenses += 
                BudgetChecks.num(r.querySelector('[name="cost_fot_plan"]').value) +
                BudgetChecks.num(r.querySelector('[name="cost_travel"]').value) +
                BudgetChecks.num(r.querySelector('[name="cost_equipment_cost"]').value) +
                BudgetChecks.num(r.querySelector('[name="cost_subcontract"]').value) +
                BudgetChecks.num(r.querySelector('[name="cost_commission"]').value) +
                BudgetChecks.num(r.querySelector('[name="cost_other"]').value);
        });

        const w1 = document.getElementById('costs-overrun-warning');
        if (w1) {
            if (stageCost > 0 && totalExpenses > stageCost) {
                w1.textContent = 'Внимание: затраты (' + totalExpenses.toFixed(2) + ' ₽) превышают стоимость этапа (' + stageCost.toFixed(2) + ' ₽). Этап убыточный.';
                w1.style.display = 'block';
            } else {
                w1.style.display = 'none';
            }
        }

        const w2 = document.getElementById('costs-sale-mismatch-warning');
        if (w2) {
            if (stageCost > 0 && totalSale > 0 && Math.abs(stageCost - totalSale) > 0.01) {
                w2.textContent = 'Внимание: сумма разложения по строкам (' + totalSale.toFixed(2) + ' ₽) не совпадает со стоимостью этапа (' + stageCost.toFixed(2) + ' ₽).';
                w2.style.display = 'block';
            } else {
                w2.style.display = 'none';
            }
        }
    }

    body.addEventListener('input', (e) => {
        const row = e.target.closest('.cost-row');
        if (!row) return;
        BudgetChecks.recalcRow(row, e.target.name);
        checkWarnings();
    });

    body.addEventListener('change', (e) => {
        const row = e.target.closest('.cost-row');
        if (!row) return;
        BudgetChecks.recalcRow(row, e.target.name);
        checkWarnings();
    });

    body.addEventListener('recalc-request', (e) => {
        const row = e.target.closest('.cost-row');
        if (!row) return;
        BudgetChecks.recalcRow(row, '');
    });

    const stageCostEl = document.querySelector('input[name="cost"]');
    if (stageCostEl) stageCostEl.addEventListener('input', checkWarnings);

    body.querySelectorAll('.cost-row').forEach(row => BudgetChecks.recalcRow(row, ''));
    checkWarnings();
})();

// 4. Добавление / удаление строк
(function () {
    const body = document.getElementById('costs-body');
    const addBtn = document.getElementById('cost-add');
    if (!body) return;

    function makeRow() {
        const tpl = body.querySelector('.cost-row');
        const row = tpl ? tpl.cloneNode(true) : document.createElement('tr');
        row.querySelectorAll('input').forEach(i => {
            if (i.type === 'hidden') { i.value = ''; return; }
            i.value = '';
        });
        row.querySelectorAll('select').forEach(s => { s.selectedIndex = 0; });
        row.querySelectorAll('.lock-btn').forEach(b => {
            b.classList.remove('locked');
            b.textContent = '🔓';
        });
        row.querySelector('.ratio-display') && (row.querySelector('.ratio-display').value = '');
        row.classList.add('cost-row');
        return row;
    }

    function reattachMoneyInputs(scope) {
        if (window.attachMoneyInput) {
            scope.querySelectorAll('input.money-input').forEach(window.attachMoneyInput);
        }
    }

    if (addBtn) {
        addBtn.addEventListener('click', () => {
            const row = makeRow();
            body.appendChild(row);
            reattachMoneyInputs(row);
        });
    }

    body.addEventListener('click', (e) => {
        if (e.target.classList.contains('cost-remove')) {
            const rows = body.querySelectorAll('.cost-row');
            if (rows.length > 1) {
                e.target.closest('tr').remove();
            } else {
                alert('Должна остаться хотя бы одна строка затрат.');
            }
        }
    });
})();

// 5. Переключатель Закрыто по актам
(function () {
    const btn      = document.getElementById('closed-by-acts-lock');
    const checkbox = document.getElementById('closed-by-acts-manual');
    const input    = document.getElementById('closed-by-acts-input');
    if (!btn || !checkbox || !input) return;

    btn.addEventListener('click', () => {
        checkbox.checked = !checkbox.checked;

        if (checkbox.checked) {
            input.readOnly = false;
            btn.textContent = '🔒';
            btn.classList.add('locked');
            btn.title = 'Сейчас ручной ввод. Нажмите, чтобы подтягивать из актов';
        } else {
            input.readOnly = true;
            btn.textContent = '🔗';
            btn.classList.remove('locked');
            btn.title = 'Сейчас подтяжка из актов. Нажмите для ручного ввода';
            loadActSum(input);
        }
    });

    function loadActSum(input) {
        const url = btn.dataset.actSumUrl;
        if (!url) return;
        fetch(url)
            .then(r => r.json())
            .then(data => {
                const sum = data.sum || 0;
                if (window.inputMoney) {
                    input.value = window.inputMoney(String(sum.toFixed(2)));
                } else {
                    input.value = sum.toFixed(2);
                }
            })
            .catch((err) => console.error('Ошибка загрузки суммы актов:', err));
    }
})();

// 6. Автоматическое раскрытие блока затрат, если в URL есть ?costs=1
document.addEventListener('DOMContentLoaded', function() {
    const urlParams = new URLSearchParams(window.location.search);
    const costsBlock = document.getElementById('section-costs');
    
    // Если в URL есть ?costs=1 и блок с затратами существует
    if (urlParams.get('costs') === '1' && costsBlock) {
        // Прокручиваем страницу к этому блоку
        costsBlock.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
});