document.addEventListener('DOMContentLoaded', function() {
    const body = document.getElementById('costs-body');
    if (!body) return;

    // 1. Функция обновления визуального состояния (auto / не auto)
    function updateRowAutoState(row) {
        const calcFieldHidden = row.querySelector('input[name="cost_calc_field"]');
        const calcField = calcFieldHidden ? calcFieldHidden.value : '';

        const fields = {
            'fot_plan': row.querySelector('input[name="cost_fot_plan"]'),
            'labor_days': row.querySelector('input[name="cost_labor_days"]'),
            'rate_per_day': row.querySelector('input[name="cost_rate_per_day"]')
        };

        // Снимаем readonly и классы со всех
        Object.values(fields).forEach(el => {
            if (!el) return;
            el.readOnly = false;
            el.classList.remove('is-auto');
            el.removeAttribute('tabindex');
        });

        // Убираем активные классы со всех кнопок
        row.querySelectorAll('.auto-toggle-btn').forEach(b => b.classList.remove('active'));

        // Применяем к выбранному полю
        if (fields[calcField]) {
            fields[calcField].readOnly = true;
            fields[calcField].setAttribute('tabindex', '-1');
            fields[calcField].classList.add('is-auto');
            
            const btn = fields[calcField].parentNode.querySelector('.auto-toggle-btn');
            if (btn) btn.classList.add('active');
        }
    }

    // 2. Логика переключения "auto" при клике на кнопку [A]
    body.addEventListener('click', (e) => {
        const btn = e.target.closest('.auto-toggle-btn');
        if (!btn) return;

        e.preventDefault();
        const row = btn.closest('.cost-row');
        const calcFieldHidden = row.querySelector('input[name="cost_calc_field"]');
        if (!calcFieldHidden) return;

        const td = btn.closest('td');
        const input = td.querySelector('input:not([type="hidden"])');
        if (!input) return;
        
        const fieldName = input.name;

        // Если кнопка уже активна (auto уже стоит здесь) — ничего не делаем
        if (btn.classList.contains('active')) {
            return;
        }

        // Иначе переносим auto на это поле
        if (fieldName === 'cost_fot_plan') calcFieldHidden.value = 'fot_plan';
        else if (fieldName === 'cost_labor_days') calcFieldHidden.value = 'labor_days';
        else if (fieldName === 'cost_rate_per_day') calcFieldHidden.value = 'rate_per_day';

        updateRowAutoState(row);
        
        if (window.StageCostsChecks) {
            StageCostsChecks.recalcRow(row);
        }
        checkWarnings();
    });

    // 3. Пересчет при вводе в поле (если оно не readonly)
    body.addEventListener('input', (e) => {
        const row = e.target.closest('.cost-row');
        if (!row) return;
        if (e.target.readOnly) return;

        setTimeout(() => {
            if (window.StageCostsChecks) {
                StageCostsChecks.recalcRow(row);
            }
            updateRowDisplay(row);
        }, 0);
    });

    // 4. Обновление коэффициента и предупреждений
    function updateRowDisplay(row) {
        const amountEl = row.querySelector('input[name="cost_amount"]');
        const fotEl = row.querySelector('input[name="cost_fot_plan"]');
        const ratioEl = row.querySelector('.ratio-display');
        
        if (ratioEl && amountEl && fotEl) {
            const amount = parseMoneyText(amountEl.value);
            const fot = parseMoneyText(fotEl.value);
            ratioEl.value = amount > 0 ? (fot / amount).toFixed(2) : '';
        }
        checkWarnings();
    }

    function checkWarnings() {
        const stageCostEl = document.querySelector('input[name="cost"]');
        const stageCost = parseMoneyText(stageCostEl ? stageCostEl.value : 0);
        
        let totalSale = 0;
        let totalExpenses = 0;

        body.querySelectorAll('.cost-row').forEach(r => {
            totalSale += 
                parseMoneyText(r.querySelector('[name="cost_amount"]').value) +
                parseMoneyText(r.querySelector('[name="cost_equipment_sale"]').value);
            
            totalExpenses += 
                parseMoneyText(r.querySelector('[name="cost_fot_plan"]').value) +
                parseMoneyText(r.querySelector('[name="cost_travel"]').value) +
                parseMoneyText(r.querySelector('[name="cost_equipment_cost"]').value) +
                parseMoneyText(r.querySelector('[name="cost_subcontract"]').value) +
                parseMoneyText(r.querySelector('[name="cost_commission"]').value) +
                parseMoneyText(r.querySelector('[name="cost_other"]').value);
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

    // 5. Добавление строк
    const addBtn = document.getElementById('cost-add');
    if (addBtn) {
        addBtn.addEventListener('click', () => {
            const tpl = body.querySelector('.cost-row');
            const row = tpl ? tpl.cloneNode(true) : document.createElement('tr');
            
            row.querySelectorAll('input').forEach(i => {
                if (i.type === 'hidden' && i.name === 'cost_calc_field') {
                    i.value = 'fot_plan';
                } else if (i.type === 'hidden') {
                    i.value = '';
                } else {
                    i.value = '';
                }
            });
            row.querySelectorAll('select').forEach(s => { s.selectedIndex = 0; });
            
            row.querySelectorAll('input').forEach(el => el.classList.remove('is-auto'));
            row.querySelectorAll('.auto-toggle-btn').forEach(b => b.classList.remove('active'));
            
            const fotInput = row.querySelector('input[name="cost_fot_plan"]');
            const fotBtn = row.querySelector('.cell-fot .auto-toggle-btn');
            if (fotInput && fotBtn) {
                fotInput.readOnly = true;
                fotInput.classList.add('is-auto');
                fotBtn.classList.add('active');
            }
            
            row.querySelector('.ratio-display') && (row.querySelector('.ratio-display').value = '');
            
            body.appendChild(row);
            
            if (window.attachMoneyInput) {
                row.querySelectorAll('input.money-input').forEach(window.attachMoneyInput);
            }
            
            if (window.StageCostsChecks) {
                StageCostsChecks.recalcRow(row);
            }
            checkWarnings();
        });
    }

    // 6. Удаление строк
    body.addEventListener('click', (e) => {
        if (e.target.classList.contains('cost-remove')) {
            const rows = body.querySelectorAll('.cost-row');
            if (rows.length > 1) {
                e.target.closest('tr').remove();
                checkWarnings();
            } else {
                alert('Должна остаться хотя бы одна строка затрат.');
            }
        }
    });

    // 7. Переключатель "Закрыто по актам" (ВАШ КОД)
    const btn = document.getElementById('closed-by-acts-lock');
    const checkbox = document.getElementById('closed-by-acts-manual');
    const input = document.getElementById('closed-by-acts-input');
    if (btn && checkbox && input) {
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
    }

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
    
    // 8. Раскрытие блока при ?costs=1
    const urlParams = new URLSearchParams(window.location.search);
    const costsBlock = document.getElementById('section-costs');
    if (urlParams.get('costs') === '1' && costsBlock) {
        costsBlock.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }

    // ПЕРВОНАЧАЛЬНАЯ ИНИЦИАЛИЗАЦИЯ (в том числе для НОВОГО этапа)
    body.querySelectorAll('.cost-row').forEach(row => {
        // Если calc_field пустой, ставим fot_plan (по умолчанию)
        const calcFieldHidden = row.querySelector('input[name="cost_calc_field"]');
        if (calcFieldHidden && calcFieldHidden.value === '') {
            calcFieldHidden.value = 'fot_plan';
        }

        updateRowAutoState(row);
        if (window.StageCostsChecks) {
            StageCostsChecks.recalcRow(row);
        }
        updateRowDisplay(row);
    });
    checkWarnings();
});