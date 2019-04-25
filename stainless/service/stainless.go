// Package stainless is a service for executing stainless verification and
// Ethereum bytecode generation on smart contracts written in a subset of
// Scala.
package stainless

import (
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

// ServiceName is the name to refer to the Stainless service.
const ServiceName = "Stainless"

func init() {
	onet.RegisterNewService(ServiceName, newStainlessService)
	network.RegisterMessage(&Request{})
	network.RegisterMessage(&Response{})
}

// Stainless is the service that performs stainless operations.
type Stainless struct {
	*onet.ServiceProcessor
}

// Request treats external request to this service.
func (service *Stainless) Request(req *Request) (network.Message, error) {
	console := ""
	report := make(map[string]interface{})

	log.Lvl4("Returning", console, report)

	return &Response{
		Console: console,
		Report:  report,
	}, nil
}

// newStatService creates a new service that is built for Status
func newStainlessService(context *onet.Context) (onet.Service, error) {
	service := &Stainless{
		ServiceProcessor: onet.NewServiceProcessor(context),
	}
	err := service.RegisterHandler(service.Request)
	if err != nil {
		return nil, err
	}

	return service, nil
}
