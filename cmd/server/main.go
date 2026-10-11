package main

import (
	"context"
	"html/template"
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

	"github.com/oleoleg/project-manager/internal/config"
	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/handlers"
	mw "github.com/oleoleg/project-manager/internal/middleware"
	"github.com/oleoleg/project-manager/internal/services"
)

func main() {
	// =====================================================
	// 1. Конфигурация
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
	// 2. База данных: подключение, миграции, bootstrap админа
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
	// 3. Шаблоны (загружаются один раз при старте)
	// =====================================================
	tmpl := template.New("").Funcs(template.FuncMap{
		// deref разыменовывает *string, nil -> ""
		"deref": func(p *string) string {
			if p == nil {
				return ""
			}
			return *p
		},
		// money форматирует сумму: 1234567.5 -> "1 234 567.50 ₽"
		"money": func(v float64) string {
			s := strconv.FormatFloat(v, 'f', 2, 64)
			return groupThousands(s) + " ₽"
		},
		// money0 — то же, но без копеек
		"money0": func(v float64) string {
			s := strconv.FormatFloat(v, 'f', 0, 64)
			return groupThousands(s) + " ₽"
		},
	})

	parsed, err := tmpl.ParseGlob(filepath.Join("web", "templates", "*.html"))
	if err != nil {
		log.Fatalf("templates: %v", err)
	}
	tmpl = parsed

	// отладочный вывод — какие шаблоны загрузились
	for _, t := range tmpl.Templates() {
		log.Printf("template loaded: %q", t.Name())
	}

	// =====================================================
	// 4. Репозитории (доступ к БД)
	// =====================================================
	userRepo := db.NewUserRepo(pool)
	counterpartyRepo := db.NewCounterpartyRepo(pool)
	contractRepo := db.NewContractRepo(pool)
	stageRepo := db.NewStageRepo(pool)

	// =====================================================
	// 5. Сервисы (бизнес-логика)
	// =====================================================
	authSvc := services.NewAuthService(userRepo)
	sessMgr := services.NewSessionManager(cfg.SessionSecret)
	userSvc := services.NewUserService(userRepo)
	counterpartySvc := services.NewCounterpartyService(counterpartyRepo)
	contractSvc := services.NewContractService(contractRepo)
	stageSvc := services.NewStageService(stageRepo)

	// =====================================================
	// 6. HTTP-хендлеры
	// =====================================================
	healthH := handlers.NewHealthHandler(cfg.AppEnv)
	authH := handlers.NewAuthHandler(authSvc, sessMgr, tmpl)
	dashH := handlers.NewDashboardHandler(tmpl)
	settingsH := handlers.NewSettingsHandler(userSvc, tmpl)
	counterpartyH := handlers.NewCounterpartyHandler(counterpartySvc, tmpl)
	contractH := handlers.NewContractHandler(contractSvc, counterpartySvc, stageSvc, tmpl)
	stageH := handlers.NewStageHandler(stageSvc, contractSvc, counterpartySvc, tmpl)

	// =====================================================
	// 7. Роутер и middleware
	// =====================================================
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(mw.Auth(sessMgr, authSvc)) // кладёт текущего пользователя в контекст

	// --- Статика ---
	fs := http.FileServer(http.Dir("./web/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fs))

	// --- Публичные роуты ---
	r.Get("/health", healthH.Health)
	r.Get("/login", authH.LoginPage)
	r.Post("/login", authH.LoginSubmit)
	r.Post("/logout", authH.Logout)

	// корень — редирект на дашборд (или на логин, если не авторизован)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	})

	// =====================================================
	// 8. Защищённые роуты (требуется вход)
	// =====================================================
	r.Group(func(r chi.Router) {
		r.Use(mw.RequireAuth)

		// --- Дашборд ---
		r.Get("/dashboard", dashH.Index)

		// --- Настройки пользователя ---
		r.Get("/settings", settingsH.Index)
		r.Post("/settings/theme", settingsH.SaveTheme)

		// --- Контрагенты: просмотр доступен всем ролям ---
		r.Get("/counterparties", counterpartyH.List)
		r.Get("/counterparties/new", counterpartyH.New)
		r.Get("/counterparties/{id}", counterpartyH.Show)
		r.Get("/counterparties/{id}/edit", counterpartyH.Edit)

		// --- Договоры: просмотр ---
		r.Get("/contracts", contractH.List)
		r.Get("/contracts/new", contractH.New)
		r.Get("/contracts/{id}", contractH.Show)
		r.Get("/contracts/{id}/edit", contractH.Edit)

		// --- Этапы договора: просмотр ---
		r.Get("/contracts/{id}/stages", stageH.List)
		r.Get("/contracts/{id}/stages/new", stageH.New)
		r.Get("/contracts/{id}/stages/{guid}/edit", stageH.Edit)

		// --- Создание / редактирование (admin, rp_chief, rp) ---
		r.Group(func(r chi.Router) {
			r.Use(mw.RequireRole("admin", "rp_chief", "rp"))

			r.Post("/counterparties", counterpartyH.Create)
			r.Post("/counterparties/{id}", counterpartyH.Update)

			r.Post("/contracts", contractH.Create)
			r.Post("/contracts/{id}", contractH.Update)

			r.Post("/contracts/{id}/stages", stageH.Create)
			r.Post("/contracts/{id}/stages/{guid}", stageH.Update)
			r.Post("/contracts/{id}/stages/{guid}/move", stageH.Move)
		})

		// --- Удаление (только admin и rp_chief) ---
		r.Group(func(r chi.Router) {
			r.Use(mw.RequireRole("admin", "rp_chief"))

			r.Post("/counterparties/{id}/delete", counterpartyH.Delete)
			r.Post("/contracts/{id}/delete", contractH.Delete)
			r.Post("/contracts/{id}/stages/{guid}/delete", stageH.Delete)
		})
	})

	// =====================================================
	// 9. Зарезервировано под админские роуты (Шаг N)
	// =====================================================
	r.Group(func(r chi.Router) {
		r.Use(mw.RequireAuth)
		r.Use(mw.RequireRole("admin"))
		// r.Get("/admin/users", ...) — появится на шаге с управлением пользователями
	})

	// =====================================================
	// 10. HTTP-сервер и graceful shutdown
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

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	shCtx, shCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shCancel()
	_ = srv.Shutdown(shCtx)
}

// groupThousands вставляет неразрывные пробелы каждые 3 разряда в целой части.
// Пример: "1000000.00" -> "1 000 000.00"
func groupThousands(s string) string {
	intPart, fracPart := s, ""
	if i := strings.Index(s, "."); i >= 0 {
		intPart, fracPart = s[:i], s[i:] // fracPart вместе с точкой
	}
	neg := ""
	if strings.HasPrefix(intPart, "-") {
		neg = "-"
		intPart = intPart[1:]
	}

	var b strings.Builder
	for i, ch := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteRune('\u00A0') // неразрывный пробел
		}
		b.WriteRune(ch)
	}
	return neg + b.String() + fracPart
}
