package auth

import (
	"context"
	"log"
	"time"
	"os"

	"github.com/nais2008/final_project_go_yandex/internal/models"
	"github.com/nais2008/final_project_go_yandex/internal/db"
	"github.com/nais2008/final_project_go_yandex/proto"
	"github.com/nais2008/final_project_go_yandex/internal/utils"

	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var jwtSecret = os.Getenv("JWT_TOKEN")

// AuthServer ...
type AuthServer struct{}

// Register ...
func (s *AuthServer) Register(ctx context.Context, req *proto.RegisterRequest) (*proto.RegisterRespnise, error) {
	dbConn := db.ConnectDB()

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to hash password")
	}

	user := models.User{
		Username: req.Username,
		Email: req.Email,
		Password: hashedPassword,
	}

	if err := dbConn.Create(&user).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to create user")
	}

	return &proto.RegisterRespnise{UserId: int64(user.ID)}, nil
}

// Login ...
func (s *AuthServer) Login(ctx context.Context, req *proto.LoginRequest) (*proto.LoginResponse, error) {
	dbConn := db.ConnectDB()

	var user models.User
	if err := dbConn.Where("email = ? OR username = ?", req.Login, req.Login).First(&user).Error; err != nil {
		return nil, status.Error(codes.NotFound, "Invalid login")
	}

	if err := utils.ComparePasswords(user.Password, req.Password); err != nil {
		return nil, status.Error(codes.Unauthenticated, "Invalid password")
	}

	// Generate JWT
	token, err := generateJWT(user)
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to generate JWT")
	}

	return &proto.LoginResponse{Token: token}, nil
}

// generateJWT ...
func generateJWT(user models.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}
