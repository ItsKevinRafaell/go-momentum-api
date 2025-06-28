# Project: Momentum - Backend API

Backend API untuk aplikasi AI Productivity Coach yang dibangun menggunakan Go (Golang).

---

## Tentang Proyek

Project Momentum adalah backend API yang tangguh dan aman, dirancang untuk menjadi otak dari sebuah aplikasi asisten produktivitas cerdas. Aplikasi ini membantu pengguna mengatasi penundaan dan membangun kebiasaan positif dengan menyediakan jadwal harian adaptif yang didukung oleh AI, serta memberikan pengguna kontrol penuh untuk menyesuaikan rencana mereka.

## Instalasi & Menjalankan Lokal

Untuk menjalankan proyek ini di lingkungan lokal Anda, ikuti langkah-langkah berikut.

### 1. Prasyarat

- [Go](https://go.dev/doc/install) (versi 1.18 atau lebih baru)
- [PostgreSQL](https://www.postgresql.org/download/)
- [migrate CLI](https://github.com/golang-migrate/migrate/releases)

### 2. Setup Proyek

1.  **Clone repositori ini:**

    ```bash
    git clone https://github.com/ItsKevinRafaell/go-momentum-api.git
    cd project-momentum
    ```

2.  **Konfigurasi Environment:**
    Buat file `.env` di root proyek dengan menyalin dari contoh.

    ```bash
    cp .env.example .env
    ```

    Isi file `.env` dengan kredensial database Anda dan rahasia JWT. Contoh isi file `.env`:

    ```env
    # Konfigurasi Port Aplikasi
    API_PORT=8080

    # Konfigurasi Database PostgreSQL
    DATABASE_URL="postgresql://postgres:[password]@localhost:5432/momentum_db?sslmode=disable"

    # Konfigurasi JWT
    JWT_SECRET_KEY="ganti-dengan-kunci-rahasia-acak-yang-sangat-panjang"
    JWT_EXPIRATION_IN_HOURS=24
    ```

3.  **Instalasi Dependencies:**

    ```bash
    go mod tidy
    ```

4.  **Migrasi Database:**
    Jalankan migrasi untuk membuat semua tabel yang diperlukan. Ganti `[DATABASE_URL]` dengan URL dari file `.env` Anda.

    ```bash
    migrate -path internal/database/migration -database "[DATABASE_URL]" up
    ```

5.  **Jalankan Server:**
    ```bash
    go run ./cmd/server/main.go
    ```
    Server akan berjalan di `http://localhost:8080` (atau port yang Anda tentukan di `.env`).

---

## Penggunaan API (Dokumentasi Endpoint)

Semua endpoint berada di bawah base URL `http://localhost:8080/api`. Endpoint yang terproteksi memerlukan header `Authorization: Bearer <token>`.

### Modul Autentikasi

#### 1. Registrasi Pengguna Baru

- `POST /auth/register`

  Mendaftarkan pengguna baru. Tidak memerlukan autentikasi.

  **Request Body:**

  ```json
  {
    "email": "user.baru@example.com",
    "password": "password123"
  }
  ```

  **Success Response (`201 Created`):**

  ```json
  { "message": "User registered successfully" }
  ```

  **Error Responses:** `400 Bad Request`, `409 Conflict`.

#### 2. Login Pengguna

- `POST /auth/login`

  Memverifikasi kredensial dan mengembalikan JWT.

  **Request Body:**

  ```json
  {
    "email": "user.baru@example.com",
    "password": "password123"
  }
  ```

  **Success Response (`200 OK`):**

  ```json
  { "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." }
  ```

  **Error Response:** `401 Unauthorized`.

---

### Modul Tujuan & Roadmap

Memerlukan autentikasi.

#### 1. Membuat Tujuan & Roadmap Baru

- `POST /goals`

  Membuat tujuan utama baru dan memicu AI untuk membuatkan _roadmap_.

  **Request Body:**

  ```json
  {
    "description": "Menjadi Full-Stack Developer dalam 1 tahun"
  }
  ```

  **Success Response (`201 Created`):** Mengembalikan objek `goal` dan `steps`.
  **Error Responses:** `400 Bad Request`, `409 Conflict`.

#### 2. Mengambil Tujuan & Roadmap Aktif

- `GET /goals/active`

  Mengambil detail tujuan yang sedang aktif.

  **Success Response (`200 OK`):** Mengembalikan objek `goal` dan `steps`.
  **Error Response:** `404 Not Found`.

---

### Modul Jadwal & Tugas Harian

Memerlukan autentikasi.

#### 1. Mendapatkan Jadwal Hari Ini

- `GET /schedule/today`

  Mengambil atau, jika belum ada, membuat jadwal tugas untuk hari ini.

  **Success Response (`200 OK`):** Mengembalikan array dari objek `task`.

#### 2. Menambah Tugas Manual

- `POST /tasks`

  **Request Body:**

  ```json
  {
    "title": "Tugas tambahan: Beli kopi",
    "deadline": "2025-06-29T17:00:00Z" // Opsional
  }
  ```

  **Success Response (`201 Created`):** Mengembalikan objek `task` yang baru dibuat.

#### 3. Mengedit Judul Tugas

- `PUT /tasks/{taskId}`

  **Request Body:** `{"title": "Judul tugas yang baru"}`
  **Success Response (`200 OK`):** `{"message": "Task title updated successfully"}`

#### 4. Mengubah Status Tugas

- `PUT /tasks/{taskId}/status`

  **Request Body:** `{"status": "completed"}` atau `{"status": "pending"}`
  **Success Response (`200 OK`):** `{"message": "Task status updated"}`

#### 5. Mengubah Deadline Tugas

- `PUT /tasks/{taskId}/deadline`

  **Request Body:** `{"deadline": "2025-06-29T17:00:00Z"}`
  **Success Response (`200 OK`):** `{"message": "Task deadline updated successfully"}`

#### 6. Menghapus Tugas

- `DELETE /tasks/{taskId}`

  **Success Response (`204 No Content`):** Tidak ada body respons.
  **Error Response:** `404 Not Found`.

---

### Modul Review

Memerlukan autentikasi.

#### 1. Melakukan Review Harian

- `POST /schedule/review`

  Memfinalisasi jadwal hari itu dan mendapatkan ringkasan serta feedback dari AI.

  **Success Response (`200 OK`):**

  ```json
  {
    "summary": [
      { "status": "completed", "count": 1 },
      { "status": "missed", "count": 1 }
    ],
    "ai_feedback": "Progres yang bagus dengan 1 tugas selesai!..."
  }
  ```

---

## Berkontribusi (Contributing)

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## Lisensi (License)

[MIT](https://choosealicense.com/licenses/mit/)
