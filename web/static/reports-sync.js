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

    if (!progressBox || !resultBox || !errorBox || !errorsList) {
        console.error("Элементы модалки не найдены в DOM!");
        return;
    }

    if (!btnSync) {
        console.error("Кнопка btn-sync-reports не найдена!");
        return;
    }
    if (!modal) {
        console.error("Модалка sync-modal не найдена!");
        return;
    }

    btnSync.addEventListener('click', function(e) {
        e.preventDefault();
        console.log("Старт синхронизации...");

        let contractId = this.getAttribute('data-contract-id');
        if (contractId === '0') contractId = '';

        // Показываем модалку
        modal.style.display = 'flex';
        progressBox.style.display = 'block';
        resultBox.style.display = 'none';
        errorBox.style.display = 'none';
        errorsList.style.display = 'none';
        btnClose.disabled = true;
        btnOk.style.display = 'none';
        btnCancel.style.display = 'none';

        const formData = new FormData();
        if (contractId) formData.append('contract_id', contractId);

        fetch('/reports/sync', {
            method: 'POST',
            body: formData
        })
        .then(async response => {
            console.log("Ответ сервера:", response.status);
            const isJson = response.headers.get('content-type')?.includes('application/json');
            const data = isJson ? await response.json() : { error: 'Сервер вернул не JSON (код ' + response.status + ')' };
            
            if (!response.ok) {
                throw new Error(data.error || 'Ошибка на сервере');
            }
            return data;
        })
        .then(data => {
            console.log("Данные синхронизации:", data);
            progressBox.style.display = 'none';
            resultBox.style.display = 'block';
            btnClose.disabled = false;
            btnCancel.style.display = 'inline-block';

            document.getElementById('res-departments').textContent = data.departments || 0;
            document.getElementById('res-employees').textContent = data.employees || 0;
            document.getElementById('res-worktypes').textContent = data.worktypes || 0;
            document.getElementById('res-projects').textContent = data.projects || 0;
            document.getElementById('res-reports').textContent = data.reports || 0;
            document.getElementById('res-skipped').textContent = data.skipped || 0;

            if (data.errors && data.errors.length > 0) {
                errorsList.style.display = 'block';
                errorsList.innerHTML = '<strong>Пропущенные записи:</strong><br>' + data.errors.join('<br>');
            }
 
        })
        .catch(err => {
            console.error("Ошибка:", err);
            progressBox.style.display = 'none';
            resultBox.style.display = 'block';
            errorBox.style.display = 'block';
            errorBox.textContent = err.message;
            btnClose.disabled = false;
            btnOk.style.display = 'inline-block';
        });
    });

    const closeModal = () => { modal.style.display = 'none'; };
    btnClose.addEventListener('click', closeModal);
    btnOk.addEventListener('click', closeModal);
    btnCancel.addEventListener('click', () => window.location.reload());
});