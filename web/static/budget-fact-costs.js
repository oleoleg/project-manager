// web/static/budget-fact-costs.js
document.addEventListener('DOMContentLoaded', function() {

    const table = document.querySelector('#tab-costs .costs-table');
    if (!table) return;

    const getNum = (text) => {
        if (!text) return 0;
        // Убираем все кроме цифр, запятых и точек. Запятую меняем на точку.
        let s = text.replace(/[^\d.-]/g, '').replace(',', '.');
        return parseFloat(s) || 0;
    };

    let totals = {
        sale: 0,  payments: 0, fot: 0, days: 0, travel: 0, 
        eqSale: 0, eqCost: 0, sub: 0, comm: 0, over: 0, other: 0, overhead: 0
    };

    // 1. Собираем суммы по строкам затрат (td.ft-*)
    table.querySelectorAll('tbody tr.cost-row').forEach(row => {
        totals.sale += getNum(row.querySelector('.ft-sale')?.textContent);
        totals.fot += getNum(row.querySelector('.ft-fot')?.textContent);
        totals.days += getNum(row.querySelector('.ft-days')?.textContent);
        totals.travel += getNum(row.querySelector('.ft-travel')?.textContent);
        totals.eqSale += getNum(row.querySelector('.ft-eq-sale')?.textContent);
        totals.eqCost += getNum(row.querySelector('.ft-eq-cost')?.textContent);
        totals.sub += getNum(row.querySelector('.ft-sub')?.textContent);
        totals.comm += getNum(row.querySelector('.ft-comm')?.textContent);
        totals.over += getNum(row.querySelector('.ft-over')?.textContent);
        totals.other += getNum(row.querySelector('.ft-other')?.textContent);
    });

    // 2. Отдельно собираем платежи из заголовков этапов (tr.stage-header-row)
    table.querySelectorAll('tbody tr.stage-header-row').forEach(header => {
        const paymentsText = header.querySelector('span')?.textContent;
        if (paymentsText) {
            const match = paymentsText.match(/Платежи по договору:\s*([\d\s]+)/);
            if (match) {
                totals.payments += getNum(match[1]);
            }
        }
    });

    // Записываем в tfoot
    const formatMoney = (v) => new Intl.NumberFormat('ru-RU', { style: 'currency', currency: 'RUB', maximumFractionDigits: 0 }).format(v);
    
    document.getElementById('ft-foot-sale').textContent = formatMoney(totals.sale);
    document.getElementById('ft-foot-fot').textContent = formatMoney(totals.fot);
    document.getElementById('ft-foot-days').textContent = totals.days;
    document.getElementById('ft-foot-travel').textContent = formatMoney(totals.travel);
    document.getElementById('ft-foot-eq-sale').textContent = formatMoney(totals.eqSale);
    document.getElementById('ft-foot-eq-cost').textContent = formatMoney(totals.eqCost);
    document.getElementById('ft-foot-sub').textContent = formatMoney(totals.sub);
    document.getElementById('ft-foot-comm').textContent = formatMoney(totals.comm);
    document.getElementById('ft-foot-over').textContent = formatMoney(totals.over);
    document.getElementById('ft-foot-other').textContent = formatMoney(totals.other);

    // Карточки
    const totalCost = totals.fot + totals.travel + totals.eqCost + totals.sub + totals.comm + totals.over + totals.other
    
    document.getElementById('ft-total-sale-and-eq').textContent = '(Всего по Договору: ' + formatMoney(totals.sale + totals.eqSale) + ' )'; 
    document.getElementById('ft-total-payments').textContent = formatMoney(totals.payments)
    document.getElementById('ft-total-fot').textContent = formatMoney(totals.fot);
    document.getElementById('ft-total-eq-cost').textContent = formatMoney(totals.eqCost);
    document.getElementById('ft-total-cost').textContent = formatMoney(totalCost);
});