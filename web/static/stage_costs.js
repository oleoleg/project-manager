document.addEventListener('DOMContentLoaded', function() {
    // Считаем итоги
    function recalcTotals() {
        let totalFot = 0;
        let totalCost = 0;
        
        document.querySelectorAll('.costs-table tbody tr').forEach(tr => {
            const cells = tr.querySelectorAll('td');
            if (cells.length > 5) {
                const costVal = parseMoneyText(cells[2].textContent);
                const fotVal = parseMoneyText(cells[3].textContent);
                totalCost += costVal;
                totalFot += fotVal;
            }
        });

        const fotEl = document.getElementById('total-fot');
        const costEl = document.getElementById('total-cost');
        if (fotEl) fotEl.textContent = formatMoney(totalFot);
        if (costEl) costEl.textContent = formatMoney(totalCost);
    }

    recalcTotals();
});
