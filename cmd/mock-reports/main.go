package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

func main() {
	http.HandleFunc("/reports/api/database/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"id": 1, "username": "ivanov", "longname": "Иванов И.И.", "city": "Москва", "email": "ivanov@test.ru", "is_admin": false, "is_blocked": false, "department": "Отдел ПО и ПНР"},
				{"id": 2, "username": "petrov", "longname": "Петров П.П.", "city": "СПб", "email": "petrov@test.ru", "is_admin": false, "is_blocked": false, "department": "Проектный отдел"},
				{"id": 3, "username": "sidorov", "longname": "Сидоров С.С.", "city": "Казань", "email": "sidorov@test.ru", "is_admin": false, "is_blocked": false, "department": "Монтажный отдел"},
			},
		})
	})

	http.HandleFunc("/reports/api/database/worktypes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"id": 0, "name": "Офис", "calculate_hours": true},
				{"id": 1, "name": "Командировка", "calculate_hours": true},
				{"id": 4, "name": "Удаленная работа", "calculate_hours": true},
				{"id": 10, "name": "Отпуск", "calculate_hours": false},
			},
		})
	})

	http.HandleFunc("/reports/api/database/projects", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"prj_id": "6666", "description": "Договор 6666"},
				{"prj_id": "7777", "description": "Договор 7777"},
			},
		})
	})

	http.HandleFunc("/reports/api/database/reports", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Простая пагинация
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit == 0 {
			limit = 10
		}

		// Генерируем фейковые данные
		var items []map[string]any
		now := time.Now()

		for i := 0; i < 9; i++ {
			start := now.AddDate(0, 0, -i).Truncate(24 * time.Hour).Add(9 * time.Hour)
			end := start.Add(9 * time.Hour) // 9 часов работы

			// Чередуем проекты 6666 и 7777
			prjID := "6666"
			if i%3 == 0 {
				prjID = "7777"
			}
			if i%4 == 0 {
				prjID = "8888"
			}

			items = append(items, map[string]any{
				"id":          1000 + offset + i,
				"userid":      (i % 3) + 1,
				"prj_id":      prjID,
				"work_type":   0,
				"start_time":  start.Format("2006-01-02T15:04:05"),
				"end_time":    end.Format("2006-01-02T15:04:05"),
				"description": "Тестовая запись " + strconv.Itoa(i+1),
				"timestamp":   now.Format("2006-01-02T15:04:05"),
			})

		}

		json.NewEncoder(w).Encode(map[string]any{
			"items":  items,
			"total":  5,
			"limit":  limit,
			"offset": offset,
		})
	})

	log.Println("Mock Reports API starting on :8001...")
	log.Fatal(http.ListenAndServe(":8001", nil))
}
