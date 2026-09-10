// Command server is the entrypoint for the autocheck.kz import microservice.
// It wires configuration, database, router, and the import scheduler.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"awesomeProject5/internal/api"
	"awesomeProject5/internal/config"
	"awesomeProject5/internal/database"
	"awesomeProject5/internal/domain"
	"awesomeProject5/internal/importer"
	"awesomeProject5/internal/repository"
	"awesomeProject5/internal/scheduler"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	repo := repository.New(db)
	imp := importer.New(repo)
	router := api.NewRouter(repo, imp)

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on :%s", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	// Start scheduled import if configured.
	var sched *scheduler.Scheduler
	if cfg.ImportCron != "" && cfg.ImportFile != "" {
		sched = scheduler.New(cfg.ImportCron, cfg.ImportFile, func(ctx context.Context, fileName string, data []byte) (*domain.ImportReport, error) {
			return imp.Import(ctx, "scheduled", fileName, data)
		})
		sched.Start(context.Background())
	} else {
		log.Println("scheduler: not configured (IMPORT_CRON or IMPORT_FILE missing)")
	}

	// Graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	if sched != nil {
		sched.Stop()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
