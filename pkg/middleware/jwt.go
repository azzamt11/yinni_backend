package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/golang-jwt/jwt/v5"
)

// Claims is the JWT claims structure
type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

// JWT middleware validates JWT tokens
func JWT(secret string) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tr, ok := transport.FromServerContext(ctx)
			if ok {
				println("JWT middleware: request path =", tr.Operation())
			} else {
				println("JWT middleware: no transport context")
				return nil, errors.New("missing transport context")
			}

			auth := tr.RequestHeader().Get("Authorization")
			if auth == "" {
				println("JWT middleware: invalid token")
				return nil, errors.New("missing authorization header")
			}

			tokenStr := strings.TrimPrefix(auth, "Bearer ")

			// Parse with standard claims
			token, err := jwt.ParseWithClaims(
				tokenStr,
				&Claims{},
				func(token *jwt.Token) (interface{}, error) {
					return []byte(secret), nil
				},
			)
			if err != nil || !token.Valid {
				println("JWT middleware: invalid token")
				return nil, errors.New("invalid token")
			}

			// Get basic claims
			claims := token.Claims.(*Claims)

			// Store user_id in context (for backward compatibility)
			ctx = context.WithValue(ctx, "user_id", claims.UserID)

			// Parse additional claims for admin check
			userClaims := &UserClaims{
				UserID: claims.UserID,
			}

			// Parse token again to get map claims
			token2, _ := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			})

			if mapClaims, ok := token2.Claims.(jwt.MapClaims); ok {
				// Parse is_admin
				if isAdmin, ok := mapClaims["is_admin"].(bool); ok {
					userClaims.IsAdmin = isAdmin
				}

				// Parse email
				if email, ok := mapClaims["email"].(string); ok {
					userClaims.Email = email
				}

				// Parse roles
				if rolesRaw, ok := mapClaims["roles"]; ok {
					if rolesJSON, err := json.Marshal(rolesRaw); err == nil {
						json.Unmarshal(rolesJSON, &userClaims.Roles)
					}
				}
			}

			// Store full claims in context
			ctx = context.WithValue(ctx, "jwt_claims", userClaims)

			println("JWT middleware: token validated, calling handler")
			return handler(ctx, req)
		}
	}
}
