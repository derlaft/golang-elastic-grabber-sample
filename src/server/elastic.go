package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"gopkg.in/olivere/elastic.v5"
)

const hotelIndex = "booking"

// H is a syntax sugar to make
// dynamic-json code more befautiful
type H map[string]interface{}

var analyzers = map[string]string{
	"ru": "russian",
	"en": "english",
}

func defaultCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Second*15)
}

func connect(connect string) (*elastic.Client, error) {
	client, err := elastic.NewClient(
		elastic.SetURL(connect),
		elastic.SetHealthcheckInterval(10*time.Second),
		// @TODO delete this
		elastic.SetErrorLog(log.New(os.Stderr, "ELASTIC ", log.LstdFlags)),
		elastic.SetInfoLog(log.New(os.Stdout, "", log.LstdFlags)),
	)
	if err != nil {
		return nil, err
	}

	indexExists, err := isIndexExists(client)
	if err != nil {
		return nil, err
	}

	if indexExists {
		_, err := client.DeleteIndex(hotelIndex).Do(context.Background())
		if err != nil {
			return nil, err
		}
	}

	err = createIndex(client)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func isIndexExists(client *elastic.Client) (bool, error) {

	ctx, cancel := defaultCtx()
	defer cancel()

	exists, err := client.IndexExists(hotelIndex).Do(ctx)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func createIndex(client *elastic.Client) error {

	ctx, cancel := defaultCtx()
	defer cancel()

	mappings := H{}
	for _, lang := range Languages {

		// language-specific stemmed field
		var (
			stemmedName     = "stemmed-" + lang
			stemmedSubfield = H{
				stemmedName: H{
					"type":     "string",
					"analyzer": analyzers[lang],
				},
			}
		)

		mappings["hotel-"+lang] = H{
			"properties": H{
				"address":  H{"type": "string", "fields": stemmedSubfield},
				"summary":  H{"type": "string", "fields": stemmedSubfield},
				"location": H{"type": "geo_point"},
				"name": H{
					"type": "string",
					"fields": H{
						stemmedName: stemmedSubfield[stemmedName],
						// extra sorting subfield
						// https://www.elastic.co/guide/en/elasticsearch/guide/current/multi-fields.html
						"raw": H{
							"type":  "string",
							"index": "not_analyzed",
						},
					},
				},
			},
		}
	}

	index, err := client.CreateIndex(hotelIndex).BodyJson(H{
		"mappings": mappings,
	}).Do(ctx)

	if err != nil {
		return err
	}
	if !index.Acknowledged {
		return fmt.Errorf("Index is not ackd")
	}

	return nil
}
