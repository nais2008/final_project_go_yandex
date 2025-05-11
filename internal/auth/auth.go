package auth

import (
	context "context"
	"log"
	"time"
	"os"

	"gorm.io/gorm"
	"github.com/golang-jwt/jwt/v4"

	pb "github.com/nais2008/final_project_go_yandex/proto/gen/go/sso"
	"github.com/nais2008/final_project_go_yandex/internal/utils"
	"github.com/nais2008/final_project_go_yandex/internal/models"
)

// ServiceServer  ...
type ServiceServer  struct {
	pb.UnimplementedAuthServiceServer
	DB *gorm.DB
}

// Register ...
func (s *ServiceServer ) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	log.Println("Register request received:", req)

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := models.User{
		Username: req.Username,
		Email: req.Email,
		Password: hashedPassword,
	}

	if err := s.DB.Create(&user).Error; err != nil {
		return nil, err
	}

	return &pb.RegisterResponse{UserId: int64(user.ID)}, nil
}

// Login ...
func (s *ServiceServer ) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	log.Println("Login request received:", req)

	var user models.User
	if err := s.DB.Where("username = ? OR email = ?", req.Login, req.Login).First(&user).Error; err != nil {
		return nil, err
	}
	if err := utils.ComparePasswords(user.Password, req.Password); err != nil {
		return nil, err
	}

	secret := os.Getenv("JWT_TOKEN")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"user_id": user.ID,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return nil, err
	}

	return &pb.LoginResponse{Token: tokenString}, nil
}

// NewAuthServiceServer ...
func NewAuthServiceServer(db *gorm.DB) *ServiceServer  {
	return &ServiceServer {DB: db}
}
