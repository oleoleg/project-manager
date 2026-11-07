document.addEventListener('DOMContentLoaded', function() {
    // 1. Функция пересчета итогов
    function recalcTotals() {
        let totals = {
            sale: 0,
            fot: 0,
            days: 0,
            travel: 0,
            eqSale: 0,
            eqCost: 0,
            sub: 0,
            comm: 0,
            other: 0
        };
        
        // Берем процент накладных из data-атрибута
        const calcData = document.getElementById('calc-data');
        const overheadRate = calcData ? parseFloat(calcData.dataset.overhead) : 1.0;

        document.querySelectorAll('.costs-table tbody tr.cost-row').forEach(tr => {
            const getVal = (name) => {
                const el = tr.querySelector(`input[name="${name}[]"]`);
                return el ? parseMoneyText(el.value) : 0;
            };

            totals.sale   += getVal('cost_amount');
            totals.fot    += getVal('cost_fot_plan');
            totals.days   += getVal('cost_labor_days');
            totals.travel += getVal('cost_travel');
            totals.eqSale += getVal('cost_equipment_sale');
            totals.eqCost += getVal('cost_equipment_cost');
            totals.sub    += getVal('cost_subcontract');
            totals.comm   += getVal('cost_commission');
            totals.other  += getVal('cost_other');
        });

        // Считаем накладные и итоговую себестоимость
        const overheadAmount = totals.fot * overheadRate; // Если у вас коэф 1.2, то умножаем ФОТ на 1.2
        const totalCost = totals.fot + overheadAmount + totals.travel + totals.eqCost + totals.sub + totals.comm + totals.other;

        // Обновляем карточки
        const elFot = document.getElementById('total-fot');
        const elSale = document.getElementById('total-sale');
        const elCost = document.getElementById('total-cost');
        
        if (elFot) elFot.textContent = formatMoney(totals.fot);
        if (elSale) elSale.textContent = formatMoney(totals.sale);
        if (elCost) elCost.textContent = formatMoney(totalCost);

        // Обновляем итоговую строку (tfoot)
        const setFoot = (id, val, isMoney = true) => {
            const el = document.getElementById(id);
            if (el) el.textContent = isMoney ? formatMoney(val) : Math.round(val);
        };

        setFoot('foot-total-sale', totals.sale);
        setFoot('foot-total-fot', totals.fot);
        setFoot('foot-total-days', totals.days, false);
        setFoot('foot-total-travel', totals.travel);
        setFoot('foot-total-eq-sale', totals.eqSale);
        setFoot('foot-total-eq-cost', totals.eqCost);
        setFoot('foot-total-sub', totals.sub);
        setFoot('foot-total-comm', totals.comm);
        setFoot('foot-total-other', totals.other);
    }

    // Первоначальный расчет
    recalcTotals();

    // 2. Функция сортировки (заглушка, так как мы сгруппировали строки)
    window.sortCostsTable = function(field, btnElement) {
        alert('Сортировка временно недоступна из-за группировки по этапам. Пожалуйста, используйте фильтры на странице этапов.');
    };

    // 3. Инициализация логики пересчета (ФОТп, т/з, ставка)
    const tableBody = document.querySelector('.costs-table tbody');
    if (tableBody && window.StageCostsChecks) {
        StageCostsChecks.initLocks(tableBody);

        tableBody.addEventListener('input', (e) => {
            const row = e.target.closest('tr.cost-row');
            if (!row) return;
            
            setTimeout(() => {
                StageCostsChecks.recalcRow(row, e.target.name);
                recalcTotals(); // Пересчитываем итоги при каждом изменении
            }, 0);
        });
        
        // Первоначальный пересчет для всех строк
        tableBody.querySelectorAll('tr.cost-row').forEach(row => {
            StageCostsChecks.recalcRow(row, '');
        });
        
        // И пересчитываем итоги
        recalcTotals();
    }
});