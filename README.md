# Cinema-backend

## 📂 Clean Architecture Layout
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