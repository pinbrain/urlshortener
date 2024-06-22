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

type AuthMiddleware struct {
	urlStore storage.URLStorage
}

type JWTClaims struct {
	jwt.RegisteredClaims
	UserID int
}

const (
	jwtCookieName = "shortener_jwt"
	jwtSecretKey  = "some_secret_jwt_key"
)

func NewAuthMiddleware(urlStore storage.URLStorage) AuthMiddleware {
	return AuthMiddleware{
		urlStore: urlStore,
	}
}

func (amw *AuthMiddleware) AuthenticateUser(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var jwtClaims *JWTClaims
		jwtCookie, err := r.Cookie(jwtCookieName)
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

func (amw *AuthMiddleware) RequireUser(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// В запросе должна быть кука
		_, err := r.Cookie(jwtCookieName)
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

func (amw *AuthMiddleware) createNewReqUser(ctx context.Context, w http.ResponseWriter) (*storage.User, error) {
	userData, err := amw.urlStore.CreateUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating user in store: %w", err)
	}
	var jwtString string
	jwtString, err = buildJWTString(userData.ID)
	if err != nil {
		return nil, fmt.Errorf("error creating jwt string: %w", err)
	}
	jwtCookie := &http.Cookie{
		Name:  jwtCookieName,
		Value: jwtString,
	}
	http.SetCookie(w, jwtCookie)
	return userData, nil
}

func buildJWTString(userID int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, JWTClaims{UserID: userID})
	tokenString, err := token.SignedString([]byte(jwtSecretKey))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

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

func deleteJWTCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:  jwtCookieName,
		Value: "",
	}
	cookie.MaxAge = -1
	http.SetCookie(w, cookie)
}
