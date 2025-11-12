package constants

// Sorting constants
const (
	PubKeyColumn      = "p.public_key"
	UptimeColumn      = "p.uptime"
	WorkingTimeColumn = "p.registered_at"
	RatingColumn      = "p.rating"
	PriceColumn       = "p.rate_per_mb_per_day"
	LocationColumn    = "(p.ip_info->>'country' || ' (' || COALESCE(p.ip_info->>'country_iso', '') || ')', p.rating)"
)

var SortingMap = map[string]string{
	"pubkey":      PubKeyColumn,
	"uptime":      UptimeColumn,
	"workingtime": WorkingTimeColumn,
	"rating":      RatingColumn,
	"price":       PriceColumn,
	"location":    LocationColumn,
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

type ReasonCode uint32

const (
	ValidStorageProof ReasonCode = 0

	IPNotFound          ReasonCode = 101
	NotFound            ReasonCode = 102
	UnavailableProvider ReasonCode = 103
	CantCreatePeer      ReasonCode = 104
	UnknownPeer         ReasonCode = 105

	PingFailed        ReasonCode = 201
	InvalidBagID      ReasonCode = 202
	FailedInitialPing ReasonCode = 203

	GetInfoFailed ReasonCode = 301
	InvalidHeader ReasonCode = 302

	CantGetPiece     ReasonCode = 401
	CantParseBoC     ReasonCode = 402
	ProofCheckFailed ReasonCode = 403
)
