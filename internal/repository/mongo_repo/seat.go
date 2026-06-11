package mongo_repo

import (
	"context"
	"log"

	"cinema-backend/internal/domain"

	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoSeatRepository struct {
	collection *mongo.Collection
}

func NewMongoSeatRepository(db *mongo.Database) *MongoSeatRepository {
	return &MongoSeatRepository{collection: db.Collection("seats")}
}

func (r *MongoSeatRepository) EnsureSeatsInitialized(ctx context.Context, showID string) error {
	count, err := r.collection.CountDocuments(ctx, bson.M{"show_id": showID})
	if err != nil {
		return err
	}

	if count == 0 {
		log.Printf("[MongoDB] Seeding seats for show %s", showID)
		var seats []interface{}
		rows := []string{"A", "B", "C", "D"}
		for _, rName := range rows {
			for i := 1; i <= 8; i++ {
				seatNo := rName + string(rune('0'+i))
				seats = append(seats, domain.Seat{
					ID:     showID + "_" + seatNo,
					ShowID: showID,
					SeatNo: seatNo,
					Status: domain.SeatAvailable,
				})
			}
		}
		_, err = r.collection.InsertMany(ctx, seats)
		if err != nil {
			log.Printf("[MongoDB] Failed to seed seats: %v", err)
			return err
		}
	}
	return nil
}

func (r *MongoSeatRepository) GetSeatMap(ctx context.Context, showID string) ([]domain.Seat, error) {
	// Auto-seed seats for simplicity of this assignment
	if err := r.EnsureSeatsInitialized(ctx, showID); err != nil {
		return nil, err
	}

	cursor, err := r.collection.Find(ctx, bson.M{"show_id": showID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var seats []domain.Seat
	if err := cursor.All(ctx, &seats); err != nil {
		return nil, err
	}

	return seats, nil
}

func (r *MongoSeatRepository) GetSeat(ctx context.Context, showID string, seatNo string) (*domain.Seat, error) {
	var seat domain.Seat
	err := r.collection.FindOne(ctx, bson.M{"show_id": showID, "seat_no": seatNo}).Decode(&seat)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &seat, nil
}

func (r *MongoSeatRepository) UpdateSeatStatus(ctx context.Context, showID, seatNo string, status domain.SeatStatus, userID string, userEmail string) error {
	filter := bson.M{"show_id": showID, "seat_no": seatNo}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	if userID != "" {
		update["$set"].(bson.M)["user_id"] = userID
		update["$set"].(bson.M)["user_email"] = userEmail
	} else if status == domain.SeatAvailable {
		// Clear user_id when the seat becomes available
		update["$unset"] = bson.M{"user_id": "", "user_email": ""}
	}

	_, err := r.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		log.Printf("[MongoDB] Error updating seat %s to %v for show %s: %v", seatNo, status, showID, err)
		return err
	}
	return nil
}
