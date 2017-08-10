package main

import (
	"config"
	"db"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {

	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	cfg.DatabaseConfig.DropOnStartup = false
	db, err := db.Connect(&cfg.DatabaseConfig)
	if err != nil {
		log.Fatal(err)
	}

	s := &server{
		db: db,
	}

	router := gin.Default()

	router.POST("/search", s.search)
	router.POST("/get", s.get)

	log.Fatal(router.Run(cfg.Listen))
}
