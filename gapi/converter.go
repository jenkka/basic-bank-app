package gapi

import (
	db "github.com/jenkka/dummy-bank/db/sqlc"
	pb "github.com/jenkka/dummy-bank/pb/dummybank/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func convertUser(user db.User) *pb.User {
	return &pb.User{
		Username:          user.Username,
		Email:             user.Email,
		FullName:          user.FullName,
		CreatedAt:         timestamppb.New(user.CreatedAt),
		PasswordUpdatedAt: timestamppb.New(user.PwdUpdatedAt),
	}
}
