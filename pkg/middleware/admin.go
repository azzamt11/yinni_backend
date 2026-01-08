package middleware

import (
	"context"
	"errors"
	"strings"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/golang-jwt/jwt/v5"
)

// Admin middleware checks if the user is an admin
func Admin(secret string) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, errors.New("missing transport context")
			}

			// Get the request path
			operation := tr.Operation()

			// Only check admin for admin endpoints
			if !strings.Contains(operation, "/admin") {
				return handler(ctx, req)
			}

			auth := tr.RequestHeader().Get("Authorization")
			if auth == "" {
				return nil, errors.New("missing authorization header")
			}

			tokenStr := strings.TrimPrefix(auth, "Bearer ")

			// Parse token
			token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				return nil, errors.New("invalid token")
			}

			// Check if user is admin
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return nil, errors.New("invalid token claims")
			}

			// Check is_admin field
			if isAdmin, ok := claims["is_admin"].(bool); ok && isAdmin {
				return handler(ctx, req)
			}

			// Check roles array
			if roles, ok := claims["roles"].([]interface{}); ok {
				for _, role := range roles {
					if roleStr, ok := role.(string); ok && roleStr == "admin" {
						return handler(ctx, req)
					}
				}
			}

			// Check user_id == 1 (fallback for first admin)
			if userID, ok := claims["user_id"].(float64); ok && int64(userID) == 1 {
				return handler(ctx, req)
			}

			return nil, errors.New("admin access required")
		}
	}
}
