// Модуль аутентификации клиентских запросов.
package auth

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

// Claims структура клайма
type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

// tokenExp время жизни токена
const tokenExp = time.Hour * 3

// cookieName название куки
const cookieName = "jwt-token"

// UserIDKey ID пользователя в качестве ключа
const UserIDKey = "userID"

// ErrTokenNotValid ошибка - токен не валиден
var ErrTokenNotValid = errors.New("token is not valid")

// ErrNoUserInToken ошибка - в токене отсутствует информацию по пользователю
var ErrNoUserInToken = errors.New("no user data in token")

// BuildJWTString метод по созданию JWT токена в виде строки
func BuildJWTString(seed string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExp)),
		},
		UserID: uuid.New().String(),
	})

	tokenString, err := token.SignedString([]byte(seed))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GetUserID метод по получению пользователя по ID
func GetUserID(tokenString string, seed string) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			return []byte(seed), nil
		})
	if err != nil {
		if !token.Valid {
			return "", ErrTokenNotValid
		} else {
			return "", errors.New("parsing error")
		}
	}

	if claims.UserID == "" {
		return "", ErrNoUserInToken
	}

	return claims.UserID, nil
}

// AuthMiddleware метод для установки куки и ID пользователя
func AuthMiddleware(seed string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(cookieName)
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				token, err := BuildJWTString(seed)
				if err != nil {
					log.Printf("Error building JWT string: %v", err)
					c.Writer.WriteHeader(http.StatusInternalServerError)
					return
				}
				c.SetCookie(cookieName, token, 3600*24*30, "", "", false, true)
				cookie = token
			} else {
				log.Printf("Error reading cookie[%v]: %v", cookieName, err)
				c.Writer.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		userID, err := GetUserID(cookie, seed)
		if err != nil {
			if errors.Is(err, ErrNoUserInToken) {
				c.Writer.WriteHeader(http.StatusUnauthorized)
				return
			}
			if errors.Is(err, ErrTokenNotValid) {
				token, err := BuildJWTString(seed)
				if err != nil {
					log.Printf("Error building JWT string: %v", err)
					c.Writer.WriteHeader(http.StatusInternalServerError)
					return
				}
				userID, err = GetUserID(token, seed)
				if err != nil {
					log.Printf("Revalidate error user id from renewed token: %v", err)
					c.Writer.WriteHeader(http.StatusInternalServerError)
					return
				}
				c.SetCookie(cookieName, token, 3600*24*30, "", "", false, true)
			}
		}

		c.Set(UserIDKey, userID)
		c.Next()
	}
}
