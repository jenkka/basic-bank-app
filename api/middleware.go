package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jenkka/dummy-bank/token"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationPayloadKey = "authorization_payload"
	authorizationTypeBearer = "bearer"
)

func authMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader(authorizationHeaderKey)
		if authHeader == "" {
			err := errors.New("authorization header is missing")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		authHeaderFields := strings.Split(authHeader, " ")
		if len(authHeaderFields) != 2 ||
			strings.ToLower(authHeaderFields[0]) != authorizationTypeBearer {
			err := errors.New("bearer token was not provided")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		tokenString := authHeaderFields[1]
		payload, err := tokenMaker.VerifyToken(tokenString)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		ctx.Set(authorizationPayloadKey, payload)
		ctx.Next()
	}
}
