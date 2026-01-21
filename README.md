# G4 Drivers Sign-Up API

Backend service for the G4 Car Service driver registration platform. Built with Go, optimized for performance, security, and scalability.

## 🚀 Technologies

- **Language**: Go (Golang) 1.23
- **Database**: PostgreSQL (via Supabase)
- **Auth**: JWT (Supabase Auth integration)
- **Storage**: S3-compatible Object Storage (Supabase Storage)
- **Documentation**: Swagger / OpenAPI
- **Deployment**: Docker + Render

## ✨ Features

- **Authentication**: Secure JWT validation middleware.
- **Driver Registration**: Multipart form handling for profile/vehicle data and document uploads.
- **File Storage**: Direct upload integration for images and documents.
- **Email Notifications**: Transactional emails (Welcome, Security Alerts) via SMTP (Gmail).
- **Security**:
  - **Rate Limiting**: Token bucket algorithm to prevent abuse (Strict limits for uploads).
  - **CORS & Headers**: Configured for security best practices.
  - **Input Validation**: Strict typing and validation for all forms.
- **Optimizations**: SQL indices and pagination for high-performance dashboard queries.
- **Observability**: Structured JSON logging (`log/slog`).

## 🛠️ Setup & Installation

### Prerequisites
- Go 1.23+
- PostgreSQL / Supabase Project
- SMTP Credentials

### 1. Clone the repository
```bash
git clone https://github.com/GalahadDev/g4-drivers-sign-up-backend
cd g4-drivers-sign-up
```

### 2. Environment Variables
Create a `.env` file in the root directory:
```bash
PORT=8080

# Supabase Auth & DB
SUPABASE_URL="your_supabase_url"
SUPABASE_KEY="your_supabase_anon_key"
SUPABASE_DB_HOST="your_db_host"
SUPABASE_DB_PORT="5432"
SUPABASE_DB_NAME="postgres"
SUPABASE_DB_USER="postgres"
SUPABASE_DB_PASSWORD="your_db_password"

# SMTP (Email)
SMTP_HOST="smtp.gmail.com"
SMTP_PORT="587"
SMTP_EMAIL="your_email@gmail.com"
SMTP_PASSWORD="your_app_password"
```

### 3. Run Locally
```bash
go mod tidy
go run main.go
```
The server will start at `http://localhost:8080`.

### 4. Run with Docker
```bash
docker build -t g4-app .
docker run -p 8080:8080 --env-file .env g4-app
```

## 📖 API Documentation

Interactive API documentation (Swagger UI) is available at:
`http://localhost:8080/swagger/index.html`

Use this to explore endpoints, request formats, and test the API.

## 🚀 Deployment (Render)

This project includes a `render.yaml` Blueprint for automatic deployment on Render.

1. Push your code to GitHub.
2. Create a new **Blueprint Instance** in Render.
3. Connect your repository.
4. Render will automatically detect the Docker configuration and deploy the service.

## 📂 Project Structure

- `/api/handlers`: HTTP Controllers (Drivers, Users, Admin, Dashboard).
- `/api/middleware`: Auth, Logging, Rate Limiting, CORS.
- `/api/database`: Database connection and SQL migrations.
- `/api/services`: External services (Email, Storage).
- `/api/config`: Configuration loader.
- `/docs`: Generated Swagger documentation.
