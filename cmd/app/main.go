package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kardianos/service"

	"github.com/oleoleg/project-manager/internal/config"
	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/handlers"
	mw "github.com/oleoleg/project-manager/internal/middleware"
	"github.com/oleoleg/project-manager/internal/models"
	"github.com/oleoleg/project-manager/internal/services"
)

// ---------------------------------------------------------------
// Служба
// ---------------------------------------------------------------

type program struct {
	stopCh chan struct{}
}

func (p *program) Start(s service.Service) error {
	p.stopCh = make(chan struct{})
	go startHTTPServer(p.stopCh)
	return nil
}

func (p *program) Stop(s service.Service) error {
	log.Println("=== Project Manager stopping ===")
	if p.stopCh != nil {
		close(p.stopCh)
	}
	return nil
}

func startHTTPServer(stopCh <-chan struct{}) {
	// =====================================================
	// 1. Логирование в файл
	// =====================================================
	// baseDir — где искать .env, web/, logs/
	// 1. Рядом с exe (для службы Windows и собранного бинарника).
	// 2. В CWD — для `go run` из корня проекта, где exe лежит во временной папке.
	baseDir := detectBaseDir()
	if baseDir == "" {
		log.Fatal("cannot determine base directory: web/templates not found")
	}

	logDir := filepath.Join(baseDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("mkdir logs: %v", err)
	}

	logFile := filepath.Join(logDir, "project-manager.log")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("open log file: %v", err)
	}
	defer f.Close()

	// Пишем и в файл, и в консоль — удобно при разработке
	var out io.Writer = f
	if service.Interactive() {
		out = io.MultiWriter(f, os.Stdout)
	}
	log.SetOutput(out)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("=== Project Manager starting ===")

	// =====================================================
	// 2. Конфигурация
	// =====================================================
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.SessionSecret == "" {
		log.Fatal("SESSION_SECRET is not set in .env")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// =====================================================
	// 3. База данных
	// =====================================================
	pool, err := db.Connect(ctx, cfg)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	if err := db.BootstrapAdmin(ctx, pool, cfg); err != nil {
		log.Fatalf("bootstrap admin: %v", err)
	}

	// =====================================================
	// 4. Шаблоны
	// =====================================================
	tmpl := template.New("").Funcs(template.FuncMap{
		"deref": func(p *string) string {
			if p == nil {
				return ""
			}
			return *p
		},
		"derefInt": func(p *int64) int64 {
			if p == nil {
				return 0
			}
			return *p
		},
		"money": func(v float64) string {
			s := strconv.FormatFloat(v, 'f', 2, 64)
			return groupThousands(s) + " ₽"
		},
		"money0": func(v float64) string {
			s := strconv.FormatFloat(v, 'f', 0, 64)
			return groupThousands(s) + " ₽"
		},
		"addVAT": func(v float64, vatPercent float64) float64 {
			return v * (1 + vatPercent/100.0)
		},
		"isComplexWork": func(code string) bool {
			return code == "complex"
		},
		"workKindLabel": func(list []models.WorkKindRef, code string) string {
			for _, w := range list {
				if w.Code == code {
					return w.Name
				}
			}
			return code
		},
		"moneyOrBlur": func(u *models.User, v float64) template.HTML {
			if u != nil && u.HideFinancials {
				return template.HTML(`<span class="money-blur" title="Скрыто">•••••••</span>`)
			}
			s := strconv.FormatFloat(v, 'f', 2, 64)
			return template.HTML(groupThousands(s) + " ₽")
		},
		"vatHint": func(vatPercent float64, mode string) string {
			if vatPercent <= 0 {
				return "НДС не начисляется"
			}
			p := strconv.FormatFloat(vatPercent, 'f', -1, 64)
			if mode == "with" {
				return "в т.ч. НДС " + p + "%"
			}
			return "без НДС (НДС " + p + "%)"
		},
		"div": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"containsLock": func(locked, field string) bool {
			for _, p := range strings.Split(locked, ",") {
				if strings.TrimSpace(p) == field {
					return true
				}
			}
			return false
		},
		"statusName": func(list []models.ContractStatusRef, code int16) string {
			for _, s := range list {
				if s.Code == code {
					return s.Name
				}
			}
			return fmt.Sprintf("%d", code)
		},
		"counterpartyName": func(list []models.Counterparty, id string) string {
			for _, c := range list {
				if c.ID == id {
					return c.Name
				}
			}
			return id
		},
		"add": func(a, b float64) float64 {
			return a + b
		},
		"list": func(items ...string) []string {
			return items
		},
		"filterByType": func(lines []models.BudgetFactLine, t string) []models.BudgetFactLine {
			var res []models.BudgetFactLine
			for _, l := range lines {
				if l.OperationType == t {
					res = append(res, l)
				}
			}
			return res
		},
	})

	parsed, err := tmpl.ParseGlob(filepath.Join(baseDir, "web", "templates", "*.html"))
	if err != nil {
		log.Fatalf("templates: %v", err)
	}
	tmpl = parsed

	for _, t := range tmpl.Templates() {
		log.Printf("template loaded: %q", t.Name())
	}

	// =====================================================
	// 5. Репозитории
	// =====================================================
	userRepo := db.NewUserRepo(pool)
	counterpartyRepo := db.NewCounterpartyRepo(pool)
	contractRepo := db.NewContractRepo(pool)
	stageRepo := db.NewStageRepo(pool)
	stageCostRepo := db.NewStageCostRepo(pool)
	actRepo := db.NewActRepo(pool)
	budgetRepo := db.NewBudgetRepo(pool)

	// =====================================================
	// 6. Сервисы
	// =====================================================
	authSvc := services.NewAuthService(userRepo)
	sessMgr := services.NewSessionManager(cfg.SessionSecret)
	userSvc := services.NewUserService(userRepo)
	counterpartySvc := services.NewCounterpartyService(counterpartyRepo)
	contractSvc := services.NewContractService(contractRepo)
	stageSvc := services.NewStageService(stageRepo)
	stageCostSvc := services.NewStageCostService(stageCostRepo)
	actSvc := services.NewActService(actRepo, stageRepo)
	budgetSvc := services.NewBudgetService(budgetRepo)

	// =====================================================
	// 7. Хендлеры
	// =====================================================
	healthH := handlers.NewHealthHandler(cfg.AppEnv)
	authH := handlers.NewAuthHandler(authSvc, sessMgr, tmpl)
	dashH := handlers.NewDashboardHandler(tmpl)
	settingsH := handlers.NewSettingsHandler(userSvc, tmpl)
	counterpartyH := handlers.NewCounterpartyHandler(counterpartySvc, tmpl)
	contractH := handlers.NewContractHandler(contractSvc, counterpartySvc, stageSvc, budgetSvc, tmpl)
	stageH := handlers.NewStageHandler(stageSvc, stageCostSvc, contractSvc, counterpartySvc, actSvc, tmpl)
	adminUserH := handlers.NewAdminUserHandler(userSvc, tmpl)
	actH := handlers.NewActHandler(actSvc, stageSvc, contractSvc, tmpl)
	budgetH := handlers.NewBudgetHandler(budgetSvc, contractSvc, stageSvc, stageCostSvc, counterpartySvc, tmpl)

	// =====================================================
	// 8. Роутер
	// =====================================================
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(mw.Auth(sessMgr, authSvc))

	fs := http.FileServer(http.Dir(filepath.Join(baseDir, "web", "static")))
	r.Handle("/static/*", http.StripPrefix("/static/", fs))

	r.Get("/health", healthH.Health)
	r.Get("/login", authH.LoginPage)
	r.Post("/login", authH.LoginSubmit)
	r.Post("/logout", authH.Logout)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	})

	ownerCheck := func(ctx context.Context, contractID string) (int64, bool) {
		return contractSvc.GetResponsibleUserID(ctx, contractID)
	}

	r.Group(func(r chi.Router) {
		r.Use(mw.RequireAuth)

		r.Get("/dashboard", dashH.Index)

		r.Get("/settings", settingsH.Index)
		r.Post("/settings/theme", settingsH.SaveTheme)
		r.Post("/settings/privacy", settingsH.SavePrivacy)
		r.Post("/settings/check-path", settingsH.SaveCheckPath)

		r.Get("/counterparties", counterpartyH.List)
		r.Get("/counterparties/new", counterpartyH.New)
		r.Get("/counterparties/{id}", counterpartyH.Show)
		r.Get("/counterparties/{id}/edit", counterpartyH.Edit)

		r.Get("/contracts", contractH.List)
		r.Get("/contracts/new", contractH.New)
		r.Post("/contracts", contractH.Create)
		r.Get("/contracts/{id}", contractH.Show)
		r.Get("/contracts/{id}/edit", contractH.Edit)
		r.Get("/contracts/{id}/open-folder", contractH.OpenFolder)

		r.Get("/contracts/{id}/stages", stageH.List)
		r.Get("/contracts/{id}/stages/costs", stageH.CostsIndex)
		r.Get("/contracts/{id}/stages/new", stageH.New)
		r.Get("/contracts/{id}/stages/{guid}/edit", stageH.Edit)

		// Акты этапа (JSON API для модального окна)
		r.Get("/contracts/{id}/stages/{guid}/acts.json", actH.ListJSON)
		r.Get("/contracts/{id}/stages/{guid}/acts/sum.json", actH.SumJSON)
		r.Get("/contracts/{id}/stages/{guid}/act-sum.json", stageH.StageActSumJSON)

		// Формы создания/редактирования акта
		r.Get("/contracts/{id}/stages/{guid}/acts/new", actH.NewForm)
		r.Get("/contracts/{id}/acts/{actGuid}/edit", actH.EditForm)
		r.Get("/contracts/{id}/acts/{actGuid}/open", actH.OpenFile)

		// Приложения акта
		r.Get("/contracts/{id}/acts/{actGuid}/attachments/new", actH.NewAttachmentForm)
		r.Get("/contracts/{id}/attachments/{attGuid}/open", actH.OpenAttachment)
		r.Get("/contracts/{id}/attachments/{attGuid}/edit", actH.EditAttachmentForm)
		r.Get("/api/picked-path", actH.PickedPath)

		r.Group(func(r chi.Router) {
			r.Use(mw.RequireRole("admin", "rp_chief", "director", "rp"))
			r.Post("/counterparties", counterpartyH.Create)
			r.Post("/counterparties/{id}", counterpartyH.Update)
			r.Post("/contracts/{id}/stages/{guid}/acts", actH.Create)
			r.Post("/contracts/{id}/stages/{guid}/acts/toggle-manual", actH.ToggleStageManual)
			r.Post("/contracts/{id}/stages/{guid}/acts/set-closed", actH.SetClosed)
			r.Post("/contracts/{id}/acts/{actGuid}", actH.Update)
			r.Post("/contracts/{id}/acts/{actGuid}/delete", actH.Delete)
			r.Post("/contracts/{id}/acts/{actGuid}/attachments", actH.CreateAttachment)
			r.Post("/contracts/{id}/attachments/{attGuid}", actH.UpdateAttachment)
			r.Post("/contracts/{id}/attachments/{attGuid}/delete", actH.DeleteAttachment)

			// Бюджет (План)
			r.Get("/contracts/{id}/budget", budgetH.Index)
			r.Get("/contracts/{id}/budget/new", budgetH.NewVersion)
			r.Post("/contracts/{id}/budget/generate", budgetH.Generate)
			r.Post("/contracts/{id}/budget/settings", budgetH.SaveSettings)
			r.Post("/contracts/{id}/budget/snapshots/{snapshotID}/delete", budgetH.DeleteSnapshot)
			r.Post("/contracts/{id}/budget/snapshots/{snapshotID}/activate", budgetH.ActivateSnapshot)
			r.Get("/contracts/{id}/budget/charts.json", budgetH.GetSummaryChart)

			// Бюджет (Факт)
			r.Get("/contracts/{id}/budget/fact", budgetH.FactIndex)
			r.Post("/contracts/{id}/budget/fact/generate", budgetH.FactGenerate)
			r.Post("/contracts/{id}/budget/fact/delete", budgetH.FactDelete)
			r.Post("/contracts/{id}/budget/fact/{factID}/update", budgetH.FactUpdate)
			//r.Post("/contracts/{id}/budget/fact/save", budgetH.FactSaveBatch)
			r.Post("/contracts/{id}/budget/fact/save-form", budgetH.FactSaveForm)

			// Сравнить Бюджет Факт vs План
			r.Get("/contracts/{id}/budget/compare", budgetH.PlanFactIndex)

		})

		r.Group(func(r chi.Router) {
			r.Use(func(next http.Handler) http.Handler {
				return mw.RequireContractOwnerOrRoles(
					[]string{"admin", "rp_chief", "director"},
					ownerCheck,
				)(next)
			})
			r.Post("/contracts/{id}", contractH.Update)
			r.Post("/contracts/{id}/stages", stageH.Create)
			r.Post("/contracts/{id}/stages/{guid}", stageH.Update)
			r.Post("/contracts/{id}/stages/{guid}/move", stageH.Move)
		})

		r.Group(func(r chi.Router) {
			r.Use(mw.RequireRole("admin", "rp_chief"))
			r.Post("/contracts/{id}/delete", contractH.Delete)
			r.Post("/contracts/{id}/stages/{guid}/delete", stageH.Delete)
		})

		r.Group(func(r chi.Router) {
			r.Use(mw.RequireRole("admin", "rp_chief", "director"))
			r.Post("/contracts/{id}/budget/snapshots/{snapshotID}/approve", budgetH.ApproveSnapshot)
			r.Post("/contracts/{id}/budget/snapshots/{snapshotID}/unapprove", budgetH.UnapproveSnapshot)
		})

		r.Group(func(r chi.Router) {
			r.Use(mw.RequireRole("admin"))
			r.Get("/admin/users", adminUserH.List)
			r.Get("/admin/users/new", adminUserH.New)
			r.Get("/admin/users/{id}/edit", adminUserH.Edit)
			r.Post("/admin/users", adminUserH.Create)
			r.Post("/admin/users/{id}", adminUserH.Update)
			r.Post("/admin/users/{id}/password", adminUserH.ChangePassword)
			r.Post("/admin/users/{id}/toggle", adminUserH.ToggleActive)
			r.Post("/counterparties/{id}/delete", counterpartyH.Delete)
		})
	})

	// =====================================================
	// 9. HTTP-сервер
	// =====================================================
	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("starting server on :%s (env=%s)", cfg.HTTPPort, cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
		log.Println("signal received, shutting down...")
	case <-stopCh:
		log.Println("service stop requested, shutting down...")
	}

	shCtx, shCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shCancel()
	_ = srv.Shutdown(shCtx)
	log.Println("=== Project Manager stopped ===")
}

