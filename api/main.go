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
	"brainhub/api/router"
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
	adminClient, err := gbrain.NewAdminClient(configuration.GBrainBaseURL, configuration.GBrainAdminToken)
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
	writerService, err := gbrain.NewWriterService(
		configuration.GBrainBaseURL, adminClient, repositories, clock, configuration.WriterCredentialKey,
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := writerService.Backfill(context.Background()); err != nil {
		log.Printf("brain writer backfill incomplete: %v", err)
	}
	readerService, err := gbrain.NewReaderService(
		configuration.GBrainBaseURL, adminClient, repositories, clock, ids, configuration.WriterCredentialKey,
	)
	if err != nil {
		log.Fatal(err)
	}
	shimClient, err := gbrain.NewShimClient(configuration.ShimURL, configuration.ShimToken)
	if err != nil {
		log.Fatal(err)
	}
	brainUseCase, err := interactor.NewBrainUseCase(repositories, repositories, repositories, shimClient, client, writerService, clock, ids)
	if err != nil {
		log.Fatal(err)
	}
	pageUseCase, err := interactor.NewPageUseCase(writerService, repositories, repositories, configuration.PublicPageTypes)
	if err != nil {
		log.Fatal(err)
	}
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
	accessUseCase, err := interactor.NewAccessUseCase(
		repositories, repositories, repositories, repositories, repositories,
		repositories, adminClient, clock, ids,
	)
	if err != nil {
		log.Fatal(err)
	}
	mcpUseCase, err := interactor.NewMCPUseCase(repositories, repositories, readerService, writerService, clock, ids, configuration.PublicMCPURL)
	if err != nil {
		log.Fatal(err)
	}
	if err := mcpUseCase.ReconcileReaders(context.Background()); err != nil {
		log.Printf("brain reader reconciliation incomplete: %v", err)
	}
	httpHandler, err := router.New(
		brainUseCase, pageUseCase, authUseCase, accessUseCase, mcpUseCase,
		configuration.PublicMCPURL, configuration.PublicWebURL, proxy, configuration.Production,
	)
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:              ":8080",
		Handler:           httpHandler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
