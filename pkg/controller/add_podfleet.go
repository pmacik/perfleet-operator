package controller

import (
	"github.com/pmacik/podfleet-operator/pkg/controller/podfleet"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, podfleet.Add)
}
