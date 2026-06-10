package domain

import "context"

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
	UserID  string     `json:"user_id,omitempty" bson:"user_id,omitempty"`
}

type SeatRepository interface {
	GetSeatMap(ctx context.Context, showID string) ([]Seat, error)
	UpdateSeatStatus(ctx context.Context, showID string, seatNo string, status SeatStatus, userID string) error
}