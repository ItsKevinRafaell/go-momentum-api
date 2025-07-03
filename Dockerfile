# Gunakan versi Go yang sesuai dengan go.mod Anda
ARG GO_VERSION=1.23
FROM golang:${GO_VERSION}-bookworm as builder

WORKDIR /usr/src/app

# Salin dan unduh dependensi terlebih dahulu untuk caching yang lebih baik
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Salin sisa source code
COPY . .

# Build aplikasi Go
RUN go build -v -o /run-app ./cmd/server


# --- Final Stage ---
# Mulai dari gambar Debian yang bersih
FROM debian:bookworm-slim

# --- PERBAIKAN FINAL DI SINI ---
# Perbarui daftar paket dan instal paket sertifikat secara eksplisit.
# Lalu bersihkan cache untuk menjaga ukuran image tetap kecil.
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Salin HANYA binary aplikasi yang sudah di-compile dari stage 'builder'
COPY --from=builder /run-app /usr/local/bin/

# Set perintah untuk menjalankan aplikasi saat container dimulai
CMD ["run-app"]