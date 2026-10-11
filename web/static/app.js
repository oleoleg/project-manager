// Автоформат денежных полей: пробелы между разрядами целой части.
// Класс .money-input помечаем на нужных input-полях.
(function () {
    function formatMoney(raw) {
        // Оставляем цифры, минус (в начале), точку или запятую (один разделитель)
        let neg = false;
        let s = String(raw || "").trim();
        if (s.startsWith("-")) {
            neg = true;
            s = s.slice(1);
        }
        // отделим дробную часть
        let sepIdx = -1;
        for (let i = 0; i < s.length; i++) {
            const ch = s[i];
            if (ch === "." || ch === ",") { sepIdx = i; break; }
        }
        let intPart = sepIdx >= 0 ? s.slice(0, sepIdx) : s;
        let fracPart = sepIdx >= 0 ? s.slice(sepIdx + 1) : "";

        intPart = intPart.replace(/\D/g, "");        // только цифры
        fracPart = fracPart.replace(/\D/g, "").slice(0, 2); // 0..2 цифры

        // группируем целую часть
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
        el.addEventListener("input", function () {
            const before = el.value;
            const after = formatMoney(before);
            if (before !== after) {
                // сохраняем позицию каретки относительно конца строки
                const endOffset = before.length - el.selectionStart;
                el.value = after;
                const pos = Math.max(0, after.length - endOffset);
                el.setSelectionRange(pos, pos);
            }
        });
        // при потере фокуса — нормализуем: 2 знака после точки, если что-то введено
        el.addEventListener("blur", function () {
            if (!el.value) return;
            let v = el.value.replace(/\s/g, "");
            if (v.indexOf(",") >= 0) v = v.replace(",", ".");
            if (/^-?\d+$/.test(v)) v = v + ".00";
            if (/^-?\d+\.\d$/.test(v)) v = v + "0";
            el.value = formatMoney(v);
        });
    }

    document.addEventListener("DOMContentLoaded", function () {
        document.querySelectorAll("input.money-input").forEach(attach);
    });
})();


// Сворачивание секций.
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


