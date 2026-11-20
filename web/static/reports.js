document.addEventListener('DOMContentLoaded', function() {
    const form = document.querySelector('.filter-form');
    if (!form) return;

    let timer = null;

    // 1. Для input (текстовые поля и datalist) - отправляем с задержкой 500мс
    const inputs = form.querySelectorAll('input[type="text"], input[list], input[type="date"]');
    inputs.forEach(input => {
        input.addEventListener('input', function() {
            clearTimeout(timer);
            timer = setTimeout(() => {
                form.submit();
            }, 900);
        });
        
        // При выборе из datalist (кликом или Enter)
        input.addEventListener('change', function() {
            form.submit();
        });
    });

    // 2. Для select (выпадающие списки) - отправляем сразу при выборе
    const selects = form.querySelectorAll('select');
    selects.forEach(select => {
        select.addEventListener('change', function() {
            form.submit();
        });
    });
});