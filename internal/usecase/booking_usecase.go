package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cinema-backend/internal/domain"
	"cinema-backend/internal/repository/redis_repo"
	"cinema-backend/internal/service"
)

type bookingUsecase struct {
	seatRepo   domain.SeatRepository
	lockRepo   *redis_repo.RedisLockRepository
	pubSubRepo *redis_repo.PubSubRepository
	lineNotify *service.LineNotifyService
}

func NewBookingUsecase(
	seatRepo domain.SeatRepository,
	lockRepo *redis_repo.RedisLockRepository,
	pubSubRepo *redis_repo.PubSubRepository,
	lineNotify *service.LineNotifyService,
) domain.BookingUsecase {
	return &bookingUsecase{
		seatRepo:   seatRepo,
		lockRepo:   lockRepo,
		pubSubRepo: pubSubRepo,
		lineNotify: lineNotify,
	}
}

// จองและล็อกที่นั่งเป็นเวลา 5 นาที
func (b *bookingUsecase) SelectSeat(ctx context.Context, userID, userEmail, showID, seatNo string) error {
	// พยายามดึงสิทธิ์ Distributed Lock ผ่าน Redis (ป้องกันคนกดพร้อมกัน)
	acquired, err := b.lockRepo.AcquireLock(ctx, showID, seatNo, userID)
	if err != nil {
		b.publishErrorLog("LOCK_FAIL", fmt.Sprintf("User %s failed to acquire lock for seat %s due to system error: %v", userID, seatNo, err))
		return err
	}
	if !acquired {
		b.publishErrorLog("LOCK_FAIL", fmt.Sprintf("User %s tried to lock seat %s but it is already locked by another process", userID, seatNo))
		return errors.New("seat is already locked or booked by another user") // [cite: 60]
	}

	// อัปเดตสถานะที่นั่งเป็น LOCKED ลงฐานข้อมูลหลัก (MongoDB)
	err = b.seatRepo.UpdateSeatStatus(ctx, showID, seatNo, domain.SeatLocked, userID, userEmail)
	if err != nil {
		_ = b.lockRepo.ReleaseLock(ctx, showID, seatNo) // คืนล็อกใน Redis หาก DB พัง
		return err
	}

	// Timeout 5 นาที ไม่จ่ายเงินให้เคลียร์ล็อกอัตโนมัติ
	go b.handleBookingTimeout(showID, seatNo, userID)

	return nil
}

// เปลี่ยนสถานะการจองเป็นเสร็จเมื่อชำระเงิน
func (b *bookingUsecase) ConfirmPayment(ctx context.Context, userID, userEmail, showID, seatNo string) error {
	// อัปเดตสถานะเป็น BOOKED ลงฐานข้อมูลหลัก
	err := b.seatRepo.UpdateSeatStatus(ctx, showID, seatNo, domain.SeatBooked, userID, userEmail)
	if err != nil {
		return err
	}

	// ปล่อย Distributed Lock ใน Redis
	_ = b.lockRepo.ReleaseLock(ctx, showID, seatNo)

	// ส่ง Event "Booking Success" เข้าสู่ Message Queue (Async Logging)
	_ = b.pubSubRepo.PublishAuditLog(ctx, &domain.AuditLog{
		Event:     "BOOKING_SUCCESS",
		Details:   fmt.Sprintf("User %s successfully booked seat %s for show %s", userID, seatNo, showID),
		UserID:    userID,
		Timestamp: time.Now(),
	})

	// ส่งแจ้งเตือนทาง LINE
	lineMsg := fmt.Sprintf("\n🎟️ จองที่นั่งสำเร็จ!\nที่นั่ง: %s\nโรงหนัง: %s\nผู้ใช้: %s", seatNo, showID, userID)
	go b.lineNotify.SendNotification(lineMsg)

	return nil
}

func (b *bookingUsecase) handleBookingTimeout(showID, seatNo, userID string) {
	// รอเวลา 5 นาที
	time.Sleep(5 * time.Minute)

	ctx := context.Background()
	// ตรวจสอบว่าที่นั่งสถานะ LOCKED อยู่หรือป่าว (ถ้าเป็น BOOKED ไปแล้วจะไม่ทำอะไร)
	seat, err := b.seatRepo.GetSeat(ctx, showID, seatNo)
	if err != nil || seat == nil {
		b.publishErrorLog("SYSTEM_ERROR", fmt.Sprintf("Failed to get seat %s for timeout check: %v", seatNo, err))
		return
	}

	// ถ้าสถานะไม่ใช่ LOCKED หรือคนล็อกไม่ใช่คนเดิม ไม่ต้องทำอะไร
	if seat.Status != domain.SeatLocked || seat.UserID != userID {
		return
	}

	// หมดเวลาชำระเงิน: ปล่อยล็อกและคืนสถานะเป็น AVAILABLE
	_ = b.seatRepo.UpdateSeatStatus(ctx, showID, seatNo, domain.SeatAvailable, "", "")
	_ = b.lockRepo.ReleaseLock(ctx, showID, seatNo)

	// ส่ง Log แจ้งเตือนหมดเวลาไป Message Queue
	_ = b.pubSubRepo.PublishAuditLog(ctx, &domain.AuditLog{
		Event:     "BOOKING_TIMEOUT",
		Details:   fmt.Sprintf("Booking timeout for user %s on seat %s", userID, seatNo),
		UserID:    userID,
		Timestamp: time.Now(),
	})
	_ = b.pubSubRepo.PublishAuditLog(ctx, &domain.AuditLog{
		Event:     "SEAT_RELEASED",
		Details:   fmt.Sprintf("Seat %s for show %s has been released due to timeout", seatNo, showID),
		Timestamp: time.Now(),
	})
}

func (b *bookingUsecase) publishErrorLog(event, details string) {
	_ = b.pubSubRepo.PublishAuditLog(context.Background(), &domain.AuditLog{
		Event:     event,
		Details:   details,
		Timestamp: time.Now(),
	})
}
