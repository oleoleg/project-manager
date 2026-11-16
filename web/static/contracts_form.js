document.addEventListener('DOMContentLoaded', function() {
    const statusSelect = document.querySelector('select[name="status_code"]');
    const contractNumberInput = document.querySelector('input[name="contract_number"]');
    const workNameInput = document.querySelector('input[name="work_name"]');

    if (!statusSelect || !contractNumberInput || !workNameInput) return;

    // Функция для переключения состояния поля "№ договора"
    function toggleContractNumber() {
        const status = parseInt(statusSelect.value, 10);
        
        // Статус 0 - Заключение договора
        if (status === 0) {
            // Если поле пустое, ставим заглушку
            if (contractNumberInput.value.trim() === '') {
                contractNumberInput.value = '<не присвоен>';
            }
        } else {
            // Если статус другой, а там всё еще заглушка — очищаем поле
            if (contractNumberInput.value.trim() === '<не присвоен>') {
                contractNumberInput.value = '';
            }
        }
    }

    // Вешаем обработчик на изменение статуса
    statusSelect.addEventListener('change', toggleContractNumber);

    // Вызываем один раз при загрузке страницы (на случай, если форма открылась 
    // со статусом 0, но поле пустое)
    toggleContractNumber();
});



// Выбор папки проекта через pm-agent://pick-dir
(function () {
    const btn = document.getElementById('btn-pick-dir');
    const field = document.getElementById('contract-project-url');
    if (!btn || !field) return;

    let pollTimer = null;
    let pollStartedAt = 0;

    btn.addEventListener('click', (e) => {
        e.preventDefault(); // Предотвращаем стандартное поведение

        // Создаем невидимую ссылку и кликаем по ней (браузеры это любят больше)
        let currentPath = field.value.trim();
        let url = 'pm-agent://pick-dir';
        if (currentPath) {
            url += '?path=' + encodeURIComponent(currentPath);
        }

        const a = document.createElement('a');
        a.href = url;
        a.style.display = 'none';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);

        if (pollTimer) clearInterval(pollTimer);
        pollStartedAt = Date.now();
        
        // Опрашиваем сервер каждые 500 мс
        pollTimer = setInterval(checkPickedPath, 1000);
    });

    function checkPickedPath() {
        // Ждем максимум 2 минуты
        if (Date.now() - pollStartedAt > 120000) {
            clearInterval(pollTimer);
            pollTimer = null;
            return;
        }

        fetch('/api/picked-path?t=' + Date.now())
            .then(r => r.json())
            .then(data => {
                // Если пришел путь и он новее, чем начало клика
                if (data && data.path && data.mtime > pollStartedAt) {
                    field.value = data.path;
                    clearInterval(pollTimer);
                    pollTimer = null;
                }
            })
            .catch(() => {});
    }
})();
