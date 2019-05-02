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

// Verify sends a verification request
func (c *Client) Verify(dst *network.ServerIdentity, sourceFiles map[string]string) (*VerificationResponse, error) {
	response := &VerificationResponse{}

	err := c.SendProtobuf(dst, &VerificationRequest{SourceFiles: sourceFiles}, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}
