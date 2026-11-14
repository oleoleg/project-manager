// Автоматический расчет даты платежа
function updatePayDate(element) {
    const row = element.closest('tr');
    const opDateInput = row.querySelector('input[name="operation_date[]"]');
    const delayInput = row.querySelector('input[name="delay_days[]"]');
    const payDateInput = row.querySelector('input[name="payment_date[]"]');

    if (!opDateInput || !opDateInput.value) return;
    const delay = parseInt(delayInput.value) || 0;
    
    const date = new Date(opDateInput.value);
    date.setDate(date.getDate() + delay);
    
    const yyyy = date.getFullYear();
    const mm = String(date.getMonth() + 1).padStart(2, '0');
    const dd = String(date.getDate()).padStart(2, '0');
    
    if (payDateInput) payDateInput.value = `${yyyy}-${mm}-${dd}`;
}

function toggleRowInputs(checkbox) {
    const row = checkbox.closest('tr');
    
    if (checkbox.checked) {
        row.classList.add('confirmed');
    } else {
        row.classList.remove('confirmed');
    }

    const inputs = row.querySelectorAll('input.fact-input, textarea.fact-input, input.money-input');
    
    inputs.forEach(input => {
        if (checkbox.checked) {
            input.setAttribute('readonly', 'readonly');
        } else {
            input.removeAttribute('readonly');
        }
    });
    
    recalculateTotals();
}

// Фильтрация видов работ в зависимости от выбранного этапа
function filterWorkKinds(selectEl) {
    const row = selectEl.closest('tr');
    const selectedOption = selectEl.options[selectEl.selectedIndex];
    if (!selectedOption) return;

    const stageNum = selectedOption.getAttribute('data-stage-num') || '';
    const stageNameText = selectedOption.textContent.split('—').pop().trim();
    const rowOrder = selectedOption.getAttribute('data-row-order') || '0';

    // Обновляем скрытые поля
    const hiddenNum = row.querySelector('input[name="stage_number[]"]');
    if (hiddenNum) hiddenNum.value = stageNum;

    const hiddenName = row.querySelector('input[name="stage_name[]"]');
    if (hiddenName) hiddenName.value = stageNameText;

    // Обновляем data-атрибуты для сортировки
    row.dataset.stage = stageNum;
    row.dataset.stageName = selectEl.options[selectEl.selectedIndex].textContent.trim();
    row.dataset.order = rowOrder;
    
    // Получаем список видов работ для этого этапа (по номеру)
    const availableWks = window.stageWorkKinds[stageNum] || [];
    const workKindSelect = row.querySelector('.workkind-select');
    if (!workKindSelect) return;
    const currentVal = workKindSelect.value;
    
     workKindSelect.innerHTML = '<option value="" disabled selected hidden>— выберите —</option>';

    if (window.workKinds && window.workKinds.length > 0) {
        window.workKinds.forEach(wk => {
            if (availableWks.length === 0 || availableWks.includes(wk.Code)) {
                const opt = document.createElement('option');
                opt.value = wk.Code;
                opt.textContent = wk.Name;
                if (wk.Code === currentVal) opt.selected = true;
                workKindSelect.appendChild(opt);
            }
        });
    }
    
    // Если в этапе только один вид работы, выбираем его и блокируем выбор
    if (workKindSelect.options.length === 2) { // 1 скрытый + 1 реальный
        workKindSelect.selectedIndex = 1;
        workKindSelect.disabled = true;
    } else if (workKindSelect.options.length > 2) {
        // Если комплексный, просто выбираем первый доступный
        workKindSelect.selectedIndex = 1;
        workKindSelect.disabled = false;
    } else {
        // Если видов работ нет, оставляем пустым
        workKindSelect.selectedIndex = 0;
        workKindSelect.disabled = false;
    }

    const wkhidden = row.querySelector('input[name="work_kind[]"]');
    if (wkhidden) {
        wkhidden.value = workKindSelect.selectedIndex > 0 ? workKindSelect.value : "";
    }
}

