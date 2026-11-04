// web/static/formats.js

// 1. Форматирование для ОТОБРАЖЕНИЯ (с значком валюты)
// Используется в итогах, таблицах и т.д. (например: "1 000.00 ₽")
function formatMoney(val) {
    if (!val && val !== 0) return '';
    const num = typeof val === 'number' ? val : parseMoneyText(val);
    const s = Math.abs(num).toFixed(2);
    const formatted = s.replace(/\B(?=(\d{3})+(?!\d))/g, '\u00A0');
    return (num < 0 ? '-' : '') + formatted + ' ₽';
}

// 2. Парсинг строки в число (для расчетов)
// Например, "1 234,56 ₽" превратит в 1234.56
function parseMoneyText(text) {
    if (text === null || text === undefined) return 0;
    if (typeof text === 'number') return text;
    const clean = String(text)
        .replace(/\u00A0/g, '') // убираем неразрывные пробелы
        .replace(/\s/g, '')     // убираем обычные пробелы
        .replace(',', '.')      // заменяем запятую на точку
        .replace(/[^\d.-]/g, ''); // оставляем только цифры, точку и минус
    const val = parseFloat(clean);
    return isNaN(val) ? 0 : val;
}

// 3. Автоформат денежных полей при вводе: пробелы между разрядами целой части.
// Класс .money-input помечаем на нужных input-полях.
function inputMoney(raw) {
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


function attachMoney(el) {
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

    el.addEventListener("input", function (e) {
        const it = e.inputType || "";
        if (it === "deleteContentBackward" || it === "deleteContentForward") {
            return;
        }
        const before = el.value;
        const after = inputMoney(before);
        if (before === after) return;

        const endOffset = before.length - el.selectionStart;
        el.value = after;
        const pos = Math.max(0, after.length - endOffset);
        el.setSelectionRange(pos, pos);
    });

    el.addEventListener("blur", function () {
        if (!el.value) return;
        let v = el.value.replace(/\s/g, "");
        if (v.indexOf(",") >= 0) v = v.replace(",", ".");
        if (/^-?\d+$/.test(v)) v = v + ".00";
        if (/^-?\d+\.\d$/.test(v)) v = v + "0";
        el.value = inputMoney(v);
    });
}

//Формат дней
function inputDays(raw) {
    let s = String(raw || "").trim();
    return s.replace(/\D/g, "");
}

function attachDays(el) {
    el.addEventListener("input", function () {
        let val = inputDays(this.value);
        if (val !== this.value) {
            this.value = val;
        }
    });

    el.addEventListener("blur", function () {
        if (!this.value) return;
        this.value = inputDays(this.value);
    });
}


window.attachMoneyInput = attachMoney;
window.inputMoney = inputMoney;
window.inputDays =  inputDays;


document.addEventListener("DOMContentLoaded", function () {
    // Инициализация денег
    document.querySelectorAll("input.money-input").forEach(function (el) {
        if (el.value) el.value = inputMoney(el.value);
        attachMoney(el);
    });

    // Инициализация дней
    document.querySelectorAll("input.days-input").forEach(function (el) {
        if (el.value) el.value = inputDays(el.value);
        attachDays(el);
    });
});
