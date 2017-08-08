package main

import (
	"fmt"
	"gopkg.in/olivere/elastic.v5"
	"log"

	"github.com/gin-gonic/gin"
)

// get hotel URLS, add them all to elastic
func createEntries(db *elastic.Client) {

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

			ctx, cancel := defaultCtx()
			defer cancel()

			res, err := db.
				Index().Index(hotelIndex).
				Type(fmt.Sprintf("hotel-%v", lang)).
				Id(hotel.HotelID).
				BodyJson(hotel).
				Do(ctx)
			if err != nil {
				log.Fatal(err)
			}

			if !res.Created {
				log.Fatal(fmt.Errorf("Could not create hotel with id=%v", hotel.HotelID))
			}

			log.Printf("Added hotel with id=%v (%v)", hotel.HotelID, lang)

		}
	}

}

func main() {

	db, err := connect("http://localhost:9200")
	if err != nil {
		log.Fatal(err)
	}

	go createEntries(db)

	s := &server{
		db: db,
	}

	router := gin.Default()

	router.POST("/search", s.search)
	router.POST("/get", s.get)

	log.Fatal(router.Run(":8081"))
}
