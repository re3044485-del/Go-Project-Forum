package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"app/internal/domain"
	"app/internal/middleware"
	"app/internal/repository"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	repo      *repository.PostgresRepository
	jwtSecret string
}

func NewHandler(repo *repository.PostgresRepository, jwtSecret string) *Handler {
	return &Handler{repo: repo, jwtSecret: jwtSecret}
}

// --- AUTH HANDLERS ---

type RegisterReq struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	user := &domain.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
	}

	if err := h.repo.CreateUser(r.Context(), user); err != nil {
		http.Error(w, "Failed to create user", http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully"})
}

type LoginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	user, err := h.repo.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := middleware.GenerateToken(user.ID, h.jwtSecret)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Запись токена в HTTP-only Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // В продакшене со свойством HTTPS установить в true
		SameSite: http.SameSiteLaxMode,
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": token,
		"user":  map[string]interface{}{"id": user.ID, "username": user.Username},
	})
}

// --- THREADS HANDLERS ---

func (h *Handler) GetThreads(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit := 20
	offset := (page - 1) * limit

	threads, err := h.repo.GetThreads(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, "Failed to fetch threads", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(threads)
}

func (h *Handler) CreateThread(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserIDFromContext(r.Context())

	var t domain.Thread
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	t.AuthorID = userID

	if err := h.repo.CreateThread(r.Context(), &t); err != nil {
		http.Error(w, "Failed to create thread", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

func (h *Handler) GetThreadByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	thread, err := h.repo.GetThreadByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Thread not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(thread)
}

func (h *Handler) ReplyThread(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserIDFromContext(r.Context())
	threadID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var reply domain.ThreadReply
	if err := json.NewDecoder(r.Body).Decode(&reply); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	reply.ThreadID = threadID
	reply.AuthorID = userID

	if err := h.repo.CreateThreadReply(r.Context(), &reply); err != nil {
		http.Error(w, "Failed to reply", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(reply)
}

// --- POSTS HANDLERS ---

func (h *Handler) GetPosts(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserIDFromContext(r.Context())
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit := 10
	offset := (page - 1) * limit

	posts, err := h.repo.GetPosts(r.Context(), userID, limit, offset)
	if err != nil {
		http.Error(w, "Failed to fetch posts", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(posts)
}

func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserIDFromContext(r.Context())

	var p domain.Post
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	p.AuthorID = userID

	if err := h.repo.CreatePost(r.Context(), &p); err != nil {
		http.Error(w, "Failed to create post", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func (h *Handler) ToggleLike(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserIDFromContext(r.Context())
	postID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	isLiked, err := h.repo.TogglePostLike(r.Context(), postID, userID)
	if err != nil {
		http.Error(w, "Failed to toggle like", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"is_liked": isLiked})
}

func (h *Handler) CommentPost(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserIDFromContext(r.Context())
	postID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var comment domain.PostComment
	if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	comment.PostID = postID
	comment.AuthorID = userID

	if err := h.repo.CreatePostComment(r.Context(), &comment); err != nil {
		http.Error(w, "Failed to create comment", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comment)
}

// --- CHAT HISTORY HANDLER ---

func (h *Handler) GetChatHistory(w http.ResponseWriter, r *http.Request) {
	roomID, err := strconv.ParseInt(r.URL.Query().Get("room_id"), 10, 64)
	if err != nil {
		http.Error(w, "Room ID required", http.StatusBadRequest)
		return
	}

	history, err := h.repo.GetChatHistory(r.Context(), roomID, 50)
	if err != nil {
		http.Error(w, "Failed to fetch history", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(history)
}
