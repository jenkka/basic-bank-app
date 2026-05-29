package main

import (
	"database/sql"
	"log"
	"net"

	"github.com/jenkka/dummy-bank/api"
	db "github.com/jenkka/dummy-bank/db/sqlc"
	"github.com/jenkka/dummy-bank/gapi"
	pb "github.com/jenkka/dummy-bank/pb/dummybank/v1"
	"github.com/jenkka/dummy-bank/util"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("Failed to load config file:", err)
	}

	conn, err := sql.Open(config.DbDriver, config.DbSource)
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}
	defer conn.Close()

	store := db.NewStore(conn)

	runGRPCServer(config, store)
}

func runGRPCServer(config util.Config, store db.Store) {
	server, err := gapi.NewServer(store, config)
	if err != nil {
		log.Fatal("Failed to create gRPC server:", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterDummyBankServiceServer(grpcServer, server)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", config.GRPCServerAddress)
	if err != nil {
		log.Fatal("Failed to create listener:", err)
	}

	log.Printf("Starting gRPC server on %s", config.GRPCServerAddress)
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatal("Failed to start gRPC server:", err)
	}
}

func runGinServer(config util.Config, store db.Store) {
	server, err := api.NewServer(store, config)
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}

	log.Printf("Starting server on %s", config.HTTPServerAddress)
	err = server.Start(config.HTTPServerAddress)
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
