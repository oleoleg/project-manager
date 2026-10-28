// budget_checks.js
window.BudgetChecks = (function() {
    // Парсер чисел (всегда возвращает положительное число)
    function num(s) {
        if (!s) return 0;
        s = String(s).replace(/[^\d.-]/g, '');
        let v = parseFloat(s);
        if (isNaN(v)) return 0;
        return Math.abs(v);
    }

    // Присваивает значение полю и, если это деньги, форматирует его (если есть функция)
    function setVal(el, v) {
        v = Math.abs(v);
        
        // Если это дни — пишем целое число
        if (el.classList.contains('days-input')) {
            const val = Math.round(v);
            el.value = val > 0 ? val : '';
            return;
        }

        // Если это деньги
        if (v > 0) el.value = v.toFixed(2);
        else el.value = '';
        
        // Вызываем глобальный форматтер (из app.js), если он есть
        if (window.formatMoney) {
            el.value = window.formatMoney(el.value);
        }
    }

    function recalcRow(row, changedName) {
        const fotEl  = row.querySelector('[name="cost_fot_plan"], [name="cost_fot_plan[]"]');
        const daysEl = row.querySelector('[name="cost_labor_days"], [name="cost_labor_days[]"]');
        const rateEl = row.querySelector('[name="cost_rate_per_day"], [name="cost_rate_per_day[]"]');
        const lockEl = row.querySelector('input[name="cost_locked"], input[name="cost_locked[]"]');
        
        const amountEl = row.querySelector('[name="cost_amount"], [name="cost_amount[]"]');
        const ratioEl  = row.querySelector('.ratio-display');
        
        if (ratioEl && amountEl && fotEl) {
            const amount = num(amountEl.value);
            const fot = num(fotEl.value);
            ratioEl.value = amount > 0 ? (fot / amount).toFixed(2) : '';
        }

        if (!lockEl || !fotEl || !daysEl || !rateEl) return; 

        const locks = (lockEl.value || '').split(',').map(s => s.trim()).filter(Boolean);
        const hasFOT  = locks.includes('fot_plan');
        const hasDays = locks.includes('labor_days');
        const hasRate = locks.includes('rate_per_day');

        const days = Math.round(num(daysEl.value)); 
        const rate = num(rateEl.value);
        const fot  = num(fotEl.value);

        const field = changedName.replace('[]', '');

        if (hasFOT && hasDays) {
            if (days > 0) setVal(rateEl, fot / days);
        } else if (hasFOT && hasRate) {
            if (rate > 0) setVal(daysEl, fot / rate);
        } else if (hasDays && hasRate) {
            setVal(fotEl, days * rate);
        } else if (hasFOT) {
            if (field === 'cost_labor_days') {
                if (days > 0) setVal(rateEl, fot / days);
            } else if (field === 'cost_rate_per_day') {
                if (rate > 0) setVal(daysEl, fot / rate);
            }
        } else if (hasDays) {
            if (field === 'cost_fot_plan') {
                if (days > 0) setVal(rateEl, fot / days);
            } else if (field === 'cost_rate_per_day') {
                setVal(fotEl, days * rate);
            }
        } else if (hasRate) {
            if (field === 'cost_labor_days') {
                setVal(fotEl, rate * days);
            } else if (field === 'cost_fot_plan') {
                if (rate > 0) setVal(daysEl, fot / rate);
            }
        } else {
            // БЕЗ ЗАМОЧКОВ
            if (field === 'cost_fot_plan') {
                if (rate > 0) setVal(daysEl, fot / rate);
            } else if (field === 'cost_labor_days' || field === 'cost_rate_per_day') {
                if (days > 0 && rate > 0) setVal(fotEl, days * rate);
            }
        }
    }

    function initLocks(body) {
        if (!body) return;

        function refreshRow(row) {
            const hidden = row.querySelector('input[name="cost_locked"], input[name="cost_locked[]"]');
            if (!hidden) return;
            const set = (hidden.value || '').split(',').map(s => s.trim()).filter(Boolean);
            row.querySelectorAll('.lock-btn').forEach(b => {
                if (set.includes(b.dataset.field)) {
                    b.classList.add('locked');
                    b.textContent = '🔒';
                } else {
                    b.classList.remove('locked');
                    b.textContent = '🔓';
                }
            });
        }

        body.addEventListener('click', (e) => {
            const btn = e.target.closest('.lock-btn');
            if (!btn) return;
            e.preventDefault();

            const row = btn.closest('.cost-row');
            const hidden = row.querySelector('input[name="cost_locked"], input[name="cost_locked[]"]');
            if (!hidden) return;

            const field = btn.dataset.field;
            let set = (hidden.value || '').split(',').map(s => s.trim()).filter(Boolean);

            if (set.includes(field)) {
                set = set.filter(f => f !== field);
            } else {
                if (set.length >= 2) {
                    alert('Можно зафиксировать не больше двух параметров.');
                    return;
                }
                set.push(field);
            }
            hidden.value = set.join(',');
            refreshRow(row);
            row.dispatchEvent(new Event('recalc-request', { bubbles: true }));
        });

        body.querySelectorAll('.cost-row').forEach(refreshRow);
    }

    function checkRowWarnings(body) {
        let hasErrors = false;
        const warnings = [];
        const stageTotals = {};

        body.querySelectorAll('.cost-row').forEach(r => {
            const stageGUID = r.dataset.stageGuid;
            const stageCost = parseFloat(r.dataset.stageCost) || 0; 
            
            const rowSale = 
                num(r.querySelector('[name="cost_amount[]"]').value) +
                num(r.querySelector('[name="cost_equipment_sale[]"]').value);

            const rowExpenses = 
                num(r.querySelector('[name="cost_fot_plan[]"]').value) +
                num(r.querySelector('[name="cost_travel[]"]').value) +
                num(r.querySelector('[name="cost_equipment_cost[]"]').value) +
                num(r.querySelector('[name="cost_subcontract[]"]').value) +
                num(r.querySelector('[name="cost_commission[]"]').value) +
                num(r.querySelector('[name="cost_other[]"]').value);

            let stageName = 'Этап';
            let prev = r.previousElementSibling;
            while (prev) {
                if (prev.classList.contains('stage-header-row')) {
                    stageName = prev.cells[1].innerText;
                    break;
                }
                prev = prev.previousElementSibling;
            }

            if (!stageTotals[stageGUID]) {
                stageTotals[stageGUID] = { sale: 0, expenses: 0, stageCost: stageCost, stageName: stageName };
            }
            stageTotals[stageGUID].sale += rowSale;
            stageTotals[stageGUID].expenses += rowExpenses;
        });

        for (const guid in stageTotals) {
            const data = stageTotals[guid];
            
            if (data.sale > 0 && data.expenses > data.sale) {
                hasErrors = true;
                warnings.push(`${data.stageName}: себестоимость (${data.expenses.toFixed(2)} ₽) превышает стоимость работ (${data.sale.toFixed(2)} ₽). Этап убыточный.`);
            }

            if (data.stageCost > 0 && Math.abs(data.sale - data.stageCost) > 0.01) {
                hasErrors = true;
                warnings.push(`${data.stageName}: сумма «Стоимость работ» по строкам (${data.sale.toFixed(2)} ₽) не совпадает со стоимостью этапа (${data.stageCost.toFixed(2)} ₽).`);
            }
        }

        return { hasErrors, warnings };
    }

    return {
        num: num,
        setVal: setVal,
        recalcRow: recalcRow,
        initLocks: initLocks,
        checkRowWarnings: checkRowWarnings
    };
})();