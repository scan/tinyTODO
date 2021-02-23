package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	chilogger "github.com/766b/chi-logger"
	"go.uber.org/zap"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/go-redis/redis/v8"

	"github.com/gorilla/websocket"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"

	"github.com/scan/tinyTODO/graphql"
)

//go:generate go run github.com/99designs/gqlgen

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("cannot initialise zap logger: %v", err)
	}
	defer logger.Sync()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		logger.Fatal("redis connection failed", zap.Error(err))
	}
	defer redisClient.Close()

	router := chi.NewRouter()

	router.Use(
		middleware.Recoverer,
		middleware.RequestID,
		middleware.RealIP,
		middleware.Heartbeat("/health"),
		cors.Handler(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: false,
			MaxAge:           300, // Maximum value not ignored by any of major browsers
		}),
		chilogger.NewZapMiddleware("router", logger),
		middleware.Compress(6, "test/plain", "text/html", "application/json"),
	)

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"health":"ok"}`))
	})

	websocketUpgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			if origin, ok := os.LookupEnv("ALLOWED_CORS_ORIGIN"); ok {
				return r.Host == origin
			}

			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	config := graphql.Config{
		Resolvers: graphql.NewResolver(logger, redisClient),
	}

	graphqlHandler := handler.New(
		graphql.NewExecutableSchema(config),
	)

	graphqlHandler.AddTransport(transport.POST{})
	graphqlHandler.AddTransport(transport.GET{})
	graphqlHandler.AddTransport(transport.Options{})
	graphqlHandler.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader:              websocketUpgrader,
	})
	graphqlHandler.AddTransport(transport.MultipartForm{
		MaxUploadSize: 50 * 1024 * 1024,
		MaxMemory:     64 * 1024 * 1024,
	})

	graphqlHandler.Use(extension.Introspection{})

	router.Get("/graphql", graphqlHandler.ServeHTTP)
	router.Post("/graphql", graphqlHandler.ServeHTTP)

	logger.Info("Starting server...")
	if err := http.ListenAndServe("0.0.0.0:8080", router); err != nil {
		logger.Error("fatal server error", zap.Error(err))
	}
}
