package constants

// Sorting constants
const (
	UptimeColumn      = "uptime"
	WorkingTimeColumn = "working_time"
	RatingColumn      = "rating"
	MaxSpanColumn     = "max_span"
	PriceColumn       = "price"
)

var SortingMap = map[string]string{
	"uptime":      UptimeColumn,
	"workingtime": WorkingTimeColumn,
	"rating":      RatingColumn,
	"maxspan":     MaxSpanColumn,
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
