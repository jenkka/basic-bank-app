package main

import (
	"database/sql"
	"log"

	"github.com/jenkka/dummy-bank/api"
	db "github.com/jenkka/dummy-bank/db/sqlc"
	"github.com/jenkka/dummy-bank/util"
	_ "github.com/lib/pq"
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
	server, err := api.NewServer(store, config)
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}

	err = server.Start(config.ServerAddress)
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
