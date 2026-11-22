document.addEventListener('click', function (e) {
    const btn = e.target.closest('.js-open-doc-file');
    if (!btn) return;
    const p = btn.getAttribute('data-path');
    if (!p) return;
    window.location.href = 'pm-agent://open?path=' + encodeURIComponent(p);
});
