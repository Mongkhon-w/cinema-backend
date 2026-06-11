package v1

import (
	"cinema-backend/internal/delivery/ws"
	"cinema-backend/internal/domain"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

type BookingHandler struct {
	bookingUsecase domain.BookingUsecase
	wsHub          *ws.Hub
	bookingsColl   *mongo.Collection
	auditLogsColl  *mongo.Collection
}

func NewBookingHandler(bu domain.BookingUsecase, hub *ws.Hub, bookingsColl, auditLogsColl *mongo.Collection) *BookingHandler {
	return &BookingHandler{
		bookingUsecase: bu,
		wsHub:          hub,
		bookingsColl:   bookingsColl,
		auditLogsColl:  auditLogsColl,
	}
}

type SelectSeatInput struct {
	ShowID string `json:"show_id" binding:"required"`
	SeatNo string `json:"seat_no" binding:"required"`
}

// API สำหรับกดเลือกและล็อกที่นั่ง
func (h *BookingHandler) SelectSeat(c *gin.Context) {
	var input SelectSeatInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ดึง user_id จากการถอดรหัส
	userID := c.GetString("user_id")
	userEmail := c.GetString("email")

	err := h.bookingUsecase.SelectSeat(c.Request.Context(), userID, userEmail, input.ShowID, input.SeatNo)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	// ปล่อยสัญญาณ Real-time บอกคนอื่นว่าที่นั่งนี้ติดสถานะ LOCKED แล้ว
	h.wsHub.BroadcastSeatUpdate(input.ShowID, input.SeatNo, "LOCKED")

	c.JSON(http.StatusOK, gin.H{"message": "Seat successfully locked for 5 minutes"})
}

// API สำหรับจำลองการจ่ายเงินเสร็จสิ้น
func (h *BookingHandler) ConfirmPayment(c *gin.Context) {
	var input SelectSeatInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")
	userEmail := c.GetString("email")

	err := h.bookingUsecase.ConfirmPayment(c.Request.Context(), userID, userEmail, input.ShowID, input.SeatNo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// บันทึก Booking ลง MongoDB
	booking := map[string]interface{}{
		"user_id":    userID,
		"user_email": userEmail,
		"show_id":    input.ShowID,
		"seat_no":    input.SeatNo,
		"status":     "BOOKED",
		"created_at": time.Now().Format("2006-01-02 15:04:05"),
	}
	_, _ = h.bookingsColl.InsertOne(c.Request.Context(), booking)

	// บันทึก Audit Log
	auditLog := map[string]interface{}{
		"event":     "BOOKING_SUCCESS",
		"user_id":   userID,
		"seat_no":   input.SeatNo,
		"details":   "Payment confirmed for seat " + input.SeatNo,
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
	}
	_, _ = h.auditLogsColl.InsertOne(c.Request.Context(), auditLog)

	// ปล่อย Real-time บอกคนอื่นว่าที่นั่งนี้กลายเป็น BOOKED แล้ว
	h.wsHub.BroadcastSeatUpdate(input.ShowID, input.SeatNo, "BOOKED")

	c.JSON(http.StatusOK, gin.H{"message": "Payment confirmed and booking completed"})
}
