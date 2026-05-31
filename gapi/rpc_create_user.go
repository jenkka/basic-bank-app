package gapi

import (
	"context"
	"fmt"

	"github.com/jenkka/dummy-bank/api"
	db "github.com/jenkka/dummy-bank/db/sqlc"
	pb "github.com/jenkka/dummy-bank/pb/dummybank/v1"
	"github.com/jenkka/dummy-bank/util"
	"github.com/lib/pq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) CreateUser(
	ctx context.Context,
	req *pb.CreateUserRequest,
) (*pb.CreateUserResponse, error) {
	hashedPassword, err := util.HashPassword(req.GetPassword())
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"failed to hash password: %s",
			err,
		)
	}

	userParams := db.CreateUserParams{
		Username:  req.GetUsername(),
		HashedPwd: hashedPassword,
		Email:     req.GetEmail(),
		FullName:  req.GetFullName(),
	}

	user, err := server.store.CreateUser(ctx, userParams)
	if err != nil {
		pqError, ok := err.(*pq.Error)
		if !ok || pqError.Code.Name() != api.UniqueViolation {
			return nil, status.Errorf(codes.Internal, "failed to create user: %s", err)
		}

		var errMsg string
		switch pqError.Constraint {
		case api.UsersPkeyConstraint:
			errMsg = fmt.Sprintf("a user with the username %s already exists", userParams.Username)
		case api.UsersEmailKeyConstraint:
			errMsg = fmt.Sprintf("a user with the email %s already exists", userParams.Email)
		default:
			errMsg = "user already exists"
		}
		return nil, status.Errorf(codes.AlreadyExists, "%s", errMsg)
	}

	res := &pb.CreateUserResponse{
		User: convertUser(user),
	}

	return res, nil
}
