package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var jwtSecret = []byte("your-secret-key-change-in-production") // TODO: Move to env var

type Claims struct {
	UserID    uuid.UUID `json:"user_id"`
	CompanyID uuid.UUID `json:"company_id"`
	Email     string    `json:"email"`
	jwt.RegisteredClaims
}

// GenerateToken generates a JWT token for a user
func GenerateToken(userID, companyID uuid.UUID, email string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour) // Token expires in 24 hours

	claims := &Claims{
		UserID:    userID,
		CompanyID: companyID,
		Email:     email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "syncflow-backend",
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %v", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// ExtractCompanyDomain extracts company domain from email
// Example: rohan@uipath.com -> uipath
func ExtractCompanyDomain(email string) string {
	parts := splitEmail(email)
	if len(parts) != 2 {
		return ""
	}
	return parts[1] // Return domain part
}

// ExtractCompanyName extracts company name from email domain
// Example: rohan@uipath.com -> uipath (removes .com, .org, etc.)
func ExtractCompanyName(email string) string {
	domain := ExtractCompanyDomain(email)
	if domain == "" {
		return ""
	}

	// Remove common TLDs
	tlds := []string{".com", ".org", ".net", ".io", ".co", ".ai", ".dev"}
	for _, tld := range tlds {
		if len(domain) > len(tld) && domain[len(domain)-len(tld):] == tld {
			return domain[:len(domain)-len(tld)]
		}
	}

	return domain
}

func splitEmail(email string) []string {
	for i := 0; i < len(email); i++ {
		if email[i] == '@' {
			return []string{email[:i], email[i+1:]}
		}
	}
	return []string{email}
}


