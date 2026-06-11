package domain

import (
	"context"
	"time"
)

type SeatStatus string

const (
	SeatAvailable SeatStatus = "AVAILABLE" 
	SeatLocked    SeatStatus = "LOCKED"    
	SeatBooked    SeatStatus = "BOOKED"    
)

type Seat struct {
	ID      string     `json:"id" bson:"_id"`
	ShowID  string     `json:"show_id" bson:"show_id"`
	SeatNo  string     `json:"seat_no" bson:"seat_no"`
	Status  SeatStatus `json:"status" bson:"status"`
	UserID    string     `json:"user_id,omitempty" bson:"user_id,omitempty"`
	UserEmail string     `json:"user_email,omitempty" bson:"user_email,omitempty"`
	UpdatedAt time.Time  `json:"updated_at,omitempty" bson:"updated_at,omitempty"`
}

type SeatRepository interface {
	GetSeatMap(ctx context.Context, showID string) ([]Seat, error)
	GetSeat(ctx context.Context, showID string, seatNo string) (*Seat, error)
	UpdateSeatStatus(ctx context.Context, showID string, seatNo string, status SeatStatus, userID string, userEmail string) error
}