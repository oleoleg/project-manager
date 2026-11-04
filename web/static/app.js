

// Сворачивание секций (используется на contracts_show.html).
window.toggleSection = function (id) {
    const sec = document.getElementById(id);
    if (!sec) return;
    sec.classList.toggle("collapsed");
    const btn = sec.querySelector(".toggle-btn");
    if (btn) {
        const collapsed = sec.classList.contains("collapsed");
        btn.textContent = collapsed ? "Развернуть" : "Свернуть";
        btn.setAttribute("aria-expanded", String(!collapsed));
    }
};