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

	_ "github.com/ClickHouse/clickhouse-go/v2"

	"bionicpro-report/internal/config"
	"bionicpro-report/internal/handlers"
	"bionicpro-report/internal/keycloak"
	"bionicpro-report/internal/s3store"
	"bionicpro-report/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	verifier, err := retry(5, 3*time.Second, "Keycloak", func() (*keycloak.Verifier, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return keycloak.NewVerifier(ctx, cfg)
	})
	if err != nil {
		log.Fatalf("keycloak verifier: %v", err)
	}

	db, err := retry(10, 2*time.Second, "report db", func() (*sql.DB, error) {
		db, err := sql.Open("clickhouse", cfg.DBURL)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			_ = db.Close()
			return nil, err
		}
		return db, nil
	})
	if err != nil {
		log.Fatalf("report db: %v", err)
	}
	defer db.Close()

	objects, err := retry(10, 2*time.Second, "minio", func() (*s3store.Store, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		return s3store.New(ctx, s3store.Config{
			Endpoint:  cfg.S3Endpoint,
			AccessKey: cfg.S3AccessKey,
			SecretKey: cfg.S3SecretKey,
			Bucket:    cfg.S3Bucket,
			CDNBase:   cfg.CDNBaseURL,
			UseSSL:    cfg.S3UseSSL,
		})
	})
	if err != nil {
		log.Fatalf("minio: %v", err)
	}

	h := handlers.New(verifier, store.New(db), objects)

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

func retry[T any](attempts int, delay time.Duration, label string, fn func() (T, error)) (T, error) {
	var (
		result T
		err    error
	)
	for i := 0; i < attempts; i++ {
		result, err = fn()
		if err == nil {
			return result, nil
		}
		log.Printf("waiting for %s: %v", label, err)
		if i+1 < attempts {
			time.Sleep(delay)
		}
	}
	return result, err
}
