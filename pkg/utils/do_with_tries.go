package utils

import "time"

func DoWithTries(fn func() error, attemps int, delay time.Duration) error {
	var err error

	for ; attemps > 0; attemps-- {
		if err := fn(); err != nil {
			time.Sleep(delay)
			attemps--
			continue
		}
		return err
	}
	return nil
}
