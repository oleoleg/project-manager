document.addEventListener("DOMContentLoaded", function() {
    const btnOpen = document.getElementById('btn-append-plan');
    const modal = document.getElementById('append-modal');
    const btnClose = document.getElementById('append-modal-close');
    const btnCancel = document.getElementById('append-modal-cancel');
    const btnSave = document.getElementById('append-modal-save');
    const listBox = document.getElementById('append-modal-list');

    if (!btnOpen || !modal) return;

    // Получаем ID договора из URL
    const contractID = window.location.pathname.split('/')[2];

    // 1. Открытие модалки
    btnOpen.addEventListener('click', () => {
        modal.style.display = 'flex'; // Показываем
        listBox.innerHTML = '<p class="muted">Загрузка…</p>';
        btnSave.disabled = true;

        fetch(`/contracts/${contractID}/budget/fact/append-preview`, {
            method: 'POST'
        })
        .then(r => r.json())
        .then(data => {
            renderLines(data);
            btnSave.disabled = false;
        })
        .catch(err => {
            console.error(err);
            listBox.innerHTML = '<p class="error">Ошибка загрузки данных.</p>';
        });
    });

    // 2. Закрытие модалки
     const closeModal = () => { modal.style.display = 'none'; }; // Скрываем
    btnClose.addEventListener('click', closeModal);
    btnCancel.addEventListener('click', closeModal);
    modal.addEventListener('click', (e) => {
        if (e.target === modal) closeModal();
    });

    // 3. Отрисовка списка строк
    function renderLines(lines) {
        if (!lines || lines.length === 0) {
            listBox.innerHTML = '<p class="muted">Все строки из плана уже добавлены в факт.</p>';
            btnSave.disabled = true;
            return;
        }

        let html = '<table class="data-table"><thead><tr>' +
            '<th style="width: 40px; text-align: center;"><input type="checkbox" id="append-select-all" checked></th>' +
            '<th>№ этапа</th><th>Тип операции</th><th>Наименование</th><th class="num">Сумма (План)</th>' +
            '</tr></thead><tbody>';

        for (const l of lines) {
            let amountStr = '';
            // Используем Income и Expense с большой буквы, как приходит из Go
            if (l.Income > 0) amountStr = `Приход: ${formatMoney(l.Income)}`;
            if (l.Expense > 0) amountStr = `Расход: ${formatMoney(l.Expense)}`;

            html += `<tr>
                <td style="text-align: center;"><input type="checkbox" class="append-cb" value="${l.ID}" checked></td>
                <td style="text-align: center;">${l.StageNumber || '—'}</td>
                <td>${l.OperationType}</td>
                <td>${l.Name}</td>
                <td class="num">${amountStr}</td>
            </tr>`;
        }
        html += '</tbody></table>';
        listBox.innerHTML = html;

        // Обработчик для "Выбрать все"
        const selectAll = document.getElementById('append-select-all');
        if (selectAll) {
            selectAll.addEventListener('change', (e) => {
                document.querySelectorAll('.append-cb').forEach(cb => cb.checked = e.target.checked);
            });
        }
    }

    // 4. Сохранение
    btnSave.addEventListener('click', () => {
        const selectedIds = [];
        document.querySelectorAll('.append-cb:checked').forEach(cb => {
            selectedIds.push(cb.value);
        });

        if (selectedIds.length === 0) {
            alert('Выберите хотя бы одну строку.');
            return;
        }

        btnSave.disabled = true;
        btnSave.textContent = 'Добавление...';

        fetch(`/contracts/${contractID}/budget/fact/append-selected`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ids: selectedIds })
        })
        .then(r => {
            if (r.ok) {
                window.location.reload();
            } else {
                throw new Error('Ошибка сервера');
            }
        })
        .catch(err => {
            console.error(err);
            alert('Ошибка при сохранении');
            btnSave.disabled = false;
            btnSave.textContent = 'Добавить выбранные';
        });
    });
});