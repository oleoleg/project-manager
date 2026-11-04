package db_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/models"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	// Пытаемся загрузить .env.test (если библиотека godotenv подключена)
	_ = godotenv.Load("../../.env.test")

	connString := os.Getenv("TEST_DATABASE_URL")
	if connString == "" {
		// Значение по умолчанию (замените на свои данные)
		connString = "postgres://postgres:postgres@localhost:5432/project_manager_test"
	}

	var err error
	testPool, err = pgxpool.New(context.Background(), connString)
	if err != nil {
		fmt.Printf("Не удалось подключиться к тестовой БД: %v\n", err)
		os.Exit(1)
	}
	defer testPool.Close()

	// Проверяем соединение
	if err := testPool.Ping(context.Background()); err != nil {
		fmt.Printf("БД не отвечает: %v\n", err)
		os.Exit(1)
	}

	// Запускаем все тесты
	code := m.Run()

	// Выходим с кодом результата
	os.Exit(code)
}

// Глобальные переменные для тестовых данных, чтобы их было удобно использовать
var (
	testContractID = "9999-01"
	testCpID       = "9999"
	testUserID     = int64(1)
)

// setupBaseData создает минимальный набор справочников и договор для тестов.
// Вызывается один раз перед тестами, которым нужен договор.
func setupBaseData(t *testing.T) {
	ctx := context.Background()

	// 1. Контрагент
	_, err := testPool.Exec(ctx, `
		INSERT INTO counterparties (id, name) 
		VALUES ($1, 'Тестовый контрагент') 
		ON CONFLICT (id) DO NOTHING
	`, testCpID)
	if err != nil {
		t.Fatalf("setup counterparty: %v", err)
	}

	// 2. Пользователь
	_, err = testPool.Exec(ctx, `
		INSERT INTO users (id, username, full_name, password_hash, role_code, is_active) 
		VALUES ($1, 'testuser', 'Тестовый Юзер', 'hash', 'admin', true) 
		ON CONFLICT (id) DO NOTHING
	`, testUserID)
	if err != nil {
		t.Fatalf("setup user: %v", err)
	}

	// 3. Договор
	_, err = testPool.Exec(ctx, `
		INSERT INTO contracts (id, contract_number, contract_date, status_code, counterparty_id, contract_type) 
		VALUES ($1, 'TEST-001', NOW(), 1, $2, 'contract') 
		ON CONFLICT (id) DO NOTHING
	`, testContractID, testCpID)
	if err != nil {
		t.Fatalf("setup contract: %v", err)
	}
}

func TestBudget(t *testing.T) {
	// Вся эта группа тестов требует базового договора
	t.Run("SnapshotAndFact", func(t *testing.T) {
		clearTestDB(t)
		setupBaseData(t)

		// Здесь вызываем наши тесты, но они теперь не создают договор сами,
		// а используют testContractID, testUserID
		t.Run("CreateSnapshot", func(t *testing.T) {
			testCreateSnapshot(t)
		})
		t.Run("RelinkFact", func(t *testing.T) {
			testRelinkFact(t)
		})
	})
}

func testCreateSnapshot(t *testing.T) {
	ctx := context.Background()
	repo := db.NewBudgetRepo(testPool)

	settings := models.BudgetSettings{
		ContractID:       testContractID,
		ProjectStartDate: time.Now(),
		OverheadRate:     1.2,
	}

	lines := []models.BudgetPlanLine{
		{
			ContractID:    testContractID,
			StageNumber:   "1",
			StageName:     "Тестовый этап",
			WorkKind:      "ПИР",
			Name:          "Оплата по договору. Тестовый этап",
			OperationDate: time.Now(),
			Income:        1000.00,
			Expense:       0,
			OperationType: "Платежи по договору",
		},
	}

	_, err := repo.CreateSnapshotWithLines(ctx, testContractID, testUserID, "Тестовый снимок", settings, lines, nil, nil)
	if err != nil {
		t.Fatalf("ошибка при создании снимка: %v", err)
	}

	var count int
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM budget_plan_lines WHERE contract_id = $1`, testContractID).Scan(&count)
	if err != nil {
		t.Fatalf("ошибка при проверке: %v", err)
	}

	if count != 1 {
		t.Errorf("ожидалась 1 строка плана, получено %d", count)
	}

	var snapCount int
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM budget_plan_snapshots WHERE contract_id = $1`, testContractID).Scan(&snapCount)
	if snapCount != 1 {
		t.Errorf("ожидался 1 снимок, получено %d", snapCount)
	}
}

