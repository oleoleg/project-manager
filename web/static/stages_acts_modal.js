// ============ Модальное окно «Акты этапа» ============
(function () {
    const modal = document.getElementById('acts-modal');
    const closeBtn = document.getElementById('acts-modal-close');
    const listBox = document.getElementById('acts-modal-list');
    const nameBox = document.getElementById('acts-modal-stage-name');
    const addBtn = document.getElementById('acts-modal-add');

    if (!modal) return;

    // Автовосстановление модалки после перезагрузки
    (function () {
        let data;
        try {
            const raw = sessionStorage.getItem('reopen_modal');
            if (!raw) return;
            sessionStorage.removeItem('reopen_modal');
            data = JSON.parse(raw);
        } catch (e) { return; }
        if (!data || !data.contractID || !data.stageGUID) return;

        // Клик по «Акты» для восстановления
        nameBox.textContent = data.stageName || '';
        addBtn.href = `/contracts/${data.contractID}/stages/${data.stageGUID}/acts/new`;

        listBox.innerHTML = '<p class="muted">Загрузка…</p>';
        modal.hidden = false;

        fetch(`/contracts/${data.contractID}/stages/${data.stageGUID}/acts.json`)
            .then(r => r.json())
            .then(resp => {
                renderSummary(resp.header, data.contractID, data.stageGUID);
                renderActs(resp.acts || []);
            })
            .catch(() => {
                listBox.innerHTML = '<p class="error">Не удалось загрузить акты.</p>';
            });
    })();

    document.body.addEventListener('click', (e) => {
        const btn = e.target.closest('.acts-btn');
        if (!btn) return;
        e.preventDefault();

        const contractID = btn.dataset.contract;
        const stageGUID = btn.dataset.stage;
        const stageName = btn.dataset.stageName || '';

        nameBox.textContent = stageName;
        addBtn.href = `/contracts/${contractID}/stages/${stageGUID}/acts/new`;

        listBox.innerHTML = '<p class="muted">Загрузка…</p>';
        modal.hidden = false;

        fetch(`/contracts/${contractID}/stages/${stageGUID}/acts.json`)
            .then(r => r.json())
            .then(data => {
                renderSummary(data.header, contractID, stageGUID);
                renderActs(data.acts || []);
            })
            .catch(() => {
                listBox.innerHTML = '<p class="error">Не удалось загрузить акты.</p>';
            });
    });

    closeBtn.addEventListener('click', () => { modal.hidden = true; });
    modal.addEventListener('click', (e) => {
        if (e.target === modal) modal.hidden = true;
    });

    function renderActs(acts) {
        try {
            if (!acts || acts.length === 0) {
                listBox.innerHTML = '<p class="muted">Актов пока нет.</p>';
                return;
            }

            let html = '<table class="data-table acts-table"><thead><tr>' +
                '<th>Вид</th><th>Наименование</th><th class="num">Сумма (без НДС)</th>' +
                '<th>Подписан</th><th></th>' +
                '</tr></thead><tbody>';

            for (const a of acts) {
                const kindLabel = a.kind === 'custom' && a.kind_custom ? a.kind_custom : a.kind_name;

                // Строка главного акта
                html += '<tr class="act-main">';
                html += '<td>' + escapeHtml(kindLabel) + '</td>';
                html += '<td><strong>' + escapeHtml(a.name) + '</strong></td>';
                // Основная строка — сумма, которая идёт на этот этап.
                // Если совпадает с полной суммой акта — показываем одну строку.
                // Если меньше — крупно часть, мелким полная.
                const stageAmount = a.stage_amount || 0;
                const actAmount   = a.amount || 0;
                const isPartial   = stageAmount > 0 && Math.abs(stageAmount - actAmount) > 0.01;

                const distributed = a.distributed || 0;
                const isUnder = distributed > 0 && Math.abs(distributed - actAmount) > 0.01;

                if (isPartial) {
                    html += '<td class="num">' +
                        '<strong>' + formatMoney(stageAmount) + '</strong>' +
                        '<div class="vat-hint">из ' + formatMoney(actAmount) + '</div>';
                    if (isUnder) {
                        html += '<div class="vat-hint">распределено ' +
                            formatMoney(distributed) + '</div>';
                    }
                    html += '</td>';
                } else {
                    html += '<td class="num"><strong>' + formatMoney(actAmount) + '</strong>';
                    if (isUnder) {
                        html += '<div class="vat-hint">распределено ' +
                            formatMoney(distributed) + '</div>';
                    }
                    html += '</td>';
                }
                html += '<td>' + (a.sign_date || '') + '</td>';
                 html += '<td class="actions">';
                if (a.file_path) {
                    html += '<a href="/contracts/' + a.contract_id + '/acts/' + a.guid + '/open" class="btn btn-sm" target="_blank" title="Открыть файл">📄</a>';
                }
                html += '<a href="/contracts/' + a.contract_id + '/acts/' + a.guid + '/edit" class="btn btn-sm">Изменить</a>';

                // Формируем текст подтверждения в зависимости от наличия приложений
                let confirmText = 'Удалить акт';
                if (a.attachments && a.attachments.length > 0) {
                    confirmText += ' и ' + a.attachments.length +
                        ' ' + pluralizeAppendices(a.attachments.length) + ' к нему';
                }
                confirmText += '?';

                html += '<form method="post" action="/contracts/' + a.contract_id + '/acts/' + a.guid + '/delete" ' +
                    'onsubmit="return confirm(\'' + confirmText.replace(/'/g, "\\'") + '\');" ' +
                    'style="display:inline;">';
                html += '<button type="submit" class="btn btn-sm btn-danger">×</button>';
                html += '</form>';
                html += '</td>';
                html += '</tr>';

                // Строки приложений (если есть)
                if (a.attachments && a.attachments.length > 0) {
                    for (const at of a.attachments) {
                        const atKind = at.kind === 'custom' && at.kind_custom ? at.kind_custom : at.kind_name;
                        html += '<tr class="act-child">';
                        html += '<td><span class="tree-mark">└─</span> ' + escapeHtml(atKind) + '</td>';
                        html += '<td>' + escapeHtml(at.name) + '</td>';
                        html += '<td class="num muted">—</td>';
                        html += '<td class="muted">—</td>';
                         html += '<td class="actions">';
                        if (at.file_path) {
                            html += '<a href="/contracts/' + a.contract_id + '/attachments/' + at.guid + '/open" class="btn btn-sm" target="_blank" title="Открыть файл">📄</a>';
                        }
                        html += '<a href="/contracts/' + a.contract_id + '/attachments/' + at.guid + '/edit" class="btn btn-sm">Изменить</a>';
                        html += '<form method="post" action="/contracts/' + a.contract_id + '/attachments/' + at.guid + '/delete" onsubmit="return confirm(\'Удалить приложение?\');" style="display:inline;">';
                        html += '<button type="submit" class="btn btn-sm btn-danger">×</button>';
                        html += '</form>';
                        html += '</td>';
                        html += '</tr>';
                    }
                }

                // Строка "добавить приложение" для главного акта
                html += '<tr class="act-child-add">';
                html += '<td colspan="5">';
                html += '<a href="/contracts/' + a.contract_id + '/acts/' + a.guid + '/attachments/new" class="btn btn-sm">+ Добавить приложение</a>';
                html += '</td>';
                html += '</tr>';
            }

            html += '</tbody></table>';
            listBox.innerHTML = html;
        } catch (err) {
            console.error('renderActs error:', err, acts);
            listBox.innerHTML = '<p class="error">Ошибка: ' + err.message + '</p>';
        }
    }

    function escapeHtml(s) {
        if (!s) return '';
        return String(s).replace(/[&<>"']/g, c => ({
            '&': '&amp;', '<': '&lt;', '>': '&gt;',
            '"': '&quot;', "'": '&#39;'
        }[c]));
    }

    function formatMoney(v) {
        if (!v && v !== 0) return '';
        const s = Number(v).toFixed(2);
        return s.replace(/\B(?=(\d{3})+(?!\d))/g, ' ') + ' ₽';
    }

     const closedInput = document.getElementById('acts-modal-closed-input');
    const toggleBtn   = document.getElementById('acts-modal-toggle');
    const modeLabel   = document.getElementById('acts-modal-mode');

    let currentStage = { contractID: '', stageGUID: '', manual: true };

        function renderSummary(header, contractID, stageGUID) {
        if (!header || !header.stage_guid) {
            closedInput.value = '';
            return;
        }
        currentStage = {
            contractID,
            stageGUID,
            manual: !!header.closed_by_acts_manual,
        };

        // Заполняем поле
        if (window.inputMoney) {
            closedInput.value = window.inputMoney(header.closed_by_acts.toFixed(2));
        } else {
            closedInput.value = header.closed_by_acts.toFixed(2);
        }

        applyModeUI();
    }

    // applyModeUI управляет состоянием поля и кнопки в зависимости от режима
     function applyModeUI() {
        const saveBtn = document.getElementById('acts-modal-save');

        if (currentStage.manual) {
            // Ручной ввод
            closedInput.readOnly = false;
            closedInput.classList.remove('is-auto');
            toggleBtn.textContent = '🔒';
            toggleBtn.title = 'Сейчас ручной ввод. Нажмите, чтобы подтягивать из актов';
            toggleBtn.classList.add('locked');
            modeLabel.textContent = 'ручной';
            if (saveBtn) {
                saveBtn.disabled = false;
                saveBtn.style.opacity = '1';
                saveBtn.style.cursor = 'pointer';
            }
        } else {
            // Из актов
            closedInput.readOnly = true;
            closedInput.classList.add('is-auto');
            toggleBtn.textContent = '🔗';
            toggleBtn.title = 'Сейчас подтяжка из актов. Нажмите для ручного ввода';
            toggleBtn.classList.remove('locked');
            modeLabel.textContent = 'из актов';
            if (saveBtn) {
                saveBtn.disabled = true;
                saveBtn.style.opacity = '0.4';
                saveBtn.style.cursor = 'not-allowed';
            }
        }
    }

    // Клик по 🔒/🔗: переключение режима + сохранение ручного значения
     if (toggleBtn) {
        toggleBtn.addEventListener('click', () => {
            if (!currentStage.stageGUID) return;
            const url = '/contracts/' + currentStage.contractID +
                '/stages/' + currentStage.stageGUID + '/acts/toggle-manual';
            fetch(url, { method: 'POST' })
                .then(r => r.json())
                .then(data => {
                    currentStage.manual = !!data.closed_by_acts_manual;
                    if (window.inputMoney) {
                        closedInput.value = window.inputMoney(data.closed_by_acts.toFixed(2));
                    } else {
                        closedInput.value = data.closed_by_acts.toFixed(2);
                    }
                    applyModeUI();
                    // Обновляем страницу, но после загрузки снова открываем модалку
                    reloadAndReopen(currentStage.contractID, currentStage.stageGUID, nameBox.textContent);
                })
                .catch(() => {});
        });
    }

    // Сохранение ручного значения при потере фокуса / по Enter
    const saveBtn = document.getElementById('acts-modal-save');

    if (closedInput) {
        function saveManual() {
            if (!currentStage.stageGUID) return;
            if (!currentStage.manual) {
                alert('Сейчас режим «из актов». Нажмите 🔗 → 🔒, чтобы ввести вручную.');
                return;
            }
            const url = '/contracts/' + currentStage.contractID +
                '/stages/' + currentStage.stageGUID + '/acts/set-closed';
            const body = new URLSearchParams();
            body.append('closed_by_acts', closedInput.value);
            fetch(url, {
                method: 'POST',
                headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                body: body.toString(),
            }).then(() => {
                closedInput.blur();
            }).catch(() => {});
        }

        closedInput.addEventListener('blur', saveManual);
        closedInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                e.preventDefault();
                if (!currentStage.manual) return;
                saveManual();
            }
        });
    }

    if (saveBtn) {
        saveBtn.addEventListener('click', () => {
            if (!currentStage.stageGUID) return;
            if (!currentStage.manual) {
                alert('Сейчас режим «из актов». Нажмите 🔗, чтобы переключить на ручной ввод.');
                return;
            }
            const url = '/contracts/' + currentStage.contractID +
                '/stages/' + currentStage.stageGUID + '/acts/set-closed';
            const body = new URLSearchParams();
            body.append('closed_by_acts', closedInput.value);
            fetch(url, {
                method: 'POST',
                headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                body: body.toString(),
            }).then(() => {
                reloadAndReopen(currentStage.contractID, currentStage.stageGUID, nameBox.textContent);
            }).catch(() => {});
        });
    }

    // reloadAndReopen сохраняет контекст модалки в sessionStorage и перезагружает
    // страницу. При загрузке страницы — модалка снова откроется.
    function reloadAndReopen(contractID, stageGUID, stageName) {
        try {
            sessionStorage.setItem('reopen_modal', JSON.stringify({
                contractID: contractID,
                stageGUID: stageGUID,
                stageName: stageName || ''
            }));
        } catch (e) { /* ignore */ }
        window.location.reload();
    }

    // pluralizeAppendices возвращает правильную форму слова «приложение».
    function pluralizeAppendices(n) {
        const mod10 = n % 10;
        const mod100 = n % 100;

        if (mod10 === 1 && mod100 !== 11) return 'приложение';
        if (mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20)) return 'приложения';
        return 'приложений';
    }

})();


