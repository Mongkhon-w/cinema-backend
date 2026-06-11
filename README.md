# Cinema-backend

## 📂 Clean Architecture Layout
```
cinema-backend/
├── cmd/
│   └── api/
│       └── main.go             
├── config/
│   └── config.go           
├── internal/
│   ├── domain/             # Entities, Interfaces และ Business Logic Models
│   │   ├── auth.go
│   │   ├── seat.go
│   │   ├── booking.go
│   │   └── audit_log.go
│   ├── repository/         # Database Access Layer (MongoDB / Redis)
│   │   ├── mongo_repo/
│   │   └── redis_repo/
│   ├── usecase/            # Business Logic Layer
│   └── delivery/           # Transport Layer (HTTP Controllers, Middleware, WebSockets)
│       ├── http/
│       │   ├── middleware/
│       │   └── v1/
│       └── ws/
├── .env
└── Dockerfile
```

### 📁 สร้างโครงสร้างโฟลเดอร์
```bash
mkdir -p cinema-backend/{cmd/api,config,internal/{domain,repository/{mongo_repo,redis_repo},usecase,delivery/{http/{middleware,v1},ws}},.github/workflows}

```

### 📄 สร้างไฟล์เปล่า
```bash
touch cinema-backend/cmd/api/main.go \
      cinema-backend/config/config.go \
      cinema-backend/internal/delivery/http/middleware/auth_middleware.go \
      cinema-backend/internal/delivery/http/v1/auth_handler.go \
      cinema-backend/internal/delivery/http/v1/booking_handler.go \
      cinema-backend/internal/delivery/ws/hub.go \
      cinema-backend/internal/domain/auth.go \
      cinema-backend/internal/domain/seat.go \
      cinema-backend/internal/domain/booking.go \
      cinema-backend/internal/domain/audit_log.go \
      cinema-backend/internal/repository/redis_repo/lock.go \
      cinema-backend/internal/repository/redis_repo/pubsub.go \
      cinema-backend/internal/usecase/booking_usecase.go \
      cinema-backend/Dockerfile \
      cinema-backend/.env \
      cinema-backend/.gitignore \ 
      cinema-backend/docker-compose.yml \
```

## 🧭 ทดสอบระบบหลังบ้าน (Testing)
1. ทดสอบเส้นทางเข้าสู่ระบบ (Google OAuth 2.0)
เปิด Browser (Chrome/Edge) แล้วไปที่ URL:
```bash
http://localhost:8080/api/v1/auth/google
```

2. หน้าต่างเฝ้าดูผังที่นั่งแบบ Real-time (WebSocket)
เปิดการเชื่อมต่อค้างไว้เพื่อดูว่าเวลามีคนกดจอง
- วิธีทดสอบ: Postman ให้เลือกสร้าง Request ประเภท WebSocket Request แล้วกรอก URL:
```bash
ws://localhost:8080/api/v1/ws
```
- ผลลัพธ์ที่คาดหวัง: กดปุ่ม Connect แล้วสถานะต้องขึ้นว่า Connected สำเร็จ และเปิดหน้าต่างนี้ค้างไว้เพื่อรอรับสัญญานสถานะที่นั่ง

3. ทดสอบระบบล็อกที่นั่ง 5 นาที
- วิธีทดสอบ: Postman สร้าง POST Request ไปที่ URL:
```bash
http://localhost:8080/api/v1/seats/lock
```
- Headers: (เนื่องจากเรามี Middleware อยู่):
    - Authorization: Bearer <เอา_access_token_ที่ได้จากข้อ_1_มาใส่>
- Body (JSON):
```bash
{
  "show_id": "movie-avengers-123",
  "seat_no": "H-10"
}
```
- ผลลัพธ์ที่คาดหวัง: 
    - ฝั่ง HTTP Response จะต้องตอบกลับมาว่า "Seat successfully locked for 5 minutes" 
    - จุดสำคัญ: ให้สลับไปดูหน้าต่าง WebSocket มันจะมีข้อความวิ่งเข้ามาหาอัตโนมัติว่า:
```bash
{"show_id":"movie-avengers-123","seat_no":"H-10","status":"LOCKED"}
```

4. ทดสอบความปลอดภัยการกดซ้อน (Concurrency & Double Booking)
- วิธีทดสอบ: ในขณะที่เวลา 5 นาทียังไม่หมด ให้เปิดแท็บใหม่ใน Postman (จำลองเป็น User คนอื่น) แล้วส่งคำขอ POST ไปล็อกที่นั่งเดิม (H-10) ของรอบฉายเดิมซ้ำอีกครั้ง
- ผลลัพธ์ที่คาดหวัง: ระบบต้องปฏิเสธคำขออันที่สองทันที โดยตอบกลับเป็นสถานะ 409 Conflict พร้อมข้อความแจ้งเตือนว่า "seat is already locked or booked by another user" เพื่อจะไม่มีการเกิด Double Booking 

5. ทดสอบยืนยันการชำระเงิน
- วิธีทดสอบ: ส่งคำขอ POST ไปที่ URL:
```bash
http://localhost:8080/api/v1/seats/confirm
```
- Body (JSON):
```bash
{
  "show_id": "movie-avengers-123",
  "seat_no": "H-10"
}
```
- ผลลัพธ์ที่คาดหวัง: 
    - สถานะที่นั่งจะถูกเปลี่ยนเป็น "BOOKED" บนฐานข้อมูลหลัก 
    - หน้าจอ WebSocket จะได้รับข้อความแจ้งสถานะอัปเดตเป็น "BOOKED" 
    - ระบบหลังบ้านจะส่ง (Message Queue) ทำการบันทึกประวัติเหตุการณ์ "Booking Success" ลง MongoDB แบบ (Async Logging) 

6. ตรวจสอบหน้า Dashboard ผู้ดูแลระบบ (Admin Dashboard & Audit Logs)
- Api ตรวจสอบ Dashboard (มีระบบ Filter):
```bash
GET http://localhost:8080/api/v1/admin/dashboard?movie=movie-avengers-123
```
(สามารถส่งตัวกรองเป็น movie, date, หรือ user เพื่อตรวจสอบการทำงาน)
- Api เรียกดู Log ทั้งหมดที่บันทึกผ่าน Queue ลง MongoDB:
```bash
GET http://localhost:8080/api/v1/admin/audit-logs
```