function deleteRow(btn) {
    if (!confirm('Удалить эту строку?')) return;
    const row = btn.closest('tr');
    const idInput = row.querySelector('input[name="fact_id[]"]');
    const factID = idInput ? idInput.value : '';

    // Если строка новая (нет ID), просто удаляем из HTML
    if (!factID) {
        row.remove();
        recalculateTotals();
        return;
    }

    // Если строка есть в БД, отправляем запрос на удаление
    // Для этого нам нужен ID договора (его можно взять из URL или data-атрибута)
    const contractID = window.location.pathname.split('/')[2]; // /contracts/6666/budget/fact

    const formData = new URLSearchParams();
    formData.append('id', factID);

    fetch(`/contracts/${contractID}/budget/fact/delete-row`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: formData.toString()
    })
    .then(resp => {
        if (resp.ok) {
            row.remove();
            recalculateTotals();
        } else {
            alert('Ошибка при удалении строки');
        }
    })
    .catch(err => {
        console.error(err);
        alert('Ошибка сети');
    });
}

function addNewRowFromGroup(btn, groupData) {
    const finalData = {
        opType: groupData.type || 'Платежи по договору',
        stageNum: groupData.stage || '',
        stageNameFull: groupData.stageName || '',
        workKind: groupData.work || '',
        opDate: groupData.date || new Date().toISOString().split('T')[0], // Берем дату, переданную из кнопки
        delay: groupData.delay || '0',
        rowOrder: '0'
    };

    if (groupData.stage) {
        const stageOpt = Array.from(document.querySelectorAll('.stage-select option')).find(o => o.value === groupData.stage);
        if (stageOpt) finalData.rowOrder = stageOpt.getAttribute('data-row-order') || '0';
    }

    if (!finalData.stageNum && window.stageOptionsHTML) {
        const tempDiv = document.createElement('div');
        tempDiv.innerHTML = window.stageOptionsHTML;
        const firstOpt = tempDiv.querySelector('option');
        if (firstOpt) {
            finalData.stageNum = firstOpt.value;
            finalData.stageNameFull = firstOpt.textContent;
            finalData.rowOrder = firstOpt.getAttribute('data-row-order') || '0';
        }
    }

    createFactRow(btn, finalData);
}

