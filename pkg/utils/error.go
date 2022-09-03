package utils

import (
	"log"
)

func CatchErr(err error) {

	if err != nil {
		switch err.Error() {
		case "no rows in result set":
			return

		default:
			log.Fatal(err)
		}
	}

}
