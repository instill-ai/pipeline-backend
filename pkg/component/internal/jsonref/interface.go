// The following code is based on lestrrat's work, available at https://github.com/lestrrat-go/jsref.

package jsonref

import (
	"errors"
	"net/url"
	"reflect"
)

var zeroval = reflect.Value{}

var ErrMaxRecursion = errors.New("reached max number of recursions")
var ErrReferenceLoop = errors.New("reference loop detected")

// Resolver is responsible for interpreting the provided JSON
// reference.
type Resolver struct {
	providers     []Provider
	MaxRecursions int
}

// Provider resolves a URL into a ... thing.
type Provider interface {
	Get(*url.URL) (interface{}, error)
}
