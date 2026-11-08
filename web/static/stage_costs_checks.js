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
 
    function recalcRow(row) {
        // Ищем поля, поддерживая оба варианта имен (с [] и без)
        const fotEl  = row.querySelector('[name="cost_fot_plan[]"], [name="cost_fot_plan"]');
        const daysEl = row.querySelector('[name="cost_labor_days[]"], [name="cost_labor_days"]');
        const rateEl = row.querySelector('[name="cost_rate_per_day[]"], [name="cost_rate_per_day"]');
        const calcFieldInput = row.querySelector('[name="cost_calc_field[]"], [name="cost_calc_field"]');
        const amountEl = row.querySelector('[name="cost_amount[]"], [name="cost_amount"]');
        const ratioEl  = row.querySelector('.ratio-display');
        
        if (!calcFieldInput || !fotEl || !daysEl || !rateEl) return; 
        
        const calcField = calcFieldInput.value;
        let fot = num(fotEl.value);
        let days = Math.round(num(daysEl.value));
        let rate = num(rateEl.value);

        const MAX_DAYS = 999;
        const MAX_RATE = 200000;

        // Пересчитываем то поле, которое помечено как "auto"
        if (calcField === 'fot_plan') {
            fot = days * rate;
            setVal(fotEl, fot);
        } else if (calcField === 'labor_days') {
            if (rate > 0) {
                days = Math.round(fot / rate);
                if (days > MAX_DAYS) days = MAX_DAYS;
                setVal(daysEl, days);
            }
        } else if (calcField === 'rate_per_day') {
            if (days > 0) {
                rate = fot / days;
                if (rate > MAX_RATE) rate = MAX_RATE;
                setVal(rateEl, rate);
            }
        }

        // Обновляем коэффициент
        if (ratioEl && amountEl) {
            const amount = num(amountEl.value);
            const currentFot = num(fotEl.value);
            ratioEl.value = amount > 0 ? (currentFot / amount).toFixed(2) : '';
        }
    }


    // Проверка одной строки на корректность
    // Возвращает объект { hasError: boolean, message: string }
    function checkRow(row, stageCost, overheadRate) {
        // Если в строке включен режим "на удаление" - не проверяем
        if (row.classList.contains('is-deleted')) {
            return { hasError: false, message: '' };
        }

        const amountEl = row.querySelector('[name="cost_amount[]"]');
        const fotEl = row.querySelector('[name="cost_fot_plan[]"]');
        const travelEl = row.querySelector('[name="cost_travel[]"]');
        const eqCostEl = row.querySelector('[name="cost_equipment_cost[]"]');
        const subEl = row.querySelector('[name="cost_subcontract[]"]');
        const commEl = row.querySelector('[name="cost_commission[]"]');
        const otherEl = row.querySelector('[name="cost_other[]"]');
        const eqSaleEl = row.querySelector('[name="cost_equipment_sale[]"]');

        if (!amountEl) return { hasError: false, message: '' };

        const amount = num(amountEl.value);
        const eqSale = num(eqSaleEl ? eqSaleEl.value : 0);
        const totalSale = amount + eqSale;

        const fot = num(fotEl ? fotEl.value : 0);
        const overheadAmount = fot * (overheadRate || 1.0);

        const totalExpenses = 
            fot +
            overheadAmount + 
            num(travelEl ? travelEl.value : 0) +
            num(eqCostEl ? eqCostEl.value : 0) +
            num(subEl ? subEl.value : 0) +
            num(commEl ? commEl.value : 0) +
            num(otherEl ? otherEl.value : 0);

        let hasError = false;
        let message = '';

        // 1. Проверка убыточности (Затраты > Стоимость)
        if (totalSale > 0 && totalExpenses > totalSale) {
            hasError = true;
            message = `Себестоимость (${formatMoney(totalExpenses)}) превышает стоимость (${formatMoney(totalSale)}). Строка убыточная.`;
        }
        
        // 2. Проверка, что сумма строки не превышает стоимость этапа (если это единственная строка, или в целом)
        // (Для простоты мы проверяем, не превышает ли стоимость продажи стоимость этапа)
        // if (stageCost > 0 && totalSale > stageCost + 0.01) {
        //     hasError = true;
        //     message = `Стоимость работ и оборудования (${formatMoney(totalSale)}) больше стоимости этапа (${formatMoney(stageCost)}).`;
        // }

        return { hasError, message };
    }

        // Проверка итогов по каждому этапу
    function checkStageTotals(tableBody) {
        let hasErrors = false;
        let errorMessages = [];

        // Группируем строки по этапам
        const stageRows = {};
        tableBody.querySelectorAll('tr.cost-row').forEach(row => {
            if (row.classList.contains('is-deleted')) return;
            const stageGUID = row.dataset.stageGuid;
            if (!stageGUID) return;
            
            if (!stageRows[stageGUID]) {
                stageRows[stageGUID] = [];
            }
            stageRows[stageGUID].push(row);
        });

        // Для каждого этапа считаем сумму
        for (const stageGUID in stageRows) {
            const rows = stageRows[stageGUID];
            let totalSale = 0;
            
            rows.forEach(row => {
                const amountEl = row.querySelector('[name="cost_amount[]"]');
                const eqSaleEl = row.querySelector('[name="cost_equipment_sale[]"]');
                totalSale += num(amountEl ? amountEl.value : 0);
                totalSale += num(eqSaleEl ? eqSaleEl.value : 0);
            });

            // Находим заголовок этапа, чтобы узнать его стоимость
            let headerRow = rows[0].previousElementSibling;
            while (headerRow) {
                if (headerRow.classList.contains('stage-header-row')) {
                    break;
                }
                headerRow = headerRow.previousElementSibling;
            }

            if (headerRow) {
                // Извлекаем стоимость этапа напрямую из data-атрибута
                let stageCost = 0;
                if (headerRow.dataset.stageCost) {
                    stageCost = parseFloat(headerRow.dataset.stageCost) || 0;
                }

                // Если стоимость этапа больше 0 и есть расхождение
                if (stageCost > 0 && Math.abs(stageCost - totalSale) > 0.01) {
                    hasErrors = true;
                    
                    // Получаем номер этапа (первая ячейка)
                    let stageNumber = '?';
                    const numEl = headerRow.querySelector('td:first-child');
                    if (numEl) {
                        stageNumber = numEl.textContent.trim();
                    }

                    // Получаем название этапа (вторая ячейка)
                    let stageName = 'Этап';
                    const nameEl = headerRow.querySelector('td:nth-child(2)');
                    if (nameEl) {
                        stageName = nameEl.childNodes[0].textContent.trim();
                    }
                    
                    errorMessages.push(`Этап ${stageNumber} (${stageName}): Сумма продаж по строкам (${formatMoney(totalSale)}) не совпадает со стоимостью этапа (${formatMoney(stageCost)}).`);
                    
                    // Подсвечиваем заголовок этапа
                    headerRow.classList.add('row-error');
                } else {
                    headerRow.classList.remove('row-error');
                }
            }
        }

        return { hasErrors, errorMessages };
    }

    // Экспортируем наружу
    window.StageCostsChecks = {
        num: num,
        recalcRow: recalcRow,
        checkRow: checkRow,
        checkStageTotals: checkStageTotals
    };
})();