package domain

import "time"

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	AvatarURL    string    `json:"avatar_url"`
	CreatedAt    time.Time `json:"created_at"`
}

type Thread struct {
	ID        int64         `json:"id"`
	AuthorID  int64         `json:"author_id"`
	Author    string        `json:"author_username,omitempty"`
	Title     string        `json:"title"`
	Content   string        `json:"content"`
	CreatedAt time.Time     `json:"created_at"`
	Replies   []ThreadReply `json:"replies,omitempty"`
}

type ThreadReply struct {
	ID            int64     `json:"id"`
	ThreadID      int64     `json:"thread_id"`
	ParentReplyID *int64    `json:"parent_reply_id,omitempty"`
	AuthorID      int64     `json:"author_id"`
	Author        string    `json:"author_username,omitempty"`
	Content       string    `json:"content"`
	CreatedAt     time.Time `json:"created_at"`
}

type Post struct {
	ID           int64         `json:"id"`
	AuthorID     int64         `json:"author_id"`
	Author       string        `json:"author_username,omitempty"`
	AuthorAvatar string        `json:"author_avatar,omitempty"`
	Content      string        `json:"content"`
	MediaURLs    []string      `json:"media_urls"`
	ViewsCount   int64         `json:"views_count"`
	LikesCount   int64         `json:"likes_count"`
	IsLiked      bool          `json:"is_liked"`
	CreatedAt    time.Time     `json:"created_at"`
	Comments     []PostComment `json:"comments,omitempty"`
}

type PostComment struct {
	ID        int64     `json:"id"`
	PostID    int64     `json:"post_id"`
	AuthorID  int64     `json:"author_id"`
	Author    string    `json:"author_username,omitempty"`
	ParentID  *int64    `json:"parent_id,omitempty"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type ChatMessage struct {
	ID        int64     `json:"id"`
	RoomID    int64     `json:"room_id"`
	SenderID  int64     `json:"sender_id"`
	Sender    string    `json:"sender_username,omitempty"`
	Content   string    `json:"content"`
	MediaURL  string    `json:"media_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
