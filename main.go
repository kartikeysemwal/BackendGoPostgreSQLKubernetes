package main

import (
	"database/sql"
	"log"

	"github.com/kartikeysemwal/goLangBackend/api"
	db "github.com/kartikeysemwal/goLangBackend/db/sqlc"
	"github.com/kartikeysemwal/goLangBackend/util"

	_ "github.com/lib/pq"
)

func main() {

	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("connot load config:", err)
	}

	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	store := db.NewStore(conn)
	server := api.NewServer(store)

	err = server.Start(config.ServerAddress)
	if err != nil {
		log.Fatal("Cannot start server:", err)
	}
}
