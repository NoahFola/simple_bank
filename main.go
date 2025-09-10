package main

import (
	"database/sql"
	"log"

	"github.com/NoahFola/simple_bank/api"
	db "github.com/NoahFola/simple_bank/db/sqlc"
	"github.com/NoahFola/simple_bank/util"
	_ "github.com/lib/pq"
)

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("Cannot load config: ", err)
	}
	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("Cannot connect to the database", err)
	}

	store := db.NewStore(conn)
	server := api.NewServer(store)

	err = server.Start(config.ServerAddress)

	if err != nil {
		log.Fatal("cannot start server ", err)
	}
}
