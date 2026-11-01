package providersmaster

import (
	"bytes"
	"crypto/ed25519"
	"testing"

	"mytonprovider-coordinator/internal/models/db"
)

func overlayTestKey(b byte) ed25519.PublicKey {
	k := make(ed25519.PublicKey, ed25519.PublicKeySize)
	k[0] = b
	return k
}

func overlayTestCandidate(b byte, ip string, port int32, bags int) *overlayStorageCandidate {
	return &overlayStorageCandidate{
		PublicKey: overlayTestKey(b),
		IP:        ip,
		Port:      port,
		Bags:      bags,
	}
}

func TestPickOverlayStorage(t *testing.T) {
	t.Parallel()

	keyA := overlayTestKey(1)
	keyB := overlayTestKey(2)

	tests := []struct {
		name       string
		candidates map[string]*overlayStorageCandidate
		known      db.IPInfo
		wantIP     string
		wantPort   int32
		wantKey    ed25519.PublicKey
		wantErr    bool
	}{
		{
			name: "majority A over first-listed B",
			candidates: map[string]*overlayStorageCandidate{
				overlayCandidateKey(keyB): overlayTestCandidate(2, "1.2.3.4", 61985, 1),
				overlayCandidateKey(keyA): overlayTestCandidate(1, "1.2.3.4", 57969, 3),
			},
			wantIP:   "1.2.3.4",
			wantPort: 57969,
			wantKey:  keyA,
		},
		{
			name: "tie keeps known A",
			candidates: map[string]*overlayStorageCandidate{
				overlayCandidateKey(keyA): overlayTestCandidate(1, "1.2.3.4", 57969, 1),
				overlayCandidateKey(keyB): overlayTestCandidate(2, "1.2.3.4", 61985, 1),
			},
			known: db.IPInfo{
				PublicKey: keyA,
				IP:        "1.2.3.4",
				Port:      57969,
			},
			wantIP:   "1.2.3.4",
			wantPort: 57969,
			wantKey:  keyA,
		},
		{
			name: "tie without known is error",
			candidates: map[string]*overlayStorageCandidate{
				overlayCandidateKey(keyA): overlayTestCandidate(1, "1.2.3.4", 57969, 1),
				overlayCandidateKey(keyB): overlayTestCandidate(2, "1.2.3.4", 61985, 1),
			},
			wantErr: true,
		},
		{
			name: "unique winner replaces weaker known",
			candidates: map[string]*overlayStorageCandidate{
				overlayCandidateKey(keyA): overlayTestCandidate(1, "1.2.3.4", 57969, 3),
				overlayCandidateKey(keyB): overlayTestCandidate(2, "1.2.3.4", 61985, 1),
			},
			known: db.IPInfo{
				PublicKey: keyB,
				IP:        "1.2.3.4",
				Port:      61985,
			},
			wantIP:   "1.2.3.4",
			wantPort: 57969,
			wantKey:  keyA,
		},
		{
			name:       "no candidates",
			candidates: map[string]*overlayStorageCandidate{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := pickOverlayStorage(tt.candidates, tt.known)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("pickOverlayStorage() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("pickOverlayStorage() unexpected error: %v", err)
			}
			if got.IP != tt.wantIP || got.Port != tt.wantPort {
				t.Fatalf("pickOverlayStorage() endpoint = %s:%d, want %s:%d", got.IP, got.Port, tt.wantIP, tt.wantPort)
			}
			if !bytes.Equal(got.PublicKey, tt.wantKey) {
				t.Fatalf("pickOverlayStorage() key = %x, want %x", got.PublicKey, tt.wantKey)
			}
		})
	}
}
