package gapi

import (
	"fmt"

	db "github.com/jenkka/dummy-bank/db/sqlc"
	pb "github.com/jenkka/dummy-bank/pb/dummybank/v1"
	"github.com/jenkka/dummy-bank/token"
	"github.com/jenkka/dummy-bank/util"
)

type Server struct {
	pb.UnimplementedDummyBankServiceServer
	store      db.Store
	tokenMaker token.Maker
	config     util.Config
}

func NewServer(store db.Store, config util.Config) (*Server, error) {
	tokenMaker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create token maker: %w", err)
	}

	server := &Server{
		store:      store,
		tokenMaker: tokenMaker,
		config:     config,
	}

	return server, nil
}
