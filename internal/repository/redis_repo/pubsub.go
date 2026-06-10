package redis_repo

import (
	"context"
	"encoding/json"
	"cinema-backend/internal/domain"
	"github.com/go-redis/redis/v8"
)

const AuditLogChannel = "cinema:audit:logs"

type PubSubRepository struct {
	client *redis.Client
}

func NewPubSubRepository(client *redis.Client) *PubSubRepository {
	return &PubSubRepository{client: client}
}

// ส่ง Log Event เข้า Queue 
func (p *PubSubRepository) PublishAuditLog(ctx context.Context, log *domain.AuditLog) error {
	data, err := json.Marshal(log)
	if err != nil {
		return err
	}
	return p.client.Publish(ctx, AuditLogChannel, data).Err()
}

// รันเป็น Background คอยดึงข้อมูลจาก Queue มาเซฟลง MongoDB
func (p *PubSubRepository) SubscribeAuditLog(ctx context.Context, mongoRepo domain.AuditLogRepository) {
	pubsub := p.client.Subscribe(ctx, AuditLogChannel)
	defer pubsub.Close()

	ch := pubsub.Channel()

	for msg := range ch {
		var logEntry domain.AuditLog
		if err := json.Unmarshal([]byte(msg.Payload), &logEntry); err == nil {
			// บันทึกลง MongoDB แบบ Async
			_ = mongoRepo.Store(context.Background(), &logEntry)
		}
	}
}