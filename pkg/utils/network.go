package utils

import "time"

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