func testRelinkFact(t *testing.T) {
	ctx := context.Background()
	repo := db.NewBudgetRepo(testPool)

	settings := models.BudgetSettings{
		ContractID:       testContractID,
		ProjectStartDate: time.Now(),
		OverheadRate:     1.2,
	}

	lines1 := []models.BudgetPlanLine{
		{
			ContractID:    testContractID,
			StageNumber:   "1",
			StageName:     "Этап 1",
			WorkKind:      "ПИР",
			Name:          "Зарплата. Этап 1",
			OperationDate: time.Now(),
			Expense:       1000,
			OperationType: db.OpTypeSalary,
		},
	}
	_, err := repo.CreateSnapshotWithLines(ctx, testContractID, testUserID, "План 1", settings, lines1, nil, nil)
	if err != nil {
		t.Fatalf("ошибка создания плана 1: %v", err)
	}

	// Имитируем, что пользователь сгенерировал факт
	_, err = testPool.Exec(ctx, `
		INSERT INTO budget_fact_lines (
			contract_id, stage_number, stage_name, work_kind, name, operation_date, payment_date, operation_type, fact_expense
		)
		VALUES ($1, '1', 'Этап 1', 'ПИР', 'Зарплата. Этап 1', NOW(), NOW(), $2, 500.00)
	`, testContractID, db.OpTypeSalary)
	if err != nil {
		t.Fatalf("ошибка создания факта: %v", err)
	}

	// Привязываем факт к плану 1
	_, _ = testPool.Exec(ctx, `
		UPDATE budget_fact_lines f
		SET plan_line_id = p.id
		FROM budget_plan_lines p
		WHERE f.contract_id = $1 AND p.contract_id = $1 AND p.name = f.name
	`, testContractID)

	var planLineID *string
	_ = testPool.QueryRow(ctx, `SELECT plan_line_id FROM budget_fact_lines WHERE contract_id = $1`, testContractID).Scan(&planLineID)
	if planLineID == nil {
		t.Fatal("факт не привязался к первому плану")
	}

	// Создаем ВТОРОЙ план
	lines2 := []models.BudgetPlanLine{
		{
			ContractID:    testContractID,
			StageNumber:   "1",
			StageName:     "Этап 1",
			WorkKind:      "ПИР",
			Name:          "Зарплата. Этап 1",
			OperationDate: time.Now(),
			Expense:       1000,
			OperationType: db.OpTypeSalary,
		},
	}
	_, err = repo.CreateSnapshotWithLines(ctx, testContractID, testUserID, "План 2", settings, lines2, nil, nil)
	if err != nil {
		t.Fatalf("ошибка создания плана 2: %v", err)
	}

	// Проверяем, что факт НЕ УДАЛИЛСЯ
	var factCount int
	_ = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM budget_fact_lines WHERE contract_id = $1`, testContractID).Scan(&factCount)
	if factCount != 1 {
		t.Fatalf("ожидалась 1 строка факта, получено %d. Факт удален!", factCount)
	}

	// Проверяем, что факт перепривязался к НОВОМУ плану
	var newPlanLineID *string
	_ = testPool.QueryRow(ctx, `SELECT plan_line_id FROM budget_fact_lines WHERE contract_id = $1`, testContractID).Scan(&newPlanLineID)
	if newPlanLineID == nil {
		t.Error("факт не привязался ко второму плану (plan_line_id = NULL)")
	} else if *newPlanLineID == *planLineID {
		t.Error("факт остался привязанным к СТАРОМУ плану, перепривязка не сработала")
	}
}
