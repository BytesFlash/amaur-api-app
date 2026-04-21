package timeutil

import "time"

var santiagoLocation = loadSantiagoLocation()

func Santiago() *time.Location {
	return santiagoLocation
}

func loadSantiagoLocation() *time.Location {
	loc, err := time.LoadLocation("America/Santiago")
	if err == nil {
		return loc
	}
	return time.FixedZone("America/Santiago", -4*60*60)
}
