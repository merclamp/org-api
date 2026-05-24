package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/merclamp/org-api/internal/config"
	"github.com/merclamp/org-api/internal/handler"
	"github.com/merclamp/org-api/internal/repository"
	"github.com/merclamp/org-api/internal/service"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := connectDB(cfg, log)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	if err := runMigrations(db, log); err != nil {
		log.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	deptRepo := repository.NewDepartmentRepository(db)
	empRepo  := repository.NewEmployeeRepository(db)

	deptSvc := service.NewDepartmentService(deptRepo, empRepo)
	empSvc  := service.NewEmployeeService(empRepo, deptRepo)

	deptHandler := handler.NewDepartmentHandler(deptSvc)
	empHandler  := handler.NewEmployeeHandler(empSvc)

	router := handler.NewRouter(deptHandler, empHandler, log)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("server started", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	log.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("forced shutdown", "error", err)
	}

	log.Info("server stopped")
}

func connectDB(cfg *config.Config, log *slog.Logger) (*gorm.DB, error) {
	const maxRetries = 10
	const retryInterval = 3 * time.Second

	gormCfg := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	}

	var (
		db  *gorm.DB
		err error
	)

	for i := range maxRetries {
		db, err = gorm.Open(postgres.Open(cfg.DB.DSN()), gormCfg)
		if err == nil {
			sqlDB, _ := db.DB()
			if pingErr := sqlDB.Ping(); pingErr == nil {
				log.Info("connected to database")
				return db, nil
			}
		}
		log.Warn("database not ready, retrying...",
			"attempt", i+1,
			"max", maxRetries,
		)
		time.Sleep(retryInterval)
	}

	return nil, fmt.Errorf("could not connect to database after %d attempts: %w", maxRetries, err)
}

func runMigrations(db *gorm.DB, log *slog.Logger) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose set dialect: %w", err)
	}

	if err := goose.Up(sqlDB, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	log.Info("migrations applied successfully")
	return nil
}