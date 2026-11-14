// web/static/app.js

document.addEventListener('DOMContentLoaded', function() {
    // Плавное исчезновение Flash-плашки (глобальной)
    const flashMessage = document.querySelector('.flash');
    
    if (flashMessage) {
        // Функция для плавного скрытия
        const hideFlash = () => {
            flashMessage.style.opacity = '0';
            setTimeout(() => flashMessage.remove(), 500); // Удаляем через полсекунды (после анимации)
        };

        // Исчезает при клике в любом месте страницы
        document.addEventListener('click', hideFlash, { once: true });
        
        // Или автоматически через 4 секунды
        setTimeout(hideFlash, 4000);
    }
});

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