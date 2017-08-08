package main

// Languages is the list of languages add to Elasticsearch
var Languages = []string{"ru", "en"}

// Location of a hotel
type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Hotel is the searchable hotel document
type Hotel struct {
	HotelID  string   `json:"id"`
	Name     string   `json:"name"`
	Address  string   `json:"address"`
	Summary  string   `json:"summary"`
	Location Location `json:"location"`
	Amenties []string `json:"amenties,omitempty"`
	Rooms    []Room   `json:"rooms"`
}

// Room definition (one room can haz multiple beds)
type Room struct {
	MaxPeople int    `json:"max_people"`
	Name      string `json:"name"`
}
