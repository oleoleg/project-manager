document.addEventListener("DOMContentLoaded", function() {
    // Универсальная функция для настройки выпадающего списка с чекбоксами
    function setupDropdown(btnId, menuId, checkboxClass, hiddenInputId, clearBtnId) {
        const btn = document.getElementById(btnId);
        const menu = document.getElementById(menuId);
        const checkboxes = document.querySelectorAll('.' + checkboxClass);
        const hiddenInput = document.getElementById(hiddenInputId);
        const clearBtn = document.getElementById(clearBtnId); // НОВОЕ
        const filterForm = document.querySelector('.filter-form');
        
        if (!btn || !menu || !hiddenInput) return;

        let hasChanges = false;

        // Открытие/закрытие
        btn.addEventListener('click', (e) => {
            e.preventDefault();
            // Закрываем другие меню
            document.querySelectorAll('.custom-select-wrapper > div').forEach(el => {
                if (el !== menu) el.style.display = 'none';
            });
            menu.style.display = menu.style.display === 'none' ? 'block' : 'none';
        });

        // Клик вне меню
        document.addEventListener('click', (e) => {
            if (!menu.contains(e.target) && !btn.contains(e.target)) {
                if (menu.style.display === 'block') {
                    menu.style.display = 'none';
                    if (hasChanges && filterForm) {
                        filterForm.submit();
                    }
                }
            }
        });

        // Восстановление из скрытого поля
        const currentValue = hiddenInput.value;
        if (currentValue) {
            const values = currentValue.split(',');
            checkboxes.forEach(cb => {
                if (values.includes(cb.value)) {
                    cb.checked = true;
                }
            });
        }

         // Обновление текста кнопки и скрытого поля
        function updateSelection() {
            const selected = [];
            checkboxes.forEach(cb => {
                if (cb.checked) selected.push(cb.value);
            });

            hiddenInput.value = selected.join(',');

            const defaultText = btnId === 'status-dropdown-btn' ? '— все статусы — ▼' : '— все типы — ▼';
            const prefix = btnId === 'status-dropdown-btn' ? 'Выбрано статусов: ' : 'Выбрано типов: ';
            
            if (selected.length === 0) {
                btn.textContent = defaultText;
            } else {
                btn.textContent = `${prefix}${selected.length} ▼`;
            }
        }

        // Обработчик чекбоксов
        checkboxes.forEach(cb => {
            cb.addEventListener('change', () => {
                updateSelection();
                hasChanges = true;
            });
        });

       // Обработчик кнопки "Очистить"
        if (clearBtn) {
            clearBtn.addEventListener('click', (e) => {
                e.preventDefault();
                checkboxes.forEach(cb => cb.checked = false);
                updateSelection();
                hasChanges = true;
                // Автоматически закрываем меню и отправляем форму
                menu.style.display = 'none';
                if (filterForm) filterForm.submit();
            });
        }

        // Инициализация текста
        updateSelection();
    }

    // Настраиваем оба фильтра
    setupDropdown('status-dropdown-btn', 'status-dropdown-menu', 'status-checkbox', 'status-hidden-input', 'clear-status-btn');
    setupDropdown('type-dropdown-btn', 'type-dropdown-menu', 'type-checkbox', 'type-hidden-input', 'clear-type-btn');

    // Автоотправка при смене контрагента
    const filterForm = document.querySelector('.filter-form');
    if (filterForm) {
        const cpSelect = filterForm.querySelector('select[name="counterparty"]');
        if (cpSelect) {
            cpSelect.addEventListener('change', () => {
                filterForm.submit();
            });
        }
    }
});