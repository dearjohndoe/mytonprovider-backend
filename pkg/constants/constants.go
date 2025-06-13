package constants

// Sorting constants
const (
	PubKeyColumn      = "p.public_key"
	UptimeColumn      = "p.uptime"
	WorkingTimeColumn = "p.registered_at"
	RatingColumn      = "p.rating"
	PriceColumn       = "p.rate_per_mb_per_day"
)

var SortingMap = map[string]string{
	"pubkey":      PubKeyColumn,
	"uptime":      UptimeColumn,
	"workingtime": WorkingTimeColumn,
	"rating":      RatingColumn,
	"price":       PriceColumn,
}

// Order constants
const (
	Asc  = "ASC"
	Desc = "DESC"
)

var OrderMap = map[string]string{
	"asc":  Asc,
	"desc": Desc,
}
