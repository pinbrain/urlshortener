package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v4"

	appCtx "github.com/pinbrain/urlshortener/internal/context"
	"github.com/pinbrain/urlshortener/internal/logger"
	"github.com/pinbrain/urlshortener/internal/storage"
)

// AuthMiddleware описывает структуру обработчика для авторизации и аутентификации.
type AuthMiddleware struct {
	urlStore storage.URLStorage // Хранилище приложения
}

// JWTClaims описывает структуру JWT токена.
type JWTClaims struct {
	jwt.RegisteredClaims     // Типовые параметры JWT токена
	UserID               int // ID пользователя
}

// Константы для работы с jwt.
const (
	JWTCookieName = "shortener_jwt"       // Название cookie в которой хранится jwt токен
	jwtSecretKey  = "some_secret_jwt_key" // Ключ для подписи jwt токена
)

// NewAuthMiddleware создает обработчик авторизации и аутентификации.
func NewAuthMiddleware(urlStore storage.URLStorage) AuthMiddleware {
	return AuthMiddleware{
		urlStore: urlStore,
	}
}

// AuthenticateUser аутентифицирует пользователя запроса.
func (amw *AuthMiddleware) AuthenticateUser(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var jwtClaims *JWTClaims
		jwtCookie, err := r.Cookie(JWTCookieName)
		if err == nil {
			jwtClaims, err = getJWTClaims(jwtCookie.Value)
			if err != nil {
				logger.Log.Errorw("Error parsing jwt with claims", "err", err)
				jwtClaims = nil
			}
		}

		var userData *storage.User

		// Либо не было куки, либо она оказалась не валидной
		if jwtClaims == nil {
			userData, err = amw.createNewReqUser(r.Context(), w)
			if err != nil {
				logger.Log.Errorw("Error creating new user with jwt cookie for request", "err", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
		} else {
			userData, err = amw.urlStore.GetUser(r.Context(), jwtClaims.UserID)
			if err != nil && !errors.Is(err, storage.ErrNoData) {
				logger.Log.Errorw("Error getting user data by jwt claims from store", "err", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
		}

		if userData != nil {
			user := &appCtx.CtxUser{
				ID: userData.ID,
			}
			ctx := r.Context()
			ctx = appCtx.CtxWithUser(ctx, user)
			r = r.WithContext(ctx)
		} else {
			// Юзера из куки нет в БД, поэтому удаляем куку
			deleteJWTCookie(w)
		}

		h.ServeHTTP(w, r)
	})
}

// RequireUser проверяет что пользователь авторизован.
// В противном случае прерывает обработку запроса и возвращает ошибку Unauthorized.
func (amw *AuthMiddleware) RequireUser(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// В запросе должна быть кука
		_, err := r.Cookie(JWTCookieName)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// В куке должны быть данные существующего пользователя
		user := appCtx.GetCtxUser(r.Context())
		if user == nil || user.ID <= 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		h.ServeHTTP(w, r)
	})
}

// createNewReqUser создает нового пользователя.
// Добавляет его данные в контекст запроса и добавляет cookie с jwt токеном.
func (amw *AuthMiddleware) createNewReqUser(ctx context.Context, w http.ResponseWriter) (*storage.User, error) {
	userData, err := amw.urlStore.CreateUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating user in store: %w", err)
	}
	var jwtString string
	jwtString, err = BuildJWTString(userData.ID)
	if err != nil {
		return nil, fmt.Errorf("error creating jwt string: %w", err)
	}
	jwtCookie := &http.Cookie{
		Name:  JWTCookieName,
		Value: jwtString,
	}
	http.SetCookie(w, jwtCookie)
	return userData, nil
}

// BuildJWTString формирует jwt токен с переданными данными.
func BuildJWTString(userID int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, JWTClaims{UserID: userID})
	tokenString, err := token.SignedString([]byte(jwtSecretKey))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// getJWTClaims возвращает данные из jwt токена, проверяя его валидность.
func getJWTClaims(tokenString string) (*JWTClaims, error) {
	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(jwtSecretKey), nil
		})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// deleteJWTCookie удаляет cookie с jwt токеном.
func deleteJWTCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:  JWTCookieName,
		Value: "",
	}
	cookie.MaxAge = -1
	http.SetCookie(w, cookie)
}
