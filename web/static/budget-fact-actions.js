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
    const stageGUID = selectEl.value;
    
    // Обновляем скрытое поле stage_guid
    const hiddenGuid = row.querySelector('input[name="stage_guid[]"]');
    if (hiddenGuid) hiddenGuid.value = stageGUID;

    const workKindSelect = row.querySelector('.workkind-select');
    if (!workKindSelect) return;

    const availableWks = window.stageWorkKinds[stageGUID] || [];
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
    
    if (!workKindSelect.value && workKindSelect.options.length > 1) {
        workKindSelect.selectedIndex = 1; // 0-й это "— выберите —"
    }

    const wkhidden = row.querySelector('input[name="work_kind_code[]"]');
    if (wkhidden) wkhidden.value = workKindSelect.value;
}

function deleteRow(btn) {
    if (!confirm('Удалить эту строку?')) return;
    const row = btn.closest('tr');
    row.remove();
    recalculateTotals();
}

function addNewRow(opType, event) {
    const btn = event.target;
    const btnRow = btn.closest('tr');
    if (!window.stageOptionsHTML || window.stageOptionsHTML.trim() === '') {
        alert('Сначала добавьте хотя бы один этап в договор.');
        return;
    }
    const tbody = document.getElementById('fact-table').querySelector('tbody');
    const newRow = document.createElement('tr');
    newRow.className = 'fact-row';
    // Устанавливаем атрибуты для сортировки
    newRow.dataset.type = opType;
    newRow.dataset.date = Math.floor(Date.now() / 1000); // текущая дата как временная метка
    // Этап и вид работы проставим чуть позже, после выбора
    
    const isPayment = opType === 'Платежи по договору';
    const isSalary = opType === 'Зарплата сотрудникам' || opType === 'Накладные';
    
    let workKindsHTML = '<option value="" disabled selected hidden>— выберите —</option>';
    if (window.workKinds && window.workKinds.length > 0) {
        window.workKinds.forEach(wk => {
            workKindsHTML += `<option value="${wk.Code}">${wk.Name}</option>`;
        });
    }

    newRow.innerHTML = `
        <td style="text-align: center;">
            <select name="stage_select" class="fact-input stage-select" style="width: 100%; font-size: 12px;" onchange="filterWorkKinds(this)">
                ${window.stageOptionsHTML}
            </select>
        </td>
        <td>
            <input type="hidden" name="fact_id[]" value="">
            <input type="hidden" name="stage_guid[]" value="">
            <input type="hidden" name="work_kind_code[]" class="workkind-hidden" value="">
            <input type="hidden" name="operation_type[]" value="${opType}">
            <select class="fact-input workkind-select">
                ${workKindsHTML}
            </select>
        </td>
        <td><textarea name="name[]" class="fact-input" rows="2" placeholder="Новая операция"></textarea></td>
        <td><input type="date" name="operation_date[]" class="fact-input"></td>
        <td><input type="number" name="delay_days[]" value="0" class="fact-input" onchange="updatePayDate(this)"></td>
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
    
    tbody.insertBefore(newRow, btnRow);
    
    if (window.attachMoneyInput) {
        newRow.querySelectorAll('input.money-input').forEach(window.attachMoneyInput);
    }
    
    const today = new Date().toISOString().split('T')[0];
    const dateInput = newRow.querySelector('input[name="operation_date[]"]');
    dateInput.value = today;
    dateInput.addEventListener('change', function() { updatePayDate(this); }); // Вешаем обработчик здесь
    
    updatePayDate(dateInput);

    // Автоматически применяем фильтр для первого выбранного этапа
    const stageSelect = newRow.querySelector('.stage-select');
    if (stageSelect) {
        // Принудительно выбираем первый этап и проставляем ему data-атрибуты для сортировки
        if (stageSelect.options.length > 0) {
            stageSelect.selectedIndex = 0;
            newRow.dataset.stage = stageSelect.options[0].getAttribute('data-stage-num') || '0';
        }
        filterWorkKinds(stageSelect);
        
        // Принудительно выбираем первый доступный вид работы, если он есть
        const wkSelect = newRow.querySelector('.workkind-select');
        if (wkSelect) {
            if (wkSelect.options.length > 1) {
                wkSelect.selectedIndex = 1; // 0-й это скрытый "— выберите —"
            }
            const wkhidden = newRow.querySelector('input[name="work_kind_code[]"]');
            if (wkhidden) wkhidden.value = wkSelect.value;
        }
    }
    
    // Принудительно обновляем скрытое поле work_kind_code из селекта
    const wkSelect = newRow.querySelector('.workkind-select');
    const wkhidden = newRow.querySelector('input[name="work_kind_code[]"]');
    if (wkSelect && wkhidden) {
        wkhidden.value = wkSelect.value;
        // Добавляем обработчик для ручного выбора
        wkSelect.addEventListener('change', function() {
            wkhidden.value = this.value;
        });
    }

    recalculateTotals();
}



function toggleSalaryRows() {
    const checkbox = document.getElementById('show-salary');
    document.querySelectorAll('.salary-row').forEach(row => {
        row.style.display = checkbox.checked ? '' : 'none';
    });
}