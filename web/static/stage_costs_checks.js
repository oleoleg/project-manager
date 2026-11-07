// stage_costs_checks.js
(function() {
    // Парсер чисел
    function num(s) {
        if (!s) return 0;
        s = String(s).replace(/[^\d.-]/g, '');
        let v = parseFloat(s);
        if (isNaN(v)) return 0;
        return Math.abs(v);
    }

    // Установка значения с форматированием
    function setVal(el, v) {
        v = Math.abs(v);
        
        if (el.classList.contains('days-input')) {
            const val = Math.round(v);
            el.value = val > 0 ? val : '';
            return;
        }

        if (v > 0) el.value = v.toFixed(2);
        else el.value = '';
        
        // Используем глобальный inputMoney
        if (window.inputMoney) {
            el.value = window.inputMoney(el.value);
        }
    }

     function recalcRow(row, changedName) {
        const fotEl  = row.querySelector('[name="cost_fot_plan[]"]');
        const daysEl = row.querySelector('[name="cost_labor_days[]"]');
        const rateEl = row.querySelector('[name="cost_rate_per_day[]"]');
        const lockEl = row.querySelector('input[name="cost_locked[]"]');
        
        const amountEl = row.querySelector('[name="cost_amount[]"]');
        const ratioEl  = row.querySelector('.ratio-display');
        
        // Пересчет коэффициента (Коэф = ФОТп / Стоимость)
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

        // Очищаем имя поля от [] 
        const field = changedName ? changedName.replace('[]', '') : '';

        // СЦЕНАРИЙ 1: Пользователь ввел ФОТп
        if (field === 'cost_fot_plan') {
            if (hasRate) {
                // Если зафиксирована ставка, меняем дни
                if (rate > 0) setVal(daysEl, fot / rate);
            } else {
                // Иначе меняем ставку (если есть дни)
                if (days > 0) setVal(rateEl, fot / days);
            }
        }
        // СЦЕНАРИЙ 2: Пользователь ввел Дни
        else if (field === 'cost_labor_days') {
            if (hasRate) {
                // Если зафиксирована ставка, меняем ФОТп
                setVal(fotEl, days * rate);
            } else {
                // Иначе меняем ставку (если есть ФОТп)
                if (days > 0) setVal(rateEl, fot / days);
            }
        }
        // СЦЕНАРИЙ 3: Пользователь ввел Ставку
        else if (field === 'cost_rate_per_day') {
            if (hasDays) {
                // Если зафиксированы дни, меняем ФОТп
                setVal(fotEl, days * rate);
            } else {
                // Иначе меняем дни (если есть ФОТп)
                if (rate > 0) setVal(daysEl, fot / rate);
            }
        }
        // СЦЕНАРИЙ 4: Программный пересчет (смена замочка, загрузка страницы)
        else {
            if (hasFOT && hasDays) {
                if (days > 0) setVal(rateEl, fot / days);
            } else if (hasFOT && hasRate) {
                if (rate > 0) setVal(daysEl, fot / rate);
            } else if (hasDays && hasRate) {
                setVal(fotEl, days * rate);
            } else if (hasFOT) {
                if (days > 0) setVal(rateEl, fot / days);
            } else if (hasDays) {
                if (rate > 0) setVal(fotEl, days * rate); // Если есть только дни, пересчитываем ФОТп при наличии ставки
            } else if (hasRate) {
                if (days > 0) setVal(fotEl, days * rate); // Если есть только ставка, пересчитываем ФОТп
            }
        }
    }

    function initLocks(tableBody) {
        if (!tableBody) return;

        function refreshRow(row) {
            const hidden = row.querySelector('input[name="cost_locked[]"]');
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

        // Обработчик клика по замочку
        tableBody.addEventListener('click', (e) => {
            const btn = e.target.closest('.lock-btn');
            if (!btn) return;
            e.preventDefault();

            const row = btn.closest('tr');
            const hidden = row.querySelector('input[name="cost_locked[]"]');
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
            
            // Вызываем пересчет строки после смены замочка
            recalcRow(row, '');
        });

        tableBody.querySelectorAll('tr').forEach(refreshRow);
    }

    // Экспортируем наружу
    window.StageCostsChecks = {
        num: num,
        recalcRow: recalcRow,
        initLocks: initLocks
    };
})();