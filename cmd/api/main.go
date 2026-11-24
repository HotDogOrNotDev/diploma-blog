package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"advanced-blog-management-system/internal/handler"
	"advanced-blog-management-system/internal/middleware"
	"advanced-blog-management-system/internal/repository"
	"advanced-blog-management-system/internal/service"
	"advanced-blog-management-system/pkg/database"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("warning: .env file not found, using environment variables")
	}

	cfg := loadConfig()
	log.Println("configuration loaded")

	db, err := database.NewPostgresDB(database.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		DBName:   cfg.DBName,
		SSLMode:  cfg.DBSSLMode,
	})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("connected to PostgreSQL")

	migrations, err := database.LoadMigrationFiles("migrations")
	if err != nil {
		log.Fatalf("failed to load migrations: %v", err)
	}
	if err := database.RunMigrations(db, migrations); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	log.Println("migrations applied")

	userRepo := repository.NewUserRepository(db)
	postRepo := repository.NewPostRepository(db)
	commentRepo := repository.NewCommentRepository(db)

	userService := service.NewUserService(userRepo)
	postService := service.NewPostService(postRepo, userRepo, commentRepo)
	commentService := service.NewCommentService(commentRepo, postRepo, userRepo)

	authHandler := handler.NewAuthHandler(userService, cfg.JWTSecret)
	postHandler := handler.NewPostHandler(postService)
	commentHandler := handler.NewCommentHandler(commentService)

	router := setupRouter(cfg.JWTSecret, authHandler, postHandler, commentHandler)

	addr := cfg.ServerHost + ":" + strconv.Itoa(cfg.ServerPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	publisherCtx, publisherCancel := context.WithCancel(context.Background())
	postService.StartPublisher(publisherCtx, cfg.PublisherInterval, cfg.PublisherWorkers)

	go func() {
		log.Printf("server listening on http://%s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	sig := <-stop
	log.Printf("shutdown signal received: %v", sig)

	publisherCancel()
	postService.WaitPublisher()
	log.Println("publisher stopped")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	} else {
		log.Println("server stopped gracefully")
	}
}

type config struct {
	DBHost            string
	DBPort            int
	DBUser            string
	DBPassword        string
	DBName            string
	DBSSLMode         string
	JWTSecret         string
	ServerHost        string
	ServerPort        int
	PublisherInterval time.Duration
	PublisherWorkers  int
}

func loadConfig() config {
	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	return config{
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnvAsInt("DB_PORT", 5432),
		DBUser:            getEnv("DB_USER", "postgres"),
		DBPassword:        getEnv("DB_PASSWORD", "postgres"),
		DBName:            getEnv("DB_NAME", "blog_db"),
		DBSSLMode:         getEnv("DB_SSLMODE", "disable"),
		JWTSecret:         jwtSecret,
		ServerHost:        getEnv("SERVER_HOST", "0.0.0.0"),
		ServerPort:        getEnvAsInt("SERVER_PORT", 8080),
		PublisherInterval: time.Duration(getEnvAsInt("PUBLISHER_INTERVAL_SEC", 10)) * time.Second,
		PublisherWorkers:  getEnvAsInt("PUBLISHER_WORKERS", 4),
	}
}

func setupRouter(
	jwtSecret string,
	authHandler *handler.AuthHandler,
	postHandler *handler.PostHandler,
	commentHandler *handler.CommentHandler,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.RecoveryMiddleware)
	r.Use(middleware.CORSMiddleware)

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", handler.HealthCheckHandler)
		r.Post("/register", authHandler.RegisterHandler)
		r.Post("/login", authHandler.LoginHandler)

		r.Get("/posts", postHandler.GetAllPosts)
		r.Get("/posts/{id}", postHandler.GetPost)
		r.Get("/users/{id}/posts", postHandler.GetPostsByAuthor)
		r.Get("/posts/{postId}/comments", commentHandler.GetCommentsByPostID)

		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(jwtSecret))

			r.Post("/posts", postHandler.CreatePost)
			r.Put("/posts/{id}", postHandler.UpdatePost)
			r.Delete("/posts/{id}", postHandler.DeletePost)

			r.Post("/posts/{postId}/comments", commentHandler.CreateComment)
			r.Put("/comments/{id}", commentHandler.UpdateComment)
			r.Delete("/comments/{id}", commentHandler.DeleteComment)
		})
	})

	return r
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}
