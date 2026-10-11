// Автоформат денежных полей: пробелы между разрядами целой части.
// Класс .money-input помечаем на нужных input-полях.
(function () {
    function formatMoney(raw) {
        let neg = false;
        let s = String(raw || "").trim();
        if (s.startsWith("-")) {
            neg = true;
            s = s.slice(1);
        }
        // отделим дробную часть (первый разделитель — точка или запятая)
        let sepIdx = -1;
        for (let i = 0; i < s.length; i++) {
            const ch = s[i];
            if (ch === "." || ch === ",") { sepIdx = i; break; }
        }
        let intPart = sepIdx >= 0 ? s.slice(0, sepIdx) : s;
        let fracPart = sepIdx >= 0 ? s.slice(sepIdx + 1) : "";

        intPart = intPart.replace(/\D/g, "");
        fracPart = fracPart.replace(/\D/g, "").slice(0, 2);

        // группируем целую часть пробелами
        let grouped = "";
        for (let i = 0; i < intPart.length; i++) {
            if (i > 0 && (intPart.length - i) % 3 === 0) grouped += " ";
            grouped += intPart[i];
        }
        if (grouped === "") grouped = "0";

        let out = (neg ? "-" : "") + grouped;
        if (sepIdx >= 0) out += "." + fracPart;
        return out;
    }

    function attach(el) {
        // Запоминаем исходное значение при фокусе — для Esc
        el.addEventListener("focus", function () {
            el.dataset.originalValue = el.value;
        });

        el.addEventListener("keydown", function (e) {
            if (e.key === "Escape") {
                if (el.dataset.originalValue !== undefined) {
                    el.value = el.dataset.originalValue;
                }
                e.preventDefault();
                el.blur();
            } else if (e.key === "Enter") {
                e.preventDefault();
                el.dispatchEvent(new Event("change", { bubbles: true }));
                el.blur();
            }
        });

        // На input — форматируем, НО пропускаем события удаления (Backspace/Delete),
        // чтобы пользователь мог удалить цифру сразу после пробела.
        el.addEventListener("input", function (e) {
            const it = e.inputType || "";
            if (it === "deleteContentBackward" || it === "deleteContentForward") {
                return;
            }

            const before = el.value;
            const after = formatMoney(before);
            if (before === after) return;

            // Позиция курсора от конца строки — так надёжнее при вставке/удалении
            const endOffset = before.length - el.selectionStart;
            el.value = after;
            const pos = Math.max(0, after.length - endOffset);
            el.setSelectionRange(pos, pos);
        });

        // На blur — нормализуем до 2 знаков после точки и докручиваем пробелы
        el.addEventListener("blur", function () {
            if (!el.value) return;
            let v = el.value.replace(/\s/g, "");
            if (v.indexOf(",") >= 0) v = v.replace(",", ".");
            if (/^-?\d+$/.test(v)) v = v + ".00";
            if (/^-?\d+\.\d$/.test(v)) v = v + "0";
            el.value = formatMoney(v);
        });
    }

    // Экспортируем, чтобы можно было навешивать на новые строки
    window.attachMoneyInput = attach;
    window.formatMoney = formatMoney;

    document.addEventListener("DOMContentLoaded", function () {
        document.querySelectorAll("input.money-input").forEach(function (el) {
            if (el.value) el.value = formatMoney(el.value);
            attach(el);
        });
    });
})();


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