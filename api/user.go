package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/jenkka/basic-bank-app/db/sqlc"
	"github.com/jenkka/basic-bank-app/util"
	"github.com/lib/pq"
)

// Returned for any credential failure, to avoid user enumeration.
var errInvalidCredentials = errors.New("invalid credentials")

// Compared against on the user-not-found path so response time
// matches the wrong-password path (defends against timing attacks).
var dummyHash string

func init() {
	h, err := util.HashPassword("timing-equalization-dummy")
	if err != nil {
		panic(fmt.Sprintf("dummy hash init failed: %v", err))
	}
	dummyHash = h
}

type createUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,gte=8,lte=30"`
	Email    string `json:"email" binding:"required,email"`
	FullName string `json:"full_name" binding:"required"`
}

type userResponse struct {
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	CreatedAt    time.Time `json:"created_at"`
	PwdUpdatedAt time.Time `json:"pwd_updated_at"`
}

func newUserResponse(user db.User) userResponse {
	res := userResponse{
		Username:     user.Username,
		FullName:     user.FullName,
		Email:        user.Email,
		CreatedAt:    user.CreatedAt,
		PwdUpdatedAt: user.PwdUpdatedAt,
	}

	return res
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
					err = fmt.Errorf(
						"a user with the username %s already exists",
						userParams.Username,
					)
				case usersEmailKeyConstraint:
					err = fmt.Errorf(
						"a user with the email %s already exists",
						userParams.Email,
					)
				}
				ctx.JSON(http.StatusConflict, errorResponse(err))
				return
			}
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	res := newUserResponse(user)

	ctx.JSON(http.StatusCreated, res)
}

type loginUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,gte=8,lte=30"`
}

type loginUserResponse struct {
	AccessToken string       `json:"access_token"`
	User        userResponse `json:"user"`
}

func (server *Server) loginUser(ctx *gin.Context) {
	var req loginUserRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.store.GetUser(ctx, req.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			util.CheckPassword(req.Password, dummyHash)
			ctx.JSON(
				http.StatusUnauthorized,
				errorResponse(errInvalidCredentials),
			)
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if !util.CheckPassword(req.Password, user.HashedPwd) {
		ctx.JSON(
			http.StatusUnauthorized,
			errorResponse(errInvalidCredentials),
		)
		return
	}

	accessToken, err := server.tokenMaker.CreateToken(
		user.Username,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	res := loginUserResponse{
		AccessToken: accessToken,
		User:        newUserResponse(user),
	}
	ctx.JSON(http.StatusOK, res)
}
