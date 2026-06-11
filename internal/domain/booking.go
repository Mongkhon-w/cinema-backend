package domain

import (
	"context"
	"time"
)

type Booking struct {
	ID        string    `json:"id" bson:"_id,omitempty"`
	UserID    string    `json:"user_id" bson:"user_id"`
	ShowID    string    `json:"show_id" bson:"show_id"`
	SeatNo    string    `json:"seat_no" bson:"seat_no"`
	Status    string    `json:"status" bson:"status"` 
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

type BookingUsecase interface {
	SelectSeat(ctx context.Context, userID, userEmail, showID, seatNo string) error
	ConfirmPayment(ctx context.Context, userID, userEmail, showID, seatNo string) error
}