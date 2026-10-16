// Словарь сопоставления ключевых слов в имени файла виду акта.
// Используется для автоподстановки «Вид акта / приложения» при выборе файла.
(function () {
    const KIND_MAP = [
        // Проверяем по порядку. Первое совпадение выигрывает.
         // КС-2 / КС-3 — форма с цифрой, специфичнее «акт»
        { kind: 'ks2',              patterns: ['кс-2', 'кс2', 'кс 2'] },
        { kind: 'ks3',              patterns: ['кс-3', 'кс3', 'кс 3'] },

        // УПД
        { kind: 'upd',              patterns: ['упд'] },

        // Счёт-фактура — точнее, чем «счёт» и «сф»
        { kind: 'invoice_fact',     patterns: ['сф', 'с-ф', 'с_ф', 'sf', 'счет-фактура', 'счёт-фактура', 'счет фактура', 'счёт фактура'] },
     
        // Платёжное поручени
        { kind: 'payment_order',    patterns: ['пп', 'pp', 'платежное', 'платёжное'] },


         // === Затем общие ===
        { kind: 'acceptance',       patterns: ['акт', 'прием', 'приём'] },
        { kind: 'invoice',          patterns: ['счет', 'счёт', 'сч №', 'сч№'] },
    ];

    // detectActKind(filename) → код вида акта или '' если не распознано.
    window.detectActKind = function (filename) {
        if (!filename) return '';
        const lower = filename.toLowerCase();

        for (const entry of KIND_MAP) {
            for (const p of entry.patterns) {
                if (lower.indexOf(p) !== -1) {
                    return entry.kind;
                }
            }
        }
        return '';
    };
})();