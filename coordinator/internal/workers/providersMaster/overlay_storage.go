package providersmaster

import (
	"crypto/ed25519"
	"fmt"

	"mytonprovider-coordinator/internal/models/db"
)

type overlayStorageCandidate struct {
	PublicKey ed25519.PublicKey
	IP        string
	Port      int32
	Bags      int
}

func overlayCandidateKey(pk ed25519.PublicKey) string {
	return string(pk)
}

func overlayCandidateIP(c *overlayStorageCandidate) db.IPInfo {
	return db.IPInfo{
		PublicKey: append(ed25519.PublicKey(nil), c.PublicKey...),
		IP:        c.IP,
		Port:      c.Port,
	}
}

func pickOverlayStorage(candidates map[string]*overlayStorageCandidate, known db.IPInfo) (db.IPInfo, error) {
	if len(candidates) == 0 {
		return db.IPInfo{}, fmt.Errorf("no overlay storage candidates")
	}

	maxBags := 0
	for _, c := range candidates {
		if c.Bags > maxBags {
			maxBags = c.Bags
		}
	}

	leaders := make([]*overlayStorageCandidate, 0, 2)
	for _, c := range candidates {
		if c.Bags == maxBags {
			leaders = append(leaders, c)
		}
	}

	knownValid := hasValidResolvedStorageEndpoint(db.ProviderIP{Storage: known})
	if knownValid {
		if knownCand := candidates[overlayCandidateKey(known.PublicKey)]; knownCand != nil && knownCand.Bags >= maxBags {
			return overlayCandidateIP(knownCand), nil
		}
	}

	if len(leaders) != 1 {
		return db.IPInfo{}, fmt.Errorf("ambiguous overlay storage candidates with %d bags", maxBags)
	}

	return overlayCandidateIP(leaders[0]), nil
}

func overlayCandidateBags(candidates map[string]*overlayStorageCandidate, pk ed25519.PublicKey) int {
	if c := candidates[overlayCandidateKey(pk)]; c != nil {
		return c.Bags
	}
	return 0
}

func overlayTopBags(candidates map[string]*overlayStorageCandidate) int {
	top := 0
	for _, c := range candidates {
		if c.Bags > top {
			top = c.Bags
		}
	}
	return top
}
