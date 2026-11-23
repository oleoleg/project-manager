document.addEventListener('DOMContentLoaded', function () {

    // ========== ФИЛЬТРЫ ==========
    const filterForm = document.querySelector('.filter-form');
    if (filterForm) {
        filterForm.querySelectorAll('select').forEach(sel => {
            sel.addEventListener('change', () => filterForm.submit());
        });

        filterForm.querySelectorAll('input:not([type="date"])').forEach(inp => {
            inp.addEventListener('keydown', e => {
                if (e.key === 'Enter') {
                    e.preventDefault();
                    filterForm.submit();
                }
            });
            inp.addEventListener('change', () => filterForm.submit());
        });

        filterForm.querySelectorAll('input[type="date"]').forEach(inp => {
            let lastValue = inp.value;
            inp.addEventListener('keydown', e => {
                if (e.key === 'Enter') {
                    e.preventDefault();
                    filterForm.submit();
                }
            });
            inp.addEventListener('blur', () => {
                if (inp.value !== lastValue) {
                    lastValue = inp.value;
                    filterForm.submit();
                }
            });
        });
    }

       // ========== РЕДАКТИРОВАНИЕ ==========
    const editForm = document.getElementById('reports-form');
    if (!editForm) return;

    const btnEdit = document.getElementById('btn-edit-mode');
    const btnSave = document.getElementById('btn-save-mode');
    const btnCancel = document.getElementById('btn-cancel-edit');
    const toolbar = document.getElementById('edit-toolbar');
    const hint = document.getElementById('edit-hint');

    // Хранилище исходных значений: id строки -> { work_kind, prj_stage, prj_id }
    let originalValues = {};

    function snapshotRow(tr) {
        const id = tr.getAttribute('data-id');
        originalValues[id] = {
            work_kind: tr.querySelector('select[name="work_kind[]"]').value,
            prj_stage: tr.querySelector('input[name="prj_stage[]"]').value,
            prj_id:    tr.querySelector('input[name="prj_id[]"]').value,
        };
    }

    function rowChanged(tr) {
        const id = tr.getAttribute('data-id');
        const orig = originalValues[id];
        if (!orig) return false;
        const cur = {
            work_kind: tr.querySelector('select[name="work_kind[]"]').value,
            prj_stage: tr.querySelector('input[name="prj_stage[]"]').value,
            prj_id:    tr.querySelector('input[name="prj_id[]"]').value,
        };
        return cur.work_kind !== orig.work_kind
            || cur.prj_stage !== orig.prj_stage
            || cur.prj_id !== orig.prj_id;
    }

    function updateRowState(tr) {
        if (rowChanged(tr)) {
            tr.classList.add('row-changed');
        } else {
            tr.classList.remove('row-changed');
        }
        updateSaveButtonLabel();
    }

    function updateSaveButtonLabel() {
        if (btnSave.style.display === 'none') return;
        const changed = editForm.querySelectorAll('.row-changed').length;
        btnSave.textContent = changed > 0
            ? `💾 Сохранить (изменено: ${changed})`
            : '💾 Сохранить';
        btnSave.disabled = changed === 0;
    }

    if (btnEdit) {
        btnEdit.addEventListener('click', () => {
            editForm.classList.add('editing');
            toolbar.style.display = 'flex';
            if (hint) hint.style.display = 'inline';
            btnEdit.style.display = 'none';
            btnSave.style.display = 'inline-block';
            btnCancel.style.display = 'inline-block';

            editForm.querySelectorAll('.report-row').forEach(tr => {
                // Снимок исходных значений
                snapshotRow(tr);

                // Разблокируем поля
                tr.querySelectorAll('.editable-field, input[type="hidden"]').forEach(el => el.disabled = false);

                // Слушатели на изменения
                tr.querySelectorAll('.editable-field').forEach(el => {
                    el.addEventListener('input', () => updateRowState(tr));
                    el.addEventListener('change', () => updateRowState(tr));
                });

                // Клик по строке для выделения
                tr.addEventListener('click', handleRowClick);
            });

            updateSaveButtonLabel();
        });

        btnCancel.addEventListener('click', () => window.location.reload());
    }

    // Выделение строк
    let lastSelectedIndex = null;

    function handleRowClick(e) {
        if (e.target.tagName === 'SELECT' || e.target.tagName === 'INPUT' || e.target.tagName === 'A') return;

        const rows = Array.from(editForm.querySelectorAll('.report-row'));
        const currentIndex = rows.indexOf(this);

        if (!e.ctrlKey && !e.shiftKey) {
            rows.forEach(r => r.classList.remove('row-selected'));
            this.classList.add('row-selected');
            lastSelectedIndex = currentIndex;
        } else if (e.ctrlKey) {
            this.classList.toggle('row-selected');
            lastSelectedIndex = currentIndex;
        } else if (e.shiftKey && lastSelectedIndex !== null) {
            const start = Math.min(lastSelectedIndex, currentIndex);
            const end = Math.max(lastSelectedIndex, currentIndex);
            rows.forEach((r, idx) => {
                if (idx >= start && idx <= end) r.classList.add('row-selected');
            });
        }
    }

    // Массовое заполнение
    const btnBulkApply = document.getElementById('btn-bulk-apply');
    if (btnBulkApply) {
        btnBulkApply.addEventListener('click', () => {
            const field = document.getElementById('bulk-field').value;
            const val = document.getElementById('bulk-value').value;
            const selectedRows = editForm.querySelectorAll('.report-row.row-selected');

            if (selectedRows.length === 0) {
                alert('Выделите хотя бы одну строку (клик по строке, Shift/Ctrl для нескольких).');
                return;
            }

            selectedRows.forEach(tr => {
                if (field === 'work_kind') {
                    const sel = tr.querySelector('select[name="work_kind[]"]');
                    if (sel) sel.value = val;
                } else if (field === 'prj_stage') {
                    const inp = tr.querySelector('input[name="prj_stage[]"]');
                    if (inp) inp.value = val;
                } else if (field === 'prj_id') {
                    const inp = tr.querySelector('input[name="prj_id[]"]');
                    if (inp) inp.value = val;
                }
                updateRowState(tr);
            });
        });
    }

    // Перед отправкой формы — отключаем неизменённые строки
    if (btnSave) {
        btnSave.addEventListener('click', (e) => {
            e.preventDefault();
            const changed = editForm.querySelectorAll('.row-changed').length;
            if (changed === 0) {
                alert('Нет изменений для сохранения.');
                return;
            }
            editForm.querySelectorAll('.report-row').forEach(tr => {
                if (!tr.classList.contains('row-changed')) {
                    tr.querySelectorAll('input, select').forEach(el => el.disabled = true);
                }
            });
            // Отправляем форму
            editForm.submit();
        });
    }
});