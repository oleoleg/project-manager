package services

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var ErrMoneyFormat = errors.New("денежное значение в неверном формате")

// Разрешено:
//
//	1 000 000
//	1000000.50
//	1 234,5
//	0
//	-5000
//
// Пробелы (обычные и неразрывные) убираем ПЕРЕД валидацией.
// Регулярка проверяет уже «чистое» число.
var moneyRe = regexp.MustCompile(`^-?\d+(?:[.,]\d{1,2})?$`)

// ParseMoney принимает строку от пользователя и возвращает float64.
// Возвращает ErrMoneyFormat, если формат неверный.
func ParseMoney(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	// Убираем пробелы: обычный, неразрывный, таб.
	clean := strings.NewReplacer(
		" ", "",
		"\u00A0", "", // неразрывный пробел
		"\t", "",
	).Replace(s)

	// запятая -> точка (только первая, чтобы не портить "1,2,3")
	clean = strings.Replace(clean, ",", ".", 1)

	if !moneyRe.MatchString(clean) {
		return 0, ErrMoneyFormat
	}

	v, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0, ErrMoneyFormat
	}
	return v, nil
}