// detectBaseDir возвращает папку, где лежат web/ и (опционально) .env.
// Сначала проверяет рядом с exe — это путь для службы.
// Если там ничего нет — падает обратно на текущую рабочую директорию,
// что соответствует запуску через `go run` из корня проекта.
func detectBaseDir() string {
	// 1. Рядом с exe
	if exePath, err := os.Executable(); err == nil {
		candidate := filepath.Dir(exePath)
		if _, err := os.Stat(filepath.Join(candidate, "web", "templates")); err == nil {
			return candidate
		}
	}

	// 2. CWD
	if wd, err := os.Getwd(); err == nil {
		if _, err := os.Stat(filepath.Join(wd, "web", "templates")); err == nil {
			return wd
		}
	}

	return ""
}

// ---------------------------------------------------------------
// groupThousands — форматирование чисел с пробелами
// ---------------------------------------------------------------
func groupThousands(s string) string {
	intPart, fracPart := s, ""
	if i := strings.Index(s, "."); i >= 0 {
		intPart, fracPart = s[:i], s[i:]
	}
	neg := ""
	if strings.HasPrefix(intPart, "-") {
		neg = "-"
		intPart = intPart[1:]
	}

	var b strings.Builder
	for i, ch := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteRune('\u00A0')
		}
		b.WriteRune(ch)
	}
	return neg + b.String() + fracPart
}

// ---------------------------------------------------------------
// Точка входа — только служба
// ---------------------------------------------------------------

func main() {
	installFlag := flag.Bool("install", false, "установить службу")
	removeFlag := flag.Bool("remove", false, "удалить службу")
	flag.Parse()

	svcConfig := &service.Config{
		Name:        "ProjectManager",
		DisplayName: "Project Manager",
		Description: "Веб-служба управления проектами",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	if *installFlag {
		if err := installService(s); err != nil {
			log.Fatalf("install: %v", err)
		}
		fmt.Println("Служба установлена и запущена")
		return
	}

	if *removeFlag {
		_ = s.Stop()
		if err := s.Uninstall(); err != nil {
			log.Fatalf("remove: %v", err)
		}
		fmt.Println("Служба удалена")
		return
	}

	// Обычный запуск — как служба под управлением SCM
	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}

func installService(s service.Service) error {
	if err := s.Install(); err != nil {
		return fmt.Errorf("установка службы: %w", err)
	}
	if err := s.Start(); err != nil {
		return fmt.Errorf("запуск службы: %w", err)
	}
	return nil
}
