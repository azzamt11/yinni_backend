package middleware

import (
	"context"
	"errors"
)

// UserClaims contains the user information from JWT
type UserClaims struct {
	UserID  int64    `json:"user_id"`
	Email   string   `json:"email,omitempty"`
	IsAdmin bool     `json:"is_admin,omitempty"`
	Roles   []string `json:"roles,omitempty"`
}

// ExtractClaimsFromContext extracts user claims from context
func ExtractClaimsFromContext(ctx context.Context) (*UserClaims, error) {
	// Try to get from jwt_claims
	if claims, ok := ctx.Value("jwt_claims").(*UserClaims); ok {
		return claims, nil
	}

	// Fallback to user_id
	if userID, ok := ctx.Value("user_id").(int64); ok {
		return &UserClaims{UserID: userID}, nil
	}

	return nil, errors.New("no user claims in context")
}
