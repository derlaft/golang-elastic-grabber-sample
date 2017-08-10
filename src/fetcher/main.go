package main

import (
	"config"
	"db"
	"fmt"
	"log"

	"gopkg.in/olivere/elastic.v5"
)

// get hotel URLS, add them all to elastic
func createEntries(conn *elastic.Client) {

	ids, err := grabHotelsToParse()
	if err != nil {
		log.Fatal(err)
	}

	for parse := range parseAll(ids) {

		// parsing error happened?
		if parse.err != nil {
			log.Fatal(parse.err)
		}

		var doc = parse.h

		for lang, hotel := range doc {

			// store each language doc as a
			// separate indice

			ctx, cancel := db.DefaultCtx()
			defer cancel()

			_, err := conn.
				Update().Index(db.HotelIndex).
				Type(fmt.Sprintf("hotel-%v", lang)).
				Id(hotel.HotelID).
				Doc(hotel).
				DocAsUpsert(true).
				Do(ctx)
			if err != nil {
				log.Fatal(err)
			}

			log.Printf("Added hotel with id=%v (%v)", hotel.HotelID, lang)

		}
	}

}

func main() {

	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	db, err := db.Connect(&cfg.DatabaseConfig)
	if err != nil {
		log.Fatal(err)
	}

	createEntries(db)
}
