package main

import (
	"encoding/json"
	"fmt"
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
	Location *Location `json:"location"`
	// distance with unit name --
	// for ex. -- 20m, 10km
	Radius string `json:"radius"`

	// Search by location name
	Address string `json:"address"`
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
	NotFound bool   `json:"not_found,omitempty"`
	Hotel    *Hotel `json:"result"`
}

func (s *server) search(c *gin.Context) {

	// decode request
	var req SearchRequest
	err := c.BindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	search := s.db.Search().
		Index(hotelIndex).
		FetchSourceContext(
			// retrieve only needed fields
			elastic.NewFetchSourceContext(true).Include("name", "id"),
		)

	if req.Language > "" {
		search = search.Type(fmt.Sprintf("hotel-%v", req.Language))
	}

	// decide what to do
	switch {

	case req.Name > "":
		// search by hotel name
		search = search.Query(
			elastic.NewMatchQuery("name", req.Name),
		)

	case req.Location != nil && req.Radius > "":

		filter := elastic.NewGeoDistanceQuery("location").
			Point(req.Location.Lat, req.Location.Lon).
			Distance(req.Radius)

		query := elastic.NewBoolQuery().
			Must(elastic.NewMatchAllQuery()).
			Filter(filter)

		search = search.Query(query)

	case req.Address > "":
		// search by hotel addr
		search = search.Query(
			elastic.NewMatchQuery("address", req.Address),
		)

	default:
		c.JSON(http.StatusBadRequest, ErrorResponse{fmt.Sprintf("No search parameters")})
		return
	}

	ctx, cancel := defaultCtx()
	defer cancel()

	// perform the request
	res, err := search.Do(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{err.Error()})
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
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	// fallback to English
	if req.Language == "" {
		req.Language = "en"
	}

	ctx, cancel := defaultCtx()
	defer cancel()

	// get by ID
	res, err := s.db.Get().
		Index(hotelIndex).
		Type(fmt.Sprintf("hotel-%v", req.Language)).
		Id(req.ID).Do(ctx)

	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	// decode result
	var hotel Hotel
	err = json.Unmarshal(*res.Source, &hotel)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	c.JSON(http.StatusOK, GetResult{
		Hotel: &hotel,
	})
}
