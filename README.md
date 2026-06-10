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

### 📄 สร้างไฟล์เปล่าทั้งหมด
```bash
touch cinema-backend/cmd/api/main.go \
      cinema-backend/config/config.go \
      cinema-backend/internal/domain/auth.go \
      cinema-backend/internal/domain/seat.go \
      cinema-backend/internal/domain/booking.go \
      cinema-backend/internal/domain/audit_log.go \
      cinema-backend/Dockerfile \
      cinema-backend/.env \
      cinema-backend/.gitignore \ 
      cinema-backend/.github/workflows/ci.yml
```