// Вспомогательная функция для создания строки (чтобы не дублировать код)
function createFactRow(btn, data) {
    const tbody = document.getElementById('fact-table').querySelector('tbody');
    const newRow = document.createElement('tr');
    newRow.className = 'fact-row';

    newRow.dataset.type = data.opType;
    newRow.dataset.date = Math.floor(new Date(data.opDate).getTime() / 1000);
    newRow.dataset.stage = data.stageNum || '0';
    newRow.dataset.stageName = data.stageNameFull || '';
    newRow.dataset.order = data.rowOrder;

    const isPayment = data.opType === 'Платежи по договору';
    const isSalary = data.opType === 'Зарплата сотрудникам' || data.opType === 'Накладные';
    
    let workKindsHTML = '<option value="" disabled selected hidden>— выберите —</option>';
    if (window.workKinds && window.workKinds.length > 0) {
        window.workKinds.forEach(wk => {
            if (wk.Code === 'Комплексный') return; 
            const selected = (wk.Code === data.workKind) ? 'selected' : '';
            workKindsHTML += `<option value="${wk.Code}" ${selected}>${wk.Name}</option>`;
        });
    }

    let stageOptions = window.stageOptionsHTML;
    if (data.stageNum) {
        stageOptions = stageOptions.replace(new RegExp(`value="${data.stageNum}"`), `value="${data.stageNum}" selected`);
    }

    newRow.innerHTML = `
        <td style="text-align: center;">
            <select name="stage_select" class="fact-input stage-select" style="width: 100%; font-size: 12px;" onchange="filterWorkKinds(this)">
                ${stageOptions}
            </select>
            <input type="hidden" name="stage_number[]" value="${data.stageNum}">
            <input type="hidden" name="stage_name[]" value="${data.stageNameFull.split('—').pop().trim()}">
        </td>
        <td>
            <input type="hidden" name="fact_id[]" value="">
            <input type="hidden" name="work_kind[]" class="workkind-hidden" value="${data.workKind}">
            <select class="fact-input workkind-select">
                ${workKindsHTML}
            </select>
        </td>
        <td>
            <select name="operation_type[]" class="fact-input op-type-select" style="width: 100%; font-size: 12px;" onchange="updateRowInputsByType(this)">
                <option value="Платежи по договору" ${data.opType === 'Платежи по договору' ? 'selected' : ''}>Платежи по договору</option>
                <option value="Закупка по договору" ${data.opType === 'Закупка по договору' ? 'selected' : ''}>Закупка по договору</option>
                <option value="Командировки" ${data.opType === 'Командировки' ? 'selected' : ''}>Командировки</option>
                <option value="Субподрядчики" ${data.opType === 'Субподрядчики' ? 'selected' : ''}>Субподрядчики</option>
                <option value="Комиссия" ${data.opType === 'Комиссия' ? 'selected' : ''}>Комиссия</option>
                <option value="Другое (расходы)" ${data.opType === 'Другое (расходы)' ? 'selected' : ''}>Другое (расходы)</option>
            </select>
        </td>
        <td><textarea name="name[]" class="fact-input" rows="2" placeholder="Новая операция"></textarea></td>
        <td><input type="date" name="operation_date[]" class="fact-input" value="${data.opDate}"></td>
        <td><input type="number" name="delay_days[]" value="${data.delay}" class="fact-input" onchange="updatePayDate(this)"></td>
        <td><input type="date" name="payment_date[]" class="fact-input" readonly></td>
        <td class="num">
            ${isPayment ? `<input type="text" name="fact_income[]" class="fact-input money-input">` : `<input type="hidden" name="fact_income[]" value="0"><span class="muted">—</span>`}
        </td>
        <td class="num">
            ${!isPayment ? `<input type="text" name="fact_expense[]" class="fact-input money-input">` : `<input type="hidden" name="fact_expense[]" value="0"><span class="muted">—</span>`}
        </td>
        <td style="text-align: center;">
            <input type="hidden" name="confirmed_id[]" value="">
            <input type="checkbox" name="is_confirmed[]" value="" ${isSalary ? 'disabled' : ''} onchange="toggleRowInputs(this)">
        </td>
        <td style="text-align: right; white-space: nowrap;">
            <button type="submit" class="btn btn-sm btn-primary" title="Сохранить">💾</button>
            <button type="button" class="btn btn-sm btn-danger" onclick="deleteRow(this)" title="Удалить">×</button>
        </td>
    `;
    
    const opTypeSelect = newRow.querySelector('.op-type-select');
    if (opTypeSelect) {
        opTypeSelect.value = data.opType;
        updateRowInputsByType(opTypeSelect); 
    }

    tbody.insertBefore(newRow, btn.closest('tr'));
    
    if (window.attachMoneyInput) {
        newRow.querySelectorAll('input.money-input').forEach(window.attachMoneyInput);
    }
    
    const dateInput = newRow.querySelector('input[name="operation_date[]"]');
    if (dateInput) {
        dateInput.addEventListener('change', function() { updatePayDate(this); }); 
        updatePayDate(dateInput);
    }

    const stageSelect = newRow.querySelector('.stage-select');
    if (stageSelect && data.stageNum) {
        filterWorkKinds(stageSelect);
        const wkSelect = newRow.querySelector('.workkind-select');
        const wkhidden = newRow.querySelector('input[name="work_kind[]"]');
        if (wkSelect && data.workKind) {
            wkSelect.value = data.workKind;
            if (wkhidden) wkhidden.value = data.workKind;
        }
    }

    recalculateTotals();
}

function toggleSalaryRows() {
    const checkbox = document.getElementById('show-salary');
    const isChecked = checkbox ? checkbox.checked : false;
    
    // 1. Скрываем/показываем сами строки
    document.querySelectorAll('.salary-row').forEach(row => {
        // Если это заголовок группы (tr), мы его не трогаем напрямую, 
        // его судьбой управляет сортировка
        if (row.classList.contains('group-header-row')) return;

        if (isChecked) {
            row.style.display = ''; 
        } else {
            row.style.display = 'none'; 
        }
    });

    // 2. Пересобираем таблицу заново, чтобы заголовки групп обновились
    const activeSortBtn = document.querySelector('.filter-bar .sort-btn.btn-primary');
    
    if (activeSortBtn) {
        if (activeSortBtn.textContent.includes('По этапам')) {
            sortFactTable('stage', activeSortBtn);
        } else if (activeSortBtn.textContent.includes('По дате')) {
            sortFactTable('date', activeSortBtn);
        } else if (activeSortBtn.textContent.includes('По типу')) {
            sortFactTable('type', activeSortBtn);
        } else if (activeSortBtn.textContent.includes('По виду')) {
            sortFactTable('work', activeSortBtn);
        }
    } else {
        // Если сортировка не выбрана, применяем дефолтную (По типу)
        const defaultBtn = Array.from(document.querySelectorAll('.filter-bar button')).find(b => b.textContent.includes('По типу'));
        if (defaultBtn) {
            sortFactTable('type', defaultBtn);
        }
    }
}

function updateRowInputsByType(selectEl) {
    const row = selectEl.closest('tr');
    const newType = selectEl.value;
    
    // Обновляем скрытое поле с типом операции
    // const hiddenType = row.querySelector('input[name="operation_type[]"]');
    // if (hiddenType) {
    //     hiddenType.value = newType;
    // }
    
    // Обновляем data-type для сортировки
    row.dataset.type = newType;
    
    const isPayment = newType === 'Платежи по договору';
    
    // Индексы колонок для ФинФакта:
    // 0 - №, 1 - Вид работы, 2 - Наименование, 3 - Тип операции, 4 - Дата, 5 - Отсрочка, 6 - Дата платежа, 7 - Приход, 8 - Расход
    const incCell = row.cells[7];
    const expCell = row.cells[8];
    
    if (isPayment) {
        incCell.innerHTML = `<input type="text" name="fact_income[]" class="fact-input money-input" value="">`;
        expCell.innerHTML = `<input type="hidden" name="fact_expense[]" value="0"><span class="muted">—</span>`;
    } else {
        incCell.innerHTML = `<input type="hidden" name="fact_income[]" value="0"><span class="muted">—</span>`;
        expCell.innerHTML = `<input type="text" name="fact_expense[]" class="fact-input money-input" value="">`;
    }
    
    // Применяем маску ввода, если есть
    if (window.attachMoneyInput) {
        row.querySelectorAll('input.money-input').forEach(window.attachMoneyInput);
    }
    
    recalculateTotals();
}


function validateAndSubmitForm() {
    document.querySelectorAll('select.op-type-select[disabled]').forEach(el => {
        el.removeAttribute('disabled');
    });

    const rows = document.querySelectorAll('#fact-table tbody tr.fact-row');
    for (let row of rows) {
        if (row.style.display === 'none' || row.classList.contains('salary-row')) continue;

        const stageSelect = row.querySelector('.stage-select');
        const workKindSelect = row.querySelector('.workkind-select');
        const dateInput = row.querySelector('input[name="operation_date[]"]');
        
        if (stageSelect && stageSelect.value === '') { alert('Выберите этап.'); return false; }
        if (workKindSelect && (workKindSelect.value === '' || workKindSelect.value === 'Комплексный')) { alert('Выберите вид работы (Комплексный недопустим).'); return false; }
        if (dateInput && dateInput.value === '') { alert('Укажите дату.'); return false; }
    }

    // Отправляем через AJAX
    const form = document.getElementById('fact-form');
    const formData = new FormData(form);
    
    fetch(form.action, {
        method: 'POST',
        body: new URLSearchParams(formData) // Отправляем как urlencoded, чтобы сервер понял
    })
    .then(response => {
        if (response.ok) {
            // Успех - перезагружаем страницу, чтобы обновить данные
            window.location.href = response.url; // или window.location.reload()
        } else {
            // Ошибка - показываем текст ошибки
            return response.text().then(text => {
                const errorBox = document.getElementById('error-box');
                if (errorBox) {
                    errorBox.textContent = text;
                    errorBox.style.display = 'block';
                    errorBox.scrollIntoView({ behavior: 'smooth' });
                } else {
                    alert(text);
                }
            });
        }
    })
    .catch(err => {
        console.error(err);
        alert('Ошибка сети при сохранении');
    });

    return false; // Предотвращаем обычную отправку формы
}


// Оставьте только эту версию, старую addNewRow можно удалить или сделать алиасом
function addNewRow(opType, event) {
    const btn = event.target;
    // Если старая кнопка вызывает addNewRow напрямую, передаем базовые данные
    addNewRowFromGroup(btn, { type: opType });
}