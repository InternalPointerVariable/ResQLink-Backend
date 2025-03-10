package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/InternalPointerVariable/ResQLink-Backend/internal/api"
	"github.com/InternalPointerVariable/ResQLink-Backend/internal/disaster"
	"github.com/InternalPointerVariable/ResQLink-Backend/internal/user"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type app struct {
	user     user.Server
	disaster disaster.Server
}

func main() {
	if err := godotenv.Load(); err != nil {
		panic("Failed to load .env file.")
	}

	dbUrl, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		panic("DATABASE_URL not found.")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbUrl)
	defer pool.Close()
	if err != nil {
		panic(err)
	}

	redisUrl, ok := os.LookupEnv("REDIS_URL")
	if !ok {
		panic("REDIS_URL not found.")
	}

	opt, err := redis.ParseURL(redisUrl)
	if err != nil {
		panic(fmt.Errorf("redis url: %w", err))
	}

	redisClient := redis.NewClient(opt)

	app := app{
		user:     *user.NewServer(user.NewRepository(pool, redisClient)),
		disaster: *disaster.NewServer(disaster.NewRepository(pool, redisClient)),
	}

	router := http.NewServeMux()

	router.HandleFunc("GET /", health)
	router.Handle("POST /api/sign-up", api.HTTPHandler(app.user.SignUp))
	router.Handle("POST /api/sign-in", api.HTTPHandler(app.user.SignIn))
	router.Handle("GET /api/disaster-reports", api.HTTPHandler(app.disaster.GetDisasterReports))
	router.Handle("POST /api/disaster-reports", api.HTTPHandler(app.disaster.CreateDisasterReport))

	host, ok := os.LookupEnv("HOST")
	if !ok {
		panic("HOST not found.")
	}

	port, ok := os.LookupEnv("PORT")
	if !ok {
		panic("PORT not found.")
	}

	server := http.Server{
		Addr:    host + ":" + port,
		Handler: router,
	}

	slog.Info(fmt.Sprintf("Starting server on port: %s", port))

	server.ListenAndServe()
}

func health(w http.ResponseWriter, r *http.Request) {
	slog.Info("Hello, World!")
}
