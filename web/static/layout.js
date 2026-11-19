(function () {
    var t = document.documentElement.getAttribute('data-theme');
})();

document.addEventListener('DOMContentLoaded', function() {
        const btn = document.getElementById('user-menu-btn');
        const menu = document.getElementById('user-menu');
        
        if (btn && menu) {
            btn.addEventListener('click', function(e) {
                e.stopPropagation();
                menu.classList.toggle('active');
            });
            
            // Закрываем меню при клике в любом другом месте
            document.addEventListener('click', function(e) {
                if (!menu.contains(e.target) && !btn.contains(e.target)) {
                    menu.classList.remove('active');
                }
            });
        }
    });