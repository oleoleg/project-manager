document.addEventListener('DOMContentLoaded', function() {

    // 1. Функция пересчета итогов и проверки ошибок
    function recalcTotals() {
        let totals = {
            sale: 0, fot: 0, days: 0, travel: 0, 
            eqSale: 0, eqCost: 0, sub: 0, comm: 0, other: 0
        };
        
        const calcData = document.getElementById('calc-data');
        const overheadRate = calcData ? parseFloat(calcData.dataset.overhead) : 1.0;

        let hasErrors = false;
        const errorMessages = [];

        document.querySelectorAll('.costs-table tbody tr.cost-row').forEach(tr => {
            // Пропускаем удаленные строки
            if (tr.classList.contains('is-deleted')) return;

            const getVal = (name) => {
                const el = tr.querySelector(`input[name="${name}[]"]`);
                return el ? parseMoneyText(el.value) : 0;
            };

            totals.sale   += getVal('cost_amount');
            totals.fot    += getVal('cost_fot_plan');
            totals.days   += getVal('cost_labor_days');
            totals.travel += getVal('cost_travel');
            totals.eqSale += getVal('cost_equipment_sale');
            totals.eqCost += getVal('cost_equipment_cost');
            totals.sub    += getVal('cost_subcontract');
            totals.comm   += getVal('cost_commission');
            totals.other  += getVal('cost_other');

            // ВЫЗОВ ПРОВЕРКИ ДЛЯ КАЖДОЙ СТРОКИ
            if (window.StageCostsChecks && StageCostsChecks.checkRow) {
                // Находим номер и название этапа (из заголовка выше)
                let stageCost = 0;
                let prev = tr.previousElementSibling;
                let stageName = 'Этап';
                let stageNumber = '?';
                
                while (prev) {
                    if (prev.classList.contains('stage-header-row')) {
                        const numEl = prev.querySelector('td:first-child');
                        if (numEl) stageNumber = numEl.textContent.trim();
                        
                        const nameEl = prev.querySelector('td:nth-child(2)');
                        if (nameEl) stageName = nameEl.childNodes[0].textContent.trim();
                        break;
                    }
                    prev = prev.previousElementSibling;
                }

                const check = StageCostsChecks.checkRow(tr, 0, overheadRate); 
                if (check.hasError) {
                    hasErrors = true;
                    const fullMsg = `Этап ${stageNumber} (${stageName}): ${check.message}`;
                    errorMessages.push(fullMsg);
                    tr.classList.add('row-error'); 
                    tr.title = fullMsg; 
                } else {
                    tr.classList.remove('row-error');
                    tr.removeAttribute('title'); 
                }
            }
        });

        const overheadAmount = totals.fot * overheadRate;
        const totalCost = totals.fot + overheadAmount + totals.travel + totals.eqCost + totals.sub + totals.comm + totals.other;

        const elSaleAndEq = document.getElementById('total-sale-and-eq');
        const elFot = document.getElementById('total-fot');
        const elEqCost = document.getElementById('total-eq-cost');
        const elCost = document.getElementById('total-cost');

        if (elSaleAndEq) elSaleAndEq.textContent = formatMoney(totals.sale + totals.eqSale);
        if (elFot) elFot.textContent = formatMoney(totals.fot);
        if (elEqCost) elEqCost.textContent = formatMoney(totals.eqCost);
        if (elCost) elCost.textContent = formatMoney(totalCost);

        const setFoot = (id, val, isMoney = true) => {
            const el = document.getElementById(id);
            if (el) el.textContent = isMoney ? formatMoney(val) : Math.round(val);
        };

        setFoot('foot-total-sale', totals.sale);
        setFoot('foot-total-fot', totals.fot);
        setFoot('foot-total-days', totals.days, false);
        setFoot('foot-total-travel', totals.travel);
        setFoot('foot-total-eq-sale', totals.eqSale);
        setFoot('foot-total-eq-cost', totals.eqCost);
        setFoot('foot-total-sub', totals.sub);
        setFoot('foot-total-comm', totals.comm);
        setFoot('foot-total-other', totals.other);

         // --- ПРОВЕРКА СУММ ПО ЭТАПАМ ---
        if (window.StageCostsChecks && StageCostsChecks.checkStageTotals) {
            const tableBody = document.querySelector('.costs-table tbody');
            if (tableBody) {
                const stageCheck = StageCostsChecks.checkStageTotals(tableBody);
                if (stageCheck.hasErrors) {
                    hasErrors = true;
                    // Важно: добавляем ошибки этапов к уже собранным ошибкам строк
                    stageCheck.errorMessages.forEach(msg => {
                        if (!errorMessages.includes(msg)) {
                            errorMessages.push(msg);
                        }
                    });
                    
                    // Для ошибок этапов (превышение суммы) добавим тултип на заголовок этапа
                    // Для простоты пройдемся по всем заголовкам с ошибками
                    document.querySelectorAll('.stage-header-row.row-error').forEach(header => {
                         const stageNum = header.querySelector('td:first-child').textContent.trim();
                         const relatedError = stageCheck.errorMessages.find(m => m.includes(`Этап ${stageNum}`));
                         if (relatedError) {
                             header.title = relatedError;
                         }
                    });
                }
            }
        }
        
        // Вывод предупреждений наверх (над таблицей)
        const warnBox = document.getElementById('costs-warning-box');
        const warnList = document.getElementById('costs-warning-list');
        if (warnBox && warnList) {
            if (hasErrors) {
                warnList.innerHTML = errorMessages.map(msg => `<li>${msg}</li>`).join('');
                warnBox.style.display = 'block';
            } else {
                warnBox.style.display = 'none';
            }
        }
    }
 
    // 2. Инициализация замочков и пересчета
    const tableBody = document.querySelector('.costs-table tbody');
    if (tableBody && window.StageCostsChecks) {
        try {

            tableBody.addEventListener('input', (e) => {
                const row = e.target.closest('tr.cost-row');
                // Игнорируем ввод в удаленных строках
                if (!row || row.classList.contains('is-deleted')) return;

                // Если поле readonly — не пересчитываем (оно auto)
                if (e.target.readOnly) return;
                                
                setTimeout(() => {
                    StageCostsChecks.recalcRow(row);
                    recalcTotals();
                }, 0);
            });
            
            tableBody.querySelectorAll('tr.cost-row').forEach(row => {
                if (!row.classList.contains('is-deleted')) {
                    StageCostsChecks.recalcRow(row, '');
                }
            });
        } catch (err) {
            console.error('Ошибка инициализации:', err);
        }
    }

     // Логика переключения "auto" при клике на кнопку [A]
    tableBody.addEventListener('click', (e) => {
        const btn = e.target.closest('.auto-toggle-btn');
        if (!btn) return; // Если кликнули не по кнопке — выходим

        e.preventDefault();
        const row = btn.closest('tr');
        const calcFieldHidden = row.querySelector('input[name="cost_calc_field[]"]');
        if (!calcFieldHidden) return;

        // Определяем, какое это поле (по инпуту в этой же ячейке)
        const input = btn.parentNode.querySelector('input:not([type="hidden"])');
        if (!input) return;
        const fieldName = input.name.replace('[]', '');

       // Если кнопка уже активна (auto уже стоит здесь) — ничего не делаем
        if (btn.classList.contains('active')) {
            return;
        }

        // Иначе переносим auto на это поле
        if (fieldName === 'cost_fot_plan') calcFieldHidden.value = 'fot_plan';
        else if (fieldName === 'cost_labor_days') calcFieldHidden.value = 'labor_days';
        else if (fieldName === 'cost_rate_per_day') calcFieldHidden.value = 'rate_per_day';

        // Перерисовываем строку (переключаем классы и readonly)
        updateRowAutoState(row);
    });

    // Вспомогательная функция для обновления визуального состояния строки
    function updateRowAutoState(row) {
        const calcFieldHidden = row.querySelector('input[name="cost_calc_field[]"]');
        const calcField = calcFieldHidden ? calcFieldHidden.value : '';

        // Список полей
        const fields = {
            'fot_plan': row.querySelector('input[name="cost_fot_plan[]"]'),
            'labor_days': row.querySelector('input[name="cost_labor_days[]"]'),
            'rate_per_day': row.querySelector('input[name="cost_rate_per_day[]"]')
        };

        // Снимаем readonly и классы со всех
        Object.values(fields).forEach(el => {
            if (!el) return;
            el.readOnly = false;
            el.classList.remove('is-auto');
            el.removeAttribute('tabindex');
        });

        // Убираем активные классы со всех кнопок
        row.querySelectorAll('.auto-toggle-btn').forEach(b => b.classList.remove('active'));

        // Применяем к выбранному полю
        if (fields[calcField]) {
            fields[calcField].readOnly = true;
            fields[calcField].setAttribute('tabindex', '-1');
            fields[calcField].classList.add('is-auto');
            
            // Находим кнопку в этой же ячейке и делаем её активной
            const btn = fields[calcField].parentNode.querySelector('.auto-toggle-btn');
            if (btn) btn.classList.add('active');
        }
    }
 
    // 3. Сортировка (заглушка)
    window.sortCostsTable = function(field, btnElement) {
        alert('Сортировка временно недоступна из-за группировки по этапам.');
    };

    // 4. Делегирование кликов (только если не режим чтения)
    const isReadOnly = document.getElementById('costs-table')?.dataset.readonly === 'true';
    if (tableBody && !isReadOnly) {
        tableBody.addEventListener('click', (e) => {
            // --- ДОБАВЛЕНИЕ ---
            if (e.target.classList.contains('add-cost-btn')) {
                e.preventDefault();
                const headerRow = e.target.closest('tr');
                const stageGUID = headerRow.dataset.stageGuid;
                const workKind = headerRow.dataset.workKind;

                const tr = document.createElement('tr');
                tr.className = 'cost-row is-new'; // Добавляем класс для стилизации
                tr.dataset.stageGuid = stageGUID;

                // Создаем селект с текущим значением
                let workKindHTML = `<select name="cost_work_kind[]" class="fact-input" style="width: 100px;">
                    <option value="">— выберите —</option>
                    <option value="ПИР" ${workKind === 'ПИР' ? 'selected' : ''}>ПИР</option>
                    <option value="СМР" ${workKind === 'СМР' ? 'selected' : ''}>СМР</option>
                    <option value="ПНР" ${workKind === 'ПНР' ? 'selected' : ''}>ПНР</option>
                    <option value="ТО" ${workKind === 'ТО' ? 'selected' : ''}>ТО</option>
                    <option value="Сборка" ${workKind === 'Сборка' ? 'selected' : ''}>Сборка</option>
                    <option value="Ремонт" ${workKind === 'Ремонт' ? 'selected' : ''}>Ремонт</option>
                    <option value="Продажа" ${workKind === 'Продажа' ? 'selected' : ''}>Продажа</option>
                </select>`;

                tr.innerHTML = `
                    <input type="hidden" name="cost_stage_guid[]" value="${stageGUID}">
                    <td></td>
                    <td>${workKindHTML}</td>
                    <td class="num"><input type="text" name="cost_amount[]" class="money-input fact-input"></td>
                    
                    <!-- ФОТп (по умолчанию auto) -->
                    <td class="num cell-fot" style="position: relative;">
                        <input type="text" name="cost_fot_plan[]" class="money-input fact-input is-auto" readonly tabindex="-1">
                        <input type="hidden" name="cost_calc_field[]" value="fot_plan">
                        <button type="button" class="auto-toggle-btn active" title="Вычисляется автоматически">A</button>
                    </td>
                    
                    <td class="num cell-ratio"><input type="text" class="ratio-display fact-input" readonly tabindex="-1" style="background: transparent; border: none;"></td>
                    
                    <!-- Дни -->
                    <td class="num cell-days" style="position: relative;">
                        <input type="text" name="cost_labor_days[]" class="days-input fact-input">
                        <button type="button" class="auto-toggle-btn" title="Сделать вычисляемым">A</button>
                    </td>
                    
                    <!-- Ставка -->
                    <td class="num cell-rate" style="position: relative;">
                        <input type="text" name="cost_rate_per_day[]" class="money-input fact-input">
                        <button type="button" class="auto-toggle-btn" title="Сделать вычисляемым">A</button>
                    </td>
                    
                    <td class="num"><input type="text" name="cost_travel[]" class="money-input fact-input"></td>
                    <td class="num"><input type="text" name="cost_equipment_sale[]" class="money-input fact-input"></td>
                    <td class="num"><input type="text" name="cost_equipment_cost[]" class="money-input fact-input"></td>
                    <td class="num"><input type="text" name="cost_subcontract[]" class="money-input fact-input"></td>
                    <td class="num"><input type="text" name="cost_commission[]" class="money-input fact-input"></td>
                    <td class="num"><input type="text" name="cost_other[]" class="money-input fact-input"></td>
                    <td style="text-align: center;">
                        <button type="button" class="btn btn-sm btn-danger remove-cost-btn" title="Удалить">×</button>
                    </td>
                `;

                if (window.attachMoneyInput) {
                    tr.querySelectorAll('input.money-input').forEach(window.attachMoneyInput);
                }

                // Вставляем после последней строки этого этапа
                let nextRow = headerRow.nextElementSibling;
                while (nextRow && nextRow.classList.contains('cost-row')) {
                    nextRow = nextRow.nextElementSibling;
                }
                
                if (nextRow) {
                    tableBody.insertBefore(tr, nextRow);
                } else {
                    tableBody.appendChild(tr);
                }

                if (window.StageCostsChecks) StageCostsChecks.recalcRow(tr, '');
                recalcTotals();
            }

            // --- УДАЛЕНИЕ ---
            if (e.target.classList.contains('remove-cost-btn')) {
                e.preventDefault();
                const row = e.target.closest('tr');
                
                // Если строка уже удалена, не реагируем
                if (row.classList.contains('is-deleted')) return;

                row.classList.add('is-deleted');
                
                // Отключаем все поля ввода, чтобы они не отправлялись на сервер
                row.querySelectorAll('input, select').forEach(el => {
                    el.disabled = true;
                });
                
                recalcTotals();
            }
        });
    }

    // Первоначальный расчет
    recalcTotals();
});