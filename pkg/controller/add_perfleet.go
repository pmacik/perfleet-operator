package controller

import (
	"github.com/pmacik/perfleet-operator/pkg/controller/perfleet"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, perfleet.Add)
}
