document.addEventListener("DOMContentLoaded", function() {
    const btnSync = document.getElementById('btn-sync-reports');
    const modal = document.getElementById('sync-modal');
    const btnClose = document.getElementById('sync-modal-close');
    const btnOk = document.getElementById('sync-modal-ok');
    const btnCancel = document.getElementById('sync-modal-cancel');
    
    const progressBox = document.getElementById('sync-progress');
    const resultBox = document.getElementById('sync-result');
    const errorBox = document.getElementById('sync-error-box');
    const errorsList = document.getElementById('sync-errors-list');

    if (!btnSync || !modal) return;

    btnSync.addEventListener('click', function() {
        const contractId = this.getAttribute('data-contract-id');
        
        // 1. Открываем модалку
        modal.style.display = 'flex';
        progressBox.style.display = 'block';
        resultBox.style.display = 'none';
        errorBox.style.display = 'none';
        errorsList.style.display = 'none';
        btnClose.disabled = true;
        btnOk.style.display = 'none';
        btnCancel.style.display = 'none';

        // 2. Подготовка данных формы
        const formData = new FormData();
        if (contractId && contractId !== '0') {
            formData.append('contract_id', contractId);
        }

        // 3. Выполняем запрос
        fetch('/reports/sync', {
            method: 'POST',
            body: formData
        })
        .then(async r => {
            const data = await r.json();
            if (!r.ok) throw new Error(data.error || 'Ошибка сервера');
            return data;
        })
        .then(data => {
            // 4. Успех — показываем статистику
            progressBox.style.display = 'none';
            resultBox.style.display = 'block';
            btnClose.disabled = false;
            btnCancel.style.display = 'inline-block'; // Показываем кнопку "Обновить страницу"

            document.getElementById('res-departments').textContent = data.departments;
            document.getElementById('res-employees').textContent = data.employees;
            document.getElementById('res-worktypes').textContent = data.worktypes;
            document.getElementById('res-projects').textContent = data.projects;
            document.getElementById('res-reports').textContent = data.reports;
            document.getElementById('res-skipped').textContent = data.skipped;

            if (data.errors && data.errors.length > 0) {
                errorsList.style.display = 'block';
                errorsList.innerHTML = '<strong>Пропущенные записи:</strong><br>' + data.errors.join('<br>');
            }
        })
        .catch(err => {
            // 5. Ошибка — показываем текст ошибки
            progressBox.style.display = 'none';
            resultBox.style.display = 'block';
            errorBox.style.display = 'block';
            errorBox.textContent = 'Ошибка: ' + err.message;
            btnClose.disabled = false;
            btnOk.style.display = 'inline-block';
        });
    });

    // Закрытие модалки
    const closeModal = () => { modal.style.display = 'none'; };
    
    btnClose.addEventListener('click', closeModal);
    btnOk.addEventListener('click', closeModal);
    
    // Кнопка "Обновить страницу"
    btnCancel.addEventListener('click', () => {
        window.location.reload();
    });
});