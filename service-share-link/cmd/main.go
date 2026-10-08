package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"service-share-link/internal/platform/config"
	"service-share-link/internal/platform/database"
	"service-share-link/internal/platform/deps"
	"service-share-link/internal/platform/redis"
	"service-share-link/internal/share-link/datastore"
	"service-share-link/internal/share-link/service"
	"service-share-link/migrations"
	"syscall"
)

func main() {
	cfgEnv := config.LoadConfigEnv()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.RegisterConnectDatabase(ctx, cfgEnv.DatabaseUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := database.RunMigrations(pool, migrations.FS); err != nil {
		log.Fatal(err)
	}

	redisClient, err := redis.RegisterConnectRedis(ctx, cfgEnv.Redis)
	if err != nil {
		log.Fatal(err)
	}
	defer redisClient.Close()

	deps := deps.NewDeps(pool, redisClient)

	viewFlusher := service.NewViewFlusher(
		datastore.NewRepoShareLink(pool),
		datastore.NewRedisViewCounter(redisClient),
		service.ViewFlushInterval,
	)

	flusherDone := make(chan struct{})
	go func() {
		defer close(flusherDone)
		viewFlusher.Run(ctx)
	}()

	NewHttpServer(ctx, cfgEnv, deps)

	stop()
	<-flusherDone
}
