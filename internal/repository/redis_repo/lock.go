package redis_repo

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisLockRepository struct {
	client *redis.Client
}

func NewRedisLockRepository(client *redis.Client) *RedisLockRepository {
	return &RedisLockRepository{client: client}
}

// ล็อกที่นั่งเป็นเวลา 5 นาที 
func (r *RedisLockRepository) AcquireLock(ctx context.Context, showID, seatNo, userID string) (bool, error) {
	lockKey := fmt.Sprintf("lock:show:%s:seat:%s", showID, seatNo)
	
	// ถ้ารหัสคีย์นี้ไม่มีอยู่จริงใน Redis ให้บันทึกค่า userID 
	success, err := r.client.SetNX(ctx, lockKey, userID, 5*time.Minute).Result() 
	if err != nil {
		return false, err
	}
	return success, nil
}

// ปล่อยล็อกเมื่อชำระเงินเสร็จสิ้น หรือหมดเวลาการล็อก
func (r *RedisLockRepository) ReleaseLock(ctx context.Context, showID, seatNo string) error {
	lockKey := fmt.Sprintf("lock:show:%s:seat:%s", showID, seatNo)
	return r.client.Del(ctx, lockKey).Err()
}