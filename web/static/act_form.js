// ============ Вид акта: показ своего типа ============
(function () {
    const kind = document.getElementById('act-kind');
    const label = document.getElementById('kind-custom-label');
    if (!kind || !label) return;
    kind.addEventListener('change', () => {
        label.classList.toggle('is-visible', kind.value === 'custom');
    });
})();

// ============ Файл: Обзор + Открыть + автозаполнение наименования ============
(function () {
    const btn  = document.getElementById('btn-pick-file');
    const openBtn = document.getElementById('btn-open-file');
    const field = document.getElementById('act-file-path');
    const name  = document.getElementById('act-name');
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

        // имя файла без расширения
        const lastSlash = Math.max(path.lastIndexOf('\\'), path.lastIndexOf('/'));
        let base = lastSlash >= 0 ? path.slice(lastSlash + 1) : path;
        const dot = base.lastIndexOf('.');
        if (dot > 0) base = base.slice(0, dot);

            // Всегда перезаписываем имя файла при выборе нового файла
        if (name) {
            name.value = base;
        }

        // Пытаемся определить вид акта из имени файла
        if (window.detectActKind) {
            const detectedKind = window.detectActKind(base);
            if (detectedKind) {
                const kindSelect = document.getElementById('act-kind');
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
            window.location.href = 'pm-agent://open?path=' + encodeURIComponent(p);
        });
    }
})();

// ============ Разворачивание блока "Распределение по этапам" ============
    (function () {
    const btn    = document.getElementById('distribution-toggle');
    const body   = document.getElementById('distribution-body');
    const hidden = document.getElementById('distribution-expanded-hidden');
    if (!btn || !body || !hidden) return;

    btn.addEventListener('click', () => {
        const collapsed = body.classList.contains('is-collapsed');
        if (collapsed) {
            body.classList.remove('is-collapsed');
            hidden.value = 'on';
            btn.textContent = 'Свернуть';
        } else {
            body.classList.add('is-collapsed');
            hidden.value = '';
            btn.textContent = 'Развернуть';
        }
    });
})();

    // ============ Распределение по этапам ============
(function () {
    const body = document.getElementById('links-body');
    const addBtn = document.getElementById('link-add');
    const warn = document.getElementById('links-warning');
    if (!body || !addBtn) return;

    const templateRow = body.querySelector('.link-row');
    if (!templateRow) return;

    const amountInput = document.querySelector('input[name="amount"]');

    function parseNum(s) {
        if (!s) return 0;
        s = String(s).replace(/\s/g, '').replace(',', '.');
        const v = parseFloat(s);
        return isNaN(v) ? 0 : v;
    }

    function makeRow() {
        const row = templateRow.cloneNode(true);
        row.querySelector('select').value = '';
        row.querySelector('input[name="link_amount"]').value = '';
        return row;
    }

    addBtn.addEventListener('click', () => {
        const row = makeRow();
        body.appendChild(row);
        if (window.attachMoneyInput) {
            row.querySelectorAll('input.money-input').forEach(window.attachMoneyInput);
        }
        recalc();
    });

    // Клик по любой кнопке внутри строки
    body.addEventListener('click', (e) => {
        // Удаление строки
        if (e.target.classList.contains('link-remove')) {
            const rows = body.querySelectorAll('.link-row');
            if (rows.length > 1) {
                e.target.closest('tr').remove();
                recalc();
            } else {
                alert('Должна остаться хотя бы одна строка.');
            }
            return;
        }

        // Заполнение остатком конкретной строки
        if (e.target.classList.contains('link-fill')) {
            const row = e.target.closest('tr');
            const input = row.querySelector('input[name="link_amount"]');
            const actAmount = parseNum(amountInput ? amountInput.value : 0);

            let total = 0;
            body.querySelectorAll('.link-row').forEach((r) => {
                total += parseNum(r.querySelector('input[name="link_amount"]').value);
            });

            // Остаток с учётом текущего значения этой строки
            const rowCurrent = parseNum(input.value);
            const remaining = actAmount - (total - rowCurrent);
            if (remaining <= 0) return;

            input.value = remaining.toFixed(2);
            if (window.inputMoney) input.value = window.inputMoney(input.value);
            recalc();
            return;
        }
    });

    body.addEventListener('input', recalc);
    body.addEventListener('change', recalc);
    if (amountInput) amountInput.addEventListener('input', recalc);

    function recalc() {
        if (!warn) return;
        const actAmount = parseNum(amountInput ? amountInput.value : 0);

        let total = 0;
        body.querySelectorAll('.link-row').forEach((r) => {
            total += parseNum(r.querySelector('input[name="link_amount"]').value);
        });

        if (total > actAmount + 0.01) {
            warn.textContent = 'Сумма распределения (' + total.toFixed(2) +
                ' ₽) превышает сумму акта (' + actAmount.toFixed(2) + ' ₽).';
            warn.style.display = 'block';
        } else if (actAmount > 0 && total < actAmount - 0.01) {
            warn.textContent = 'Остаток нераспределённой суммы: ' +
                (actAmount - total).toFixed(2) + ' ₽';
            warn.style.display = 'block';
        } else {
            warn.style.display = 'none';
        }
    }

    recalc();
})();
