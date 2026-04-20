package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/jenkka/basic-bank-app/db/sqlc"
	"github.com/jenkka/basic-bank-app/util"
	"github.com/lib/pq"
)

type createUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,gte=8,lte=30"`
	Email    string `json:"email" binding:"required,email"`
	FullName string `json:"full_name" binding:"required"`
}

type createUserResponse struct {
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	CreatedAt    time.Time `json:"created_at"`
	PwdUpdatedAt time.Time `json:"pwd_updated_at"`
}

func (server *Server) createUser(ctx *gin.Context) {
	var req createUserRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		err = fmt.Errorf("failed to hash password: %w", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	userParams := db.CreateUserParams{
		Username:  req.Username,
		HashedPwd: hashedPassword,
		Email:     req.Email,
		FullName:  req.FullName,
	}

	user, err := server.store.CreateUser(ctx, userParams)
	if err != nil {
		if pqError, ok := err.(*pq.Error); ok {
			if pqError.Code.Name() == uniqueViolation {
				switch pqError.Constraint {
				case usersPkeyConstraint:
					err = fmt.Errorf("a user with the username %s already exists", userParams.Username)
				case usersEmailKeyConstraint:
					err = fmt.Errorf("a user with the email %s already exists", userParams.Email)
				}
				ctx.JSON(http.StatusConflict, errorResponse(err))
				return
			}
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	res := createUserResponse{
		Username:     user.Username,
		FullName:     user.FullName,
		Email:        user.Email,
		CreatedAt:    user.CreatedAt,
		PwdUpdatedAt: user.PwdUpdatedAt,
	}

	ctx.JSON(http.StatusCreated, res)
}
