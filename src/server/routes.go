package main

import (
	"db"
	"encoding/json"
	"fmt"
	"log"
	"models"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
	"gopkg.in/olivere/elastic.v5"
)

type server struct {
	db *elastic.Client
}

type SearchRequest struct {
	// Search only by title
	Name string `json:"name"`
	// Optionally limit results only to one language
	Language string `json:"language"`

	// Search by coordinates && radius
	Location *models.Location `json:"location"`
	// distance in meters
	Radius uint `json:"radius"`

	// Search by location name
	Address string `json:"address"`

	// Result page (0 is first one, 1 -- second, ...)
	Page uint
}

type HotelResultEntry struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type SearchResult struct {
	NothingFound bool `json:"nothing_found,omitempty"`
	// list of hotels that are sorted by names
	Hotels []HotelResultEntry `json:"hotels"`
}

type ErrorResponse struct {
	Err string `json:"error"`
}

type GetRequest struct {
	ID       string `json:"id"`
	Language string `json:"language"`
}

type GetResult struct {
	NotFound bool          `json:"not_found,omitempty"`
	Hotel    *models.Hotel `json:"result"`
}

const PerPage = 10

func (s *server) search(c *gin.Context) {

	// decode request
	var req SearchRequest
	err := c.BindJSON(&req)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, ErrorResponse{"decode request error"})
		return
	}

	search := s.db.Search().
		Index(db.HotelIndex).
		Sort("name.raw", true).
		// only first page, a real app should ofc
		// implement pagination
		From(int(req.Page * PerPage)).Size(PerPage).
		FetchSourceContext(
			// retrieve only needed fields
			elastic.NewFetchSourceContext(true).Include("name", "id"),
		)

	if req.Language > "" {
		search = search.Type(fmt.Sprintf("hotel-%v", req.Language))
	}

	query := elastic.NewBoolQuery()

	// decide what to do

	if req.Name > "" {
		// search by hotel name
		query = query.Must(
			elastic.NewMultiMatchQuery(req.Name, "name", "name.stemmed-*"),
		)
	}

	if req.Location != nil && req.Radius > 0 {

		// verify values first
		var (
			lat = req.Location.Lat
			lon = req.Location.Lon
		)

		if lat <= -90 || lat >= 90 || lon <= -180 || lon >= 180 {
			c.JSON(http.StatusBadRequest, ErrorResponse{"Invalid location"})
			return
		}

		if req.Radius > 1000 {
			c.JSON(http.StatusBadRequest, ErrorResponse{"Radius is too huge"})
			return
		}

		// filter by geo-position
		filter := elastic.NewGeoDistanceQuery("location").
			Point(lat, lon).
			Distance(fmt.Sprintf("%dm", req.Radius))

		query = query.Filter(filter)
	}

	if req.Address > "" {
		// search by hotel addr
		query = query.Must(
			elastic.NewMultiMatchQuery(req.Address, "address", "address.stemmed-*"),
		)
	}

	ctx, cancel := db.DefaultCtx()
	defer cancel()

	// perform the request
	res, err := search.Query(query).Do(ctx)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{"Database error"})
		return
	}

	// scan result
	var result SearchResult
	if res.TotalHits() == 0 {
		// I wonder why len(result.Hotels) == 0 on client-side
		// is bad. It's anyway a thing that should be checked
		result.NothingFound = true
	} else {
		for _, item := range res.Each(reflect.TypeOf(HotelResultEntry{})) {
			hotel := item.(HotelResultEntry)
			result.Hotels = append(result.Hotels, hotel)
		}
	}

	c.JSON(http.StatusOK, result)
}

func (s *server) get(c *gin.Context) {

	// decode request
	var req GetRequest
	err := c.BindJSON(&req)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, ErrorResponse{"Decode request error"})
		return
	}

	// fallback to English
	if req.Language == "" {
		req.Language = "en"
	}

	ctx, cancel := db.DefaultCtx()
	defer cancel()

	// get by ID
	res, err := s.db.Get().
		Index(db.HotelIndex).
		Type(fmt.Sprintf("hotel-%v", req.Language)).
		Id(req.ID).Do(ctx)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, ErrorResponse{"Database error"})
		return
	}

	// decode result
	var hotel models.Hotel
	err = json.Unmarshal(*res.Source, &hotel)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{"Decode result error"})
		return
	}

	c.JSON(http.StatusOK, GetResult{
		Hotel: &hotel,
	})
}
