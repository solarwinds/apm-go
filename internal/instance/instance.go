package instance

import (
	"github.com/google/uuid"
	"github.com/solarwinds/apm-go/internal/log"
)

var Id = ""

// We generate the instance ID on startup and keep the state here instead of `host.ID`
// though we report it from `(*ID).InstanceID()`
func init() {
	i, err := uuid.NewRandom()
	if err != nil {
		log.Error("error generating instance id", err)
		Id = "unknown"
	} else {
		Id = i.String()
	}
}
