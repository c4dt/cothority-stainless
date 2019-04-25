package stainless

import (
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/network"
)

// Client is a structure to communicate with stainless service
type Client struct {
	*onet.Client
}

// NewClient makes a new Client
func NewClient() *Client {
	return &Client{Client: onet.NewClient(cothority.Suite, ServiceName)}
}

// Request sends a request to the cothority
func (c *Client) Request(dst *network.ServerIdentity, sourceFiles map[string]string) (*Response, error) {
	response := &Response{}

	err := c.SendProtobuf(dst, &Request{SourceFiles: sourceFiles}, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}
