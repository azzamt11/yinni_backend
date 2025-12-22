package middleware

import (
	"context"
	"errors"
	"strings"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func JWT(secret string) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, errors.New("missing transport context")
			}

			auth := tr.RequestHeader().Get("Authorization")
			if auth == "" {
				return nil, errors.New("missing authorization header")
			}

			tokenStr := strings.TrimPrefix(auth, "Bearer ")

			token, err := jwt.ParseWithClaims(
				tokenStr,
				&Claims{},
				func(token *jwt.Token) (interface{}, error) {
					return []byte(secret), nil
				},
			)
			if err != nil || !token.Valid {
				return nil, errors.New("invalid token")
			}

			claims := token.Claims.(*Claims)
			ctx = context.WithValue(ctx, "user_id", claims.UserID)

			return handler(ctx, req)
		}
	}
}
