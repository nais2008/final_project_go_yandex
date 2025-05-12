package utils

import (
	"errors"
	"time"
	"os"

	"github.com/golang-jwt/jwt/v4"
)


// Claims ...
type Claims struct{
	Login string
	jwt.RegisteredClaims
}


var jwtSecretKey = []byte(os.Getenv("JWT_TOKEN"))

// GenerateJWT ...
func GenerateJWT(login string) (string, error){
	claim := Claims{
		Login: login,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claim,
	)

	tokenString, err := token.SignedString(jwtSecretKey)

	if err != nil{
		return "", err
	}

	return tokenString, nil
}

// VerifyJWT ...
func VerifyJWT(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecretKey, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

