package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.SessionSecret == "" {
		log.Fatal("SESSION_SECRET is not set in .env")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.Connect(ctx, cfg)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	if err := db.BootstrapAdmin(ctx, pool, cfg); err != nil {
		log.Fatalf("bootstrap admin: %v", err)
	}

	counterpartyRepo := db.NewCounterpartyRepo(pool)
	contractRepo := db.NewContractRepo(pool)

	// Репозитории и сервисы
	userRepo := db.NewUserRepo(pool)
	authSvc := services.NewAuthService(userRepo)
	sessMgr := services.NewSessionManager(cfg.SessionSecret)
	counterpartySvc := services.NewCounterpartyService(counterpartyRepo)
	contractSvc := services.NewContractService(contractRepo)

	// Шаблоны — общий набор
	tmpl := template.New("").Funcs(template.FuncMap{
		"deref": func(p *string) string {
			if p == nil {
				return ""
			}
			return *p
		},
	})
	tmpl, err = tmpl.ParseGlob(filepath.Join("web", "templates", "*.html"))
	if err != nil {
		log.Fatalf("templates: %v", err)
	}
	for _, t := range tmpl.Templates() {
		log.Printf("template loaded: %q", t.Name())
	}

	// Хендлеры
	healthH := handlers.NewHealthHandler(cfg.AppEnv)
	authH := handlers.NewAuthHandler(authSvc, sessMgr, tmpl)
	dashH := handlers.NewDashboardHandler(tmpl)
	counterpartyH := handlers.NewCounterpartyHandler(counterpartySvc, tmpl)
	contractH := handlers.NewContractHandler(contractSvc, counterpartySvc, tmpl)

	// Роутер
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(mw.Auth(sessMgr, authSvc))

	// статика
	fs := http.FileServer(http.Dir("./web/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fs))

	// публичные
	r.Get("/health", healthH.Health)
	r.Get("/login", authH.LoginPage)
	r.Post("/login", authH.LoginSubmit)
	r.Post("/logout", authH.Logout)

	// корень
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	})

	// защищённые
	r.Group(func(r chi.Router) {
		r.Use(mw.RequireAuth)
		r.Get("/dashboard", dashH.Index)

		// Контрагенты: просмотр — всем
		r.Get("/counterparties", counterpartyH.List)
		r.Get("/counterparties/new", counterpartyH.New)
		r.Get("/counterparties/{id}", counterpartyH.Show)
		r.Get("/counterparties/{id}/edit", counterpartyH.Edit)

		// Договоры — GET-ы
		r.Get("/contracts", contractH.List)
		r.Get("/contracts/new", contractH.New)
		r.Get("/contracts/{id}", contractH.Show)
		r.Get("/contracts/{id}/edit", contractH.Edit)

		// Создание/редактирование
		r.Group(func(r chi.Router) {
			r.Use(mw.RequireRole("admin", "rp_chief", "rp"))
			r.Post("/counterparties", counterpartyH.Create)
			r.Post("/counterparties/{id}", counterpartyH.Update)
			r.Post("/contracts", contractH.Create)
			r.Post("/contracts/{id}", contractH.Update)
		})

		// Удаление
		r.Group(func(r chi.Router) {
			r.Use(mw.RequireRole("admin", "rp_chief"))
			r.Post("/counterparties/{id}/delete", counterpartyH.Delete)
			r.Post("/contracts/{id}/delete", contractH.Delete)
		})
	})

	// Пример роута только для админа (пригодится в шаге 3)
	r.Group(func(r chi.Router) {
		r.Use(mw.RequireAuth)
		r.Use(mw.RequireRole("admin"))
		// r.Get("/admin/users", ...)
	})

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
