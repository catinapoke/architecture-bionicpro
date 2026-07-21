package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bionicpro-auth/internal/config"
	"bionicpro-auth/internal/handlers"
	"bionicpro-auth/internal/oidcauth"
	"bionicpro-auth/internal/profile"
	"bionicpro-auth/internal/session"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	var oidcClient *oidcauth.Client
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		oidcClient, err = oidcauth.New(ctx, cfg)
		cancel()
		if err == nil {
			break
		}
		log.Printf("waiting for Keycloak: %v", err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("oidc: %v", err)
	}

	var profiles *profile.Store
	for i := 0; i < 10; i++ {
		profiles, err = profile.NewStore(cfg.DatabaseURL)
		if err == nil {
			break
		}
		log.Printf("waiting for profile db: %v", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("profile store: %v", err)
	}
	defer profiles.Close()

	store := session.NewStore(cfg.SessionTTL)
	h := handlers.New(cfg, oidcClient, store, profiles)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           h.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("bionicpro-auth listening on :%s", cfg.Port)
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
