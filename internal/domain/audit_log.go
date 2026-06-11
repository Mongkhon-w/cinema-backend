package domain

import (
	"context"
	"time"
)

type AuditLog struct {
	ID        string    `json:"id" bson:"_id,omitempty"`
	Event     string    `json:"event" bson:"event"` 
	Details   string    `json:"details" bson:"details"`
	MovieID   string    `json:"movie_id,omitempty" bson:"movie_id,omitempty"` 
	UserID    string    `json:"user_id,omitempty" bson:"user_id,omitempty"`   
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
}

type AuditLogRepository interface {
	Store(ctx context.Context, log *AuditLog) error 
}