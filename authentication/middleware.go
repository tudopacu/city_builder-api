package authentication

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Server struct {
}

type contextKey string

const playerIDKey contextKey = "player_id"

// PlayerID returns the ID of the player that AuthMiddleware authenticated.
func PlayerID(c *gin.Context) (uint, bool) {
	value, ok := c.Get(playerIDKey)
	if !ok {
		return 0, false
	}
	id, ok := value.(uint)
	return id, ok
}

func (s *Server) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string
		if cookie, err := c.Request.Cookie("auth_token"); err == nil {
			tokenStr = cookie.Value
		} else {
			auth := c.GetHeader("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				tokenStr = strings.TrimPrefix(auth, "Bearer ")
			}
		}
		if tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			c.Abort()
			return
		}

		parsed, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			// HMAC accepts an empty key, so never verify before InitJwtSecret has run.
			if len(JwtSecret) < minJwtSecretLength {
				return nil, errors.New("jwt secret is not configured")
			}
			return JwtSecret, nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
		if err != nil || !parsed.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		claims, ok := parsed.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}
		sub, ok := claims["sub"].(string)
		if !ok || sub == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token subject"})
			c.Abort()
			return
		}

		// Unlike Sscanf, ParseUint rejects anything after the digits, so the
		// subject can only ever be a plain player ID.
		id, err := strconv.ParseUint(sub, 10, 32)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id in token"})
			c.Abort()
			return
		}

		c.Set(playerIDKey, uint(id))

		c.Next()
	}
}
