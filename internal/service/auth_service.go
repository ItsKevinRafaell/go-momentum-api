package service

import (
	"context"
	"errors"
	"time"

	"github.com/ItsKevinRafaell/go-momentum-api/internal/config"
	"github.com/ItsKevinRafaell/go-momentum-api/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo *repository.UserRepository
}

func NewAuthService(userRepo *repository.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

func (s *AuthService) RegisterUser(ctx context.Context, email, password string) (*repository.User, error) {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	newUser := &repository.User{
		Email:    email,
		Password: string(hashedPassword),
	}

	id, err := s.userRepo.CreateUser(ctx, newUser)
	if err != nil {
		return nil, err
	}

	newUser.ID = id
	return newUser, nil
}

func (s *AuthService) LoginUser(ctx context.Context, email, password string) (string, error) {
	// 1. Cari user berdasarkan email
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		// Jika user tidak ditemukan, kembalikan error yang jelas
		return "", errors.New("invalid credentials")
	}

	// 2. Bandingkan password yang diberikan dengan hash di database
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		// Jika password salah, bcrypt akan mengembalikan error
		return "", errors.New("invalid credentials")
	}

	// 3. Jika berhasil, buat JWT Token
	claims := jwt.MapClaims{
		"sub": user.ID,                                       // Subject (identitas user)
		"exp": time.Now().Add(time.Hour * 24).Unix(),         // Waktu kedaluwarsa (24 jam)
		"iat": time.Now().Unix(),                             // Waktu token dibuat
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Ambil secret key dari .env
	jwtSecret := config.Get("JWT_SECRET_KEY")
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
    user, err := s.userRepo.GetUserByID(ctx, userID)
    if err != nil {
        return errors.New("user not found")
    }

    // Bandingkan password lama yang diinput dengan yang ada di DB
    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
        return errors.New("invalid old password")
    }

    // Hash password baru
    newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
    if err != nil {
        return err
    }

    // Simpan hash password baru
    return s.userRepo.UpdatePasswordHash(ctx, userID, string(newHashedPassword))
}

func (s *AuthService) GetUserByID(ctx context.Context, userID string) (*repository.User, error) {
	// Service ini hanya meneruskan panggilan ke repository.
	return s.userRepo.GetUserByID(ctx, userID)
}