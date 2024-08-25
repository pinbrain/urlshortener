// Package context предоставляет возможность хранить в контексте данные запроса и получать к ним доступ.
package context

import "context"

type ctxKey string

// CtxUser определяет структуру данных пользователя запроса, хранящуюся в контексте.
type CtxUser struct {
	ID int
}

// Ключ контекста (по которому сохраняются и достаются данные).
const userCtxKey ctxKey = "user"

// CtxWithUser добавляет в контекст данные пользователя запроса
// (возвращает копию переданного контекста с данными пользователя).
func CtxWithUser(ctx context.Context, user *CtxUser) context.Context {
	return context.WithValue(ctx, userCtxKey, user)
}

// GetCtxUser возвращает данные пользователя из переданного контекста.
func GetCtxUser(ctx context.Context) *CtxUser {
	val := ctx.Value(userCtxKey)
	user, ok := val.(*CtxUser)
	if !ok {
		return nil
	}
	return user
}
