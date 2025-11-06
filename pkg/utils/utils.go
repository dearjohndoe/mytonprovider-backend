package utils

import (
	"strings"
	"time"
)

func TryNTimes(f func() error, n int) (err error) {
	for i := 0; i < n; i++ {
		err = f()
		if err == nil {
			return nil
		}

		time.Sleep(time.Second)
	}
	return err
}

func ValidateBagID(bagid string) bool {
	if len(bagid) != 64 {
		return false
	}

	bagid = strings.ToLower(bagid)
	for i := range 64 {
		c := bagid[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}

	return true
}
