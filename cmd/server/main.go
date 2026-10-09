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

	// Репозитории и сервисы
	userRepo := db.NewUserRepo(pool)
	authSvc := services.NewAuthService(userRepo)
	sessMgr := services.NewSessionManager(cfg.SessionSecret)

	// Шаблоны — общий набор
	tmpl, err := template.ParseGlob(filepath.Join("web", "templates", "*.html"))
	if err != nil {
		log.Fatalf("templates: %v", err)
	}

	// Хендлеры
	healthH := handlers.NewHealthHandler(cfg.AppEnv)
	authH := handlers.NewAuthHandler(authSvc, sessMgr, tmpl)
	dashH := handlers.NewDashboardHandler(tmpl)

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
