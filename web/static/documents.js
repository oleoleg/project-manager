document.addEventListener('click', function (e) {
    const btn = e.target.closest('.js-open-doc-file');
    if (!btn) return;
    const p = btn.getAttribute('data-path');
    if (!p) return;
    window.location.href = 'pm-agent://open?path=' + encodeURIComponent(p);
});

document.addEventListener('change', function (e) {
    const sel = e.target.closest('.js-stage-select');
    if (!sel) return;
    const form = sel.closest('form');
    if (!form) return;

    const wrap = form.querySelector('.js-holder-wrap');
    const input = form.querySelector('.js-holder-input');
    if (!wrap || !input) return;

    const isSigned = sel.value === 'signed';
    wrap.style.display = isSigned ? 'none' : '';
    input.required = !isSigned;
    if (isSigned) {
        input.value = '';
    }
});