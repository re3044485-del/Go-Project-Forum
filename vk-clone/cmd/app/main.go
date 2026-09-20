package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"app/internal/config"
	"app/internal/database"
	"app/internal/handler"
	"app/internal/middleware"
	"app/internal/repository"
	"app/internal/ws"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config load error: %v", err)
	}

	ctx := context.Background()
	dbPool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection error: %v", err)
	}
	defer dbPool.Close()

	repo := repository.NewPostgresRepository(dbPool)
	h := handler.NewHandler(repo, cfg.JWTSecret)

	hub := ws.NewHub(repo)
	go hub.Run()

	r := chi.NewRouter()

	// Настройка CORS (чтобы браузер не блокировал запросы)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// Общие middlewares
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	// 1. Отдача фронтенда (Главная страница и статические файлы)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/static/index.html")
	})

	fileServer := http.FileServer(http.Dir("./web/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// 2. Публичные эндпоинты авторизации
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
	})

	// 3. WebSocket соединение (вынесено отдельно, так как токен передается в URL запроса)
	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWS(hub, w, r)
	})

	// 4. Защищенные API роуты (требуют JWT-токен)
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(cfg.JWTSecret))

		// Треды
		r.Get("/api/threads", h.GetThreads)
		r.Post("/api/threads", h.CreateThread)
		r.Get("/api/threads/{id}", h.GetThreadByID)
		r.Post("/api/threads/{id}/reply", h.ReplyThread)

		// Посты (Лента)
		r.Get("/api/posts", h.GetPosts)
		r.Post("/api/posts", h.CreatePost)
		r.Post("/api/posts/{id}/like", h.ToggleLike)
		r.Post("/api/posts/{id}/comments", h.CommentPost)

		// История чата
		r.Get("/api/chat/history", h.GetChatHistory)
	})

	// Запуск сервера с привязкой к 0.0.0.0 (обязательно для Docker/Render)
	addr := fmt.Sprintf("0.0.0.0:%s", cfg.Port)
	fmt.Printf("Server starting on %s...\n", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}
}
