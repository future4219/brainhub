package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"brainhub/adapter/authentication"
	"brainhub/adapter/clock"
	"brainhub/adapter/database"
	"brainhub/adapter/database/repository"
	"brainhub/adapter/gbrain"
	"brainhub/adapter/ulid"
	"brainhub/api/api/router"
	"brainhub/config"
	"brainhub/usecase/interactor"
)

func main() {
	configuration, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	client, err := gbrain.NewClient(configuration.GBrainBaseURL, configuration.GBrainClientID, configuration.GBrainClientSecret)
	if err != nil {
		log.Fatal(err)
	}
	brainUseCase, err := interactor.NewBrainUseCase(client)
	if err != nil {
		log.Fatal(err)
	}
	pageUseCase, err := interactor.NewPageUseCase(client, client, configuration.PublicPageTypes)
	if err != nil {
		log.Fatal(err)
	}
	proxy, err := gbrain.NewProxy(configuration.GBrainBaseURL)
	if err != nil {
		log.Fatal(err)
	}

	pool, err := database.Open(context.Background(), configuration.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	clock := clock.Clock{}
	ids := ulid.Generator{}
	repositories := repository.New(pool, clock, ids)
	hasher, err := authentication.NewBcrypt(12)
	if err != nil {
		log.Fatal("initialize password hasher")
	}
	authUseCase, err := interactor.NewAuthUseCase(
		repositories,
		repositories,
		hasher,
		authentication.NewRateLimiter(5, 15*time.Minute),
		clock,
		ids,
	)
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:              ":8080",
		Handler:           router.New(brainUseCase, pageUseCase, authUseCase, configuration.PublicMCPURL, proxy, configuration.Production),
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
