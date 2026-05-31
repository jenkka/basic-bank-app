package gapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	db "github.com/jenkka/dummy-bank/db/sqlc"
	pb "github.com/jenkka/dummy-bank/pb/dummybank/v1"
	"github.com/jenkka/dummy-bank/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Returned for any credential failure, to avoid user enumeration.
const strInvalidCredentials = "invalid credentials"

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

func (server *Server) LoginUser(
	ctx context.Context,
	req *pb.LoginUserRequest,
) (*pb.LoginUserResponse, error) {
	user, err := server.store.GetUser(ctx, req.GetUsername())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			util.CheckPassword(req.GetPassword(), dummyHash)
			return nil, status.Errorf(codes.Unauthenticated, strInvalidCredentials)
		}

		return nil, status.Errorf(codes.Internal, "failed to get user: %s", err)
	}

	if !util.CheckPassword(req.GetPassword(), user.HashedPwd) {
		return nil, status.Error(codes.PermissionDenied, strInvalidCredentials)
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create access token: %s", err)
	}

	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		server.config.RefreshTokenDuration,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create refresh token: %s", err)
	}

	parsedUUID, err := uuid.Parse(refreshPayload.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to parse refresh payload ID: %s", err)
	}

	session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
		ID:           parsedUUID,
		Username:     user.Username,
		RefreshToken: refreshToken,
		UserAgent:    "TODO",
		ClientIp:     "TODO",
		ExpiresAt:    refreshPayload.ExpiresAt.Time,
		IsBlocked:    false,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create session: %s", err)
	}

	res := &pb.LoginUserResponse{
		SessionId:             session.ID.String(),
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  timestamppb.New(accessPayload.ExpiresAt.Time),
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: timestamppb.New(refreshPayload.ExpiresAt.Time),
		User:                  convertUser(user),
	}
	return res, nil
}
