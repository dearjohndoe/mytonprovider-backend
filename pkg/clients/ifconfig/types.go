package ifconfig

type Info struct {
	IP         string `json:"ip"`
	Country    string `json:"country"`
	CountryISO string `json:"country_iso"`
	City       string `json:"city"`
	TimeZone   string `json:"time_zone"`
}
