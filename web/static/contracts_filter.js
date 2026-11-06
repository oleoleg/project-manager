document.addEventListener("DOMContentLoaded", function() {
    const btn = document.getElementById('status-dropdown-btn');
    const menu = document.getElementById('status-dropdown-menu');
    const checkboxes = document.querySelectorAll('.status-checkbox');
    const hiddenInput = document.getElementById('status-hidden-input');
    const filterForm = document.querySelector('.filter-form');
    
    if (!btn || !menu) return;

    let hasChanges = false; // Флаг: изменились ли чекбоксы

    // 1. Открытие/закрытие меню
    btn.addEventListener('click', (e) => {
        e.preventDefault();
        menu.style.display = menu.style.display === 'none' ? 'block' : 'none';
    });

    // 2. Закрытие меню при клике вне его (и автоотправка, если были изменения)
    document.addEventListener('click', (e) => {
        if (!menu.contains(e.target) && !btn.contains(e.target)) {
            if (menu.style.display === 'block') {
                menu.style.display = 'none';
                // Если чекбоксы менялись, отправляем форму
                if (hasChanges) {
                    filterForm.submit();
                }
            }
        }
    });

    // 3. Восстановление выбранных чекбоксов из скрытого поля при загрузке
    const currentStatus = hiddenInput.value;
    if (currentStatus) {
        const values = currentStatus.split(',');
        checkboxes.forEach(cb => {
            if (values.includes(cb.value)) {
                cb.checked = true;
            }
        });
    }

    // 4. Обновление кнопки и скрытого поля
    function updateSelection() {
        const selected = [];
        checkboxes.forEach(cb => {
            if (cb.checked) selected.push(cb.value);
        });

        hiddenInput.value = selected.join(',');

        if (selected.length === 0) {
            btn.textContent = '— все статусы — ▼';
        } else {
            btn.textContent = `Выбрано: ${selected.length} ▼`;
        }
    }

    // 5. Обработчик кликов по чекбоксам (без отправки формы)
    checkboxes.forEach(cb => {
        cb.addEventListener('change', () => {
            updateSelection();
            hasChanges = true; // Запоминаем, что были изменения
        });
    });

    // 6. Автоотправка при смене контрагента
    if (filterForm) {
        const cpSelect = filterForm.querySelector('select[name="counterparty"]');
        if (cpSelect) {
            cpSelect.addEventListener('change', () => {
                filterForm.submit();
            });
        }
    }
    
    // Инициализация текста на кнопке
    updateSelection();
});