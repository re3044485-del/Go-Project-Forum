package repository

import (
	"context"
	"errors"

	"app/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// --- USER REPOSITORY ---

func (r *PostgresRepository) CreateUser(ctx context.Context, u *domain.User) error {
	query := `
		INSERT INTO users (username, email, password_hash, avatar_url)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	return r.db.QueryRow(ctx, query, u.Username, u.Email, u.PasswordHash, u.AvatarURL).
		Scan(&u.ID, &u.CreatedAt)
}

func (r *PostgresRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, username, email, password_hash, avatar_url, created_at FROM users WHERE email = $1`
	u := &domain.User{}
	err := r.db.QueryRow(ctx, query, email).
		Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.AvatarURL, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return u, nil
}

// --- THREADS REPOSITORY ---

func (r *PostgresRepository) GetThreads(ctx context.Context, limit, offset int) ([]domain.Thread, error) {
	query := `
		SELECT t.id, t.author_id, u.username, t.title, t.content, t.created_at
		FROM threads t
		JOIN users u ON t.author_id = u.id
		ORDER BY t.created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var threads []domain.Thread
	for rows.Next() {
		var t domain.Thread
		if err := rows.Scan(&t.ID, &t.AuthorID, &t.Author, &t.Title, &t.Content, &t.CreatedAt); err != nil {
			return nil, err
		}
		threads = append(threads, t)
	}
	return threads, nil
}

func (r *PostgresRepository) CreateThread(ctx context.Context, t *domain.Thread) error {
	query := `
		INSERT INTO threads (author_id, title, content)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	return r.db.QueryRow(ctx, query, t.AuthorID, t.Title, t.Content).Scan(&t.ID, &t.CreatedAt)
}

func (r *PostgresRepository) GetThreadByID(ctx context.Context, id int64) (*domain.Thread, error) {
	queryThread := `
		SELECT t.id, t.author_id, u.username, t.title, t.content, t.created_at
		FROM threads t
		JOIN users u ON t.author_id = u.id
		WHERE t.id = $1
	`
	t := &domain.Thread{}
	err := r.db.QueryRow(ctx, queryThread, id).Scan(&t.ID, &t.AuthorID, &t.Author, &t.Title, &t.Content, &t.CreatedAt)
	if err != nil {
		return nil, err
	}

	queryReplies := `
		SELECT r.id, r.thread_id, r.parent_reply_id, r.author_id, u.username, r.content, r.created_at
		FROM thread_replies r
		JOIN users u ON r.author_id = u.id
		WHERE r.thread_id = $1
		ORDER BY r.created_at ASC
	`
	rows, err := r.db.Query(ctx, queryReplies, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var reply domain.ThreadReply
		if err := rows.Scan(&reply.ID, &reply.ThreadID, &reply.ParentReplyID, &reply.AuthorID, &reply.Author, &reply.Content, &reply.CreatedAt); err != nil {
			return nil, err
		}
		t.Replies = append(t.Replies, reply)
	}

	return t, nil
}

func (r *PostgresRepository) CreateThreadReply(ctx context.Context, reply *domain.ThreadReply) error {
	query := `
		INSERT INTO thread_replies (thread_id, parent_reply_id, author_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	return r.db.QueryRow(ctx, query, reply.ThreadID, reply.ParentReplyID, reply.AuthorID, reply.Content).
		Scan(&reply.ID, &reply.CreatedAt)
}

// --- POSTS REPOSITORY ---

func (r *PostgresRepository) GetPosts(ctx context.Context, userID int64, limit, offset int) ([]domain.Post, error) {
	query := `
		SELECT p.id, p.author_id, u.username, u.avatar_url, p.content, p.media_urls, p.views_count, p.created_at,
		       COUNT(l.user_id) AS likes_count,
		       EXISTS(SELECT 1 FROM post_likes WHERE post_id = p.id AND user_id = $1) AS is_liked
		FROM posts p
		JOIN users u ON p.author_id = u.id
		LEFT JOIN post_likes l ON p.id = l.post_id
		GROUP BY p.id, u.username, u.avatar_url
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []domain.Post
	for rows.Next() {
		var p domain.Post
		if err := rows.Scan(&p.ID, &p.AuthorID, &p.Author, &p.AuthorAvatar, &p.Content, &p.MediaURLs, &p.ViewsCount, &p.CreatedAt, &p.LikesCount, &p.IsLiked); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, nil
}

func (r *PostgresRepository) CreatePost(ctx context.Context, p *domain.Post) error {
	query := `
		INSERT INTO posts (author_id, content, media_urls)
		VALUES ($1, $2, $3)
		RETURNING id, views_count, created_at
	`
	return r.db.QueryRow(ctx, query, p.AuthorID, p.Content, p.MediaURLs).Scan(&p.ID, &p.ViewsCount, &p.CreatedAt)
}

// TogglePostLike с атомарным переключением (лайк/дизлайк) в единой транзакции
func (r *PostgresRepository) TogglePostLike(ctx context.Context, postID, userID int64) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM post_likes WHERE post_id = $1 AND user_id = $2)`
	if err := tx.QueryRow(ctx, checkQuery, postID, userID).Scan(&exists); err != nil {
		return false, err
	}

	if exists {
		_, err = tx.Exec(ctx, `DELETE FROM post_likes WHERE post_id = $1 AND user_id = $2`, postID, userID)
		if err != nil {
			return false, err
		}
	} else {
		_, err = tx.Exec(ctx, `INSERT INTO post_likes (post_id, user_id) VALUES ($1, $2)`, postID, userID)
		if err != nil {
			return false, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}

	return !exists, nil
}

func (r *PostgresRepository) CreatePostComment(ctx context.Context, c *domain.PostComment) error {
	query := `
		INSERT INTO post_comments (post_id, author_id, parent_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	return r.db.QueryRow(ctx, query, c.PostID, c.AuthorID, c.ParentID, c.Content).
		Scan(&c.ID, &c.CreatedAt)
}

// --- CHAT REPOSITORY ---

func (r *PostgresRepository) SaveChatMessage(ctx context.Context, msg *domain.ChatMessage) error {
	query := `
		INSERT INTO chat_messages (room_id, sender_id, content, media_url)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	err := r.db.QueryRow(ctx, query, msg.RoomID, msg.SenderID, msg.Content, msg.MediaURL).
		Scan(&msg.ID, &msg.CreatedAt)
	if err != nil {
		return err
	}

	// Получаем имя отправителя для вещания по сокету
	return r.db.QueryRow(ctx, `SELECT username FROM users WHERE id = $1`, msg.SenderID).Scan(&msg.Sender)
}

func (r *PostgresRepository) GetChatHistory(ctx context.Context, roomID int64, limit int) ([]domain.ChatMessage, error) {
	query := `
		SELECT m.id, m.room_id, m.sender_id, u.username, m.content, m.media_url, m.created_at
		FROM chat_messages m
		JOIN users u ON m.sender_id = u.id
		WHERE m.room_id = $1
		ORDER BY m.created_at DESC
		LIMIT $2
	`
	rows, err := r.db.Query(ctx, query, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []domain.ChatMessage
	for rows.Next() {
		var msg domain.ChatMessage
		if err := rows.Scan(&msg.ID, &msg.RoomID, &msg.SenderID, &msg.Sender, &msg.Content, &msg.MediaURL, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	// Реверсируем список для хронологического порядка
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}
