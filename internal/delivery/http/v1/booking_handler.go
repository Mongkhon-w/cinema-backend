package v1

import (
	"net/http"
	"cinema-backend/internal/domain"
	"cinema-backend/internal/delivery/ws"

	"github.com/gin-gonic/gin"
)

type BookingHandler struct {
	bookingUsecase domain.BookingUsecase
	wsHub          *ws.Hub
}

func NewBookingHandler(bu domain.BookingUsecase, hub *ws.Hub) *BookingHandler {
	return &BookingHandler{
		bookingUsecase: bu,
		wsHub:          hub,
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

	// ดึง user_id จากการถอดรหัส AuthMiddleware (ซึ่งได้จาก JWT Token)
	userID := c.GetString("user_id")

	err := h.bookingUsecase.SelectSeat(c.Request.Context(), userID, input.ShowID, input.SeatNo)
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

	err := h.bookingUsecase.ConfirmPayment(c.Request.Context(), userID, input.ShowID, input.SeatNo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// ปล่อย Real-time บอกคนอื่นว่าที่นั่งนี้กลายเป็น BOOKED แล้ว
	h.wsHub.BroadcastSeatUpdate(input.ShowID, input.SeatNo, "BOOKED") 

	c.JSON(http.StatusOK, gin.H{"message": "Payment confirmed and booking completed"})
}