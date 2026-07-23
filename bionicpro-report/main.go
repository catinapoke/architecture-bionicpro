package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"bionicpro-report/internal/config"
	"bionicpro-report/internal/handlers"
	"bionicpro-report/internal/keycloak"
	"bionicpro-report/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	var verifier *keycloak.Verifier
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		verifier, err = keycloak.NewVerifier(ctx, cfg)
		cancel()
		if err == nil {
			break
		}
		log.Printf("waiting for Keycloak: %v", err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("keycloak verifier: %v", err)
	}

	var db *sql.DB
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", cfg.DBURL)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err = db.PingContext(ctx)
			cancel()
		}
		if err == nil {
			break
		}
		if db != nil {
			_ = db.Close()
		}
		log.Printf("waiting for report db: %v", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("report db: %v", err)
	}
	defer db.Close()

	h := handlers.New(verifier, store.New(db))

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           h.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("bionicpro-report listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}
