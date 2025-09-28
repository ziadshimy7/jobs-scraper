package models

type Job struct {
	ID          int64
	Title       string
	Company     string
	CompanyLink string
	Location    string
	JobLink     string
}

type SearchQuery struct {
	Keywords string `json:"keywords"`
	Location string `json:"location"`
	FWT      string `json:"f_WT"`  // Work type filter (1=onsite, 2=remote, 3=hybrid)
	GeoId    string `json:"geoId"` // Geographic location ID
}
