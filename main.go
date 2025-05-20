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
	"github.com/InternalPointerVariable/ResQLink-Backend/internal/ws"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/rs/cors"
)

type app struct {
	user     user.Server
	disaster disaster.Server
	ws       ws.Server
}

func main() {
	if err := godotenv.Load(); err != nil {
		panic("Failed to load .env file.")
	}

	baseURL, ok := os.LookupEnv("BASE_URL")
	if !ok {
		panic("BASE_URL not found.")
	}

	dbURL, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		panic("DATABASE_URL not found.")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	defer pool.Close()
	if err != nil {
		panic(err)
	}

	redisURL, ok := os.LookupEnv("REDIS_URL")
	if !ok {
		panic("REDIS_URL not found.")
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		panic(fmt.Errorf("redis url: %w", err))
	}

	redisClient := redis.NewClient(opt)

	hub := ws.NewHub(redisClient)
	go hub.Start()

	disasterRepo := disaster.NewRepository(pool, redisClient)
	disasterWsServer := disaster.NewSocketServer(disasterRepo)
	wsHandlers := map[string]ws.EventHandler{"disaster": disasterWsServer}

	app := app{
		user:     *user.NewServer(user.NewRepository(pool, redisClient)),
		disaster: *disaster.NewServer(disasterRepo, baseURL),
		ws:       *ws.NewServer(hub, wsHandlers),
	}

	router := http.NewServeMux()

	router.HandleFunc("GET /ws", app.ws.HandleConnection)
	router.HandleFunc("GET /", health)

	router.Handle("POST /api/sign-up", api.HTTPHandler(app.user.SignUp))
	router.Handle("POST /api/sign-in", api.HTTPHandler(app.user.SignIn))
	router.Handle("POST /api/sign-in/anonymous", api.HTTPHandler(app.user.SignInAnonymous))
	router.Handle("POST /api/sign-out", api.HTTPHandler(app.user.SignOut))
	router.Handle("GET /api/session", api.HTTPHandler(app.user.GetSession))

	router.Handle(
		"GET /api/reporters/{reporterId}/reports",
		api.HTTPHandler(app.disaster.ListDisasterReportsByReporter),
	)
	router.Handle(
		"PATCH /api/reporters/{reporterId}/reports",
		api.HTTPHandler(app.disaster.SetResponder),
	)
	router.Handle("GET /api/reports", api.HTTPHandler(app.disaster.ListDisasterReports))
	router.Handle(
		"POST /api/reports",
		api.HTTPHandler(app.disaster.CreateDisasterReportJson),
	)

	host, ok := os.LookupEnv("HOST")
	if !ok {
		panic("HOST not found.")
	}

	port, ok := os.LookupEnv("PORT")
	if !ok {
		panic("PORT not found.")
	}

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PATCH", "OPTIONS"},
	})

	server := http.Server{
		Addr:    host + ":" + port,
		Handler: c.Handler(router), // TODO: Wrap authenticated routes with `AuthMiddleware`
	}

	slog.Info(fmt.Sprintf("Starting server on port: %s", port))

	server.ListenAndServe()
}

func health(w http.ResponseWriter, r *http.Request) {
	slog.Info("Hello, World!")
}

func EnableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}
