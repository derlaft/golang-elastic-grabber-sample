package main

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

const (
	bookingRoot = "https://www.booking.com"
	hotelIds    = "https://www.booking.com/searchresults.html?dest_id=-2874130;dest_type=city"
)

// FetcherWorkers is number of workers that fetch pages from booking.com at the same time
const FetcherWorkers = 4

// regexps for the extraction of hotel coordinates
// these must be extracted directly from <script> value
var (
	latRegexp = regexp.MustCompile(`booking.env.b_map_center_latitude = (-?[0-9]+.[0-9]+);`)
	lonRegexp = regexp.MustCompile(`booking.env.b_map_center_longitude = (-?[0-9]+.[0-9]+);`)
)

// HotelID is just the name of the hotel
// Example: guest-house-snezhny-bars-abzakovo
// However, it seems it is unique only over
// the russian hotels, so production ID
// should also contain the country code
type HotelID string

type hotelParseResult struct {
	h   map[string]*Hotel
	err error
}

// GetURL returns a document URL for this language
func (h *HotelID) GetURL(language string) string {
	switch language {

	// english is the default one
	case "", "en":
		return fmt.Sprintf("%s/hotel/ru/%s.html", bookingRoot, *h)

	default:
		return fmt.Sprintf("%s/hotel/ru/%s.%s.html", bookingRoot, *h, language)
	}
}

// Extract the raw hotel ID from the component
// urls -- link of urls to hotels, example:
//  /hotel/ru/guest-house-snezhny-bars-abzakovo.html?dest_type=city;dest_id=-2874130#hotelTmpl
func parseURLs(urls []string) ([]HotelID, error) {

	var (
		results []HotelID
	)

	for _, hotelLink := range urls {

		// parse url components
		parsed, err := url.Parse(bookingRoot + hotelLink)
		if err != nil {
			return nil, err
		}

		// extract path (/hotel/ru/guest-house-snezhny-bars-abzakovo.html)
		parts := strings.Split(parsed.Path, "/")
		if len(parts) == 0 {
			return nil, fmt.Errorf("Unexpected url format")
		}

		// get the last element (guest-house-snezhny-bars-abzakovo.html)
		hotel := parts[len(parts)-1]
		// remove .html suffix
		hotelId := strings.TrimSuffix(hotel, ".html")

		// send the response back
		results = append(results, HotelID(hotelId))
	}

	return results, nil
}

func parseAll(ids []HotelID) chan hotelParseResult {

	var (
		// output chan
		results = make(chan hotelParseResult, 16)
		// output chan closing wg
		wg sync.WaitGroup
		// worker tasks chan
		workerQueue = make(chan HotelID, 16)
	)

	// run workers
	for i := 0; i < FetcherWorkers; i++ {
		go func() {
			for hotelLink := range workerQueue {
				// do the parsing
				hotel, err := hotelLink.Parse()

				// send the response back
				results <- hotelParseResult{hotel, err}
				wg.Done()
			}
		}()
	}

	// provide them with requests
	for _, id := range ids {
		var copy = id
		workerQueue <- copy
		wg.Add(1)
	}

	// close worker queue
	close(workerQueue)
	// close result queue once everything is sent
	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

// Get hotel IDS locatel in Abzakovo.
// No pagination implemented -- there's less than one page
// of results
func grabHotelsToParse() ([]HotelID, error) {

	// Load HTML
	doc, err := goquery.NewDocument(hotelIds)
	if err != nil {
		return nil, err
	}

	// Exclude "Nearby" shit
	// from the query
	beforeNearby := doc.
		Find(".sr_separator").
		PrevAll()

	// Get hotel links by selector
	sel := beforeNearby.
		Find("a.hotel_name_link.url")
	if sel.Size() <= 0 {
		return nil, fmt.Errorf("Found no hotels div")
	}

	// Extract hotel URLs from <a href... elements
	var urls []string
	sel.Each(func(_ int, hotelLink *goquery.Selection) {

		// Extract HREF from hotel link
		urlAttr, present := hotelLink.Attr("href")
		if !present {
			return
		}

		urls = append(urls, urlAttr)
	})

	// extract pure hotels IDs from the result
	result, err := parseURLs(urls)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// trim empty symbols that happen to be after calling .Text()
//  on selection
func trim(a string) string {

	return strings.Trim(a, "\t\n ")
}

// Prase grabs documents from the booking
// and returns multi-language indices in
// map: lang_name -> hotel info
func (h HotelID) Parse() (map[string]*Hotel, error) {

	var (
		result   = map[string]*Hotel{}
		once     sync.Once
		lat, lon float64
	)

	for _, lang := range Languages {

		// Load HTML
		doc, err := goquery.NewDocument(h.GetURL(lang))
		if err != nil {
			return nil, err
		}

		var onceErr error

		// There is no need to parse this field multiple times
		//  as they do not vary on the language used
		once.Do(func() {

			html, err := doc.Html()
			if err != nil {
				onceErr = err
				return
			}

			// find lat && lon; parse from string to float
			if tok := latRegexp.FindStringSubmatch(html); len(tok) > 1 {
				lat, _ = strconv.ParseFloat(tok[1], 64)
			}
			if tok := lonRegexp.FindStringSubmatch(html); len(tok) > 1 {
				lon, _ = strconv.ParseFloat(tok[1], 64)
			}
		})

		if onceErr != nil {
			return nil, err
		}

		// Get the most simple fields by
		//  CSS selectors
		var (
			hotelName = doc.Find("h2.hp__hotel-name").Text()
			addrText  = doc.Find("span.hp_address_subtitle").Text()
			summary   = doc.Find("#summary").Text()
		)

		// Grab amenties
		var amenties []string
		doc.
			Find(".facilities-sliding-keep"). // avoid dupes
			Find(".important_facility").
			Each(func(_ int, am *goquery.Selection) {
				amenties = append(amenties, trim(am.Text()))
			})

		// Process rooms
		var rooms []Room
		doc.Find(".roomstable").Find("tbody").First().Children().
			Each(func(_ int, row *goquery.Selection) {

				// skip empty
				if row.HasClass("extendedRow") {
					return
				}

				rooms = append(rooms, Room{
					MaxPeople: row.Find("i.bicon-occupancy").Size(),
					Name:      trim(row.Find("a.togglelink").Text()),
				})
			})

		result[lang] = &Hotel{
			HotelID: string(h),
			Name:    trim(hotelName),
			Address: trim(addrText),
			Summary: trim(summary),
			Location: Location{
				Lat: lat,
				Lon: lon,
			},
			Amenties: amenties,
			Rooms:    rooms,
		}

	}

	return result, nil
}
