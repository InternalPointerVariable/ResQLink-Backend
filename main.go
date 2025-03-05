package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/InternalPointerVariable/ResQLink-Backend/internal/api"
	"github.com/InternalPointerVariable/ResQLink-Backend/internal/user"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type app struct {
	user user.Server
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

	app := app{
		user: *user.NewServer(user.NewRepository(pool)),
	}

	router := http.NewServeMux()

	router.HandleFunc("GET /", health)
	router.Handle("POST /sign-up", api.HTTPHandler(app.user.SignUp))
	router.Handle("POST /sign-in", api.HTTPHandler(app.user.SignIn))

	host, ok := os.LookupEnv("HOST")
	if !ok {
		panic("HOST not found.")
	}

	port, ok := os.LookupEnv("PORT")
	if !ok {
		panic("PORT not found.")
	}

	server := http.Server{
		Addr: host + ":" + port,
	}

	slog.Info(fmt.Sprintf("Starting server on port: %s", port))

	server.ListenAndServe()
}

func health(w http.ResponseWriter, r *http.Request) {
	slog.Info("Hello, World!")
}
