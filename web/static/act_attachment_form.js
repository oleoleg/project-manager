// Вид приложения: показ своего типа
(function () {
    const kind = document.getElementById('att-kind');
    const label = document.getElementById('att-kind-custom-label');
    if (!kind || !label) return;
    kind.addEventListener('change', () => {
        label.classList.toggle('is-visible', kind.value === 'custom');
    });
})();

// Файл: Обзор + Открыть + автозаполнение наименования
(function () {
    const btn = document.getElementById('btn-pick-file');
    const openBtn = document.getElementById('btn-open-file');
    const field = document.getElementById('att-file-path');
    const name = document.getElementById('att-name');
    if (!field) return;

    let pollTimer = null;
    let pollStartedAt = 0;

    if (btn) {
        btn.addEventListener('click', () => {
            // Берем текущий путь из поля
            let currentPath = field.value.trim();
            
            // Формируем URL протокола с параметром path
            let url = 'pm-agent://pick-file?filter=pdf';
            if (currentPath) {
                url += '?path=' + encodeURIComponent(currentPath);
            }

            window.location.href = url;
            if (pollTimer) clearInterval(pollTimer);
            pollStartedAt = Date.now();
            pollTimer = setInterval(checkPickedPath, 800);
        });
    }

    function checkPickedPath() {
        if (Date.now() - pollStartedAt > 60000) {
            clearInterval(pollTimer);
            pollTimer = null;
            return;
        }
        fetch('/api/picked-path')
            .then(r => r.json())
            .then(data => {
                if (data && data.path && data.mtime > pollStartedAt) {
                    field.value = data.path;
                    autoFillName(data.path);
                    if (openBtn) openBtn.disabled = false;
                    clearInterval(pollTimer);
                    pollTimer = null;
                }
            })
            .catch(() => {});
    }

    function autoFillName(path) {
        if (!name || !path) return;

        const lastSlash = Math.max(path.lastIndexOf('\\'), path.lastIndexOf('/'));
        let base = lastSlash >= 0 ? path.slice(lastSlash + 1) : path;
        const dot = base.lastIndexOf('.');
        if (dot > 0) base = base.slice(0, dot);

        if (name) name.value = base;

        if (window.detectActKind) {
            const detectedKind = window.detectActKind(base);
            if (detectedKind) {
                const kindSelect = document.getElementById('att-kind');
                if (kindSelect) {
                    kindSelect.value = detectedKind;
                    kindSelect.dispatchEvent(new Event('change'));
                }
            }
        }
    }

    if (openBtn) {
        openBtn.addEventListener('click', () => {
            const p = field.value.trim();
            if (!p) return;
            window.location.href = 'pm-agent://open?path=' + encodeURIComponent(p); // ИСПРАВЛЕНО
        });
    }
})();
