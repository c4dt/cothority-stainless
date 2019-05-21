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

// GenBytecode sends a bytecode generation request
func (c *Client) GenBytecode(dst *network.ServerIdentity, sourceFiles map[string]string) (*BytecodeGenResponse, error) {
	response := &BytecodeGenResponse{}

	err := c.SendProtobuf(dst, &BytecodeGenRequest{SourceFiles: sourceFiles}, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (c *Client) DeployContract(dst *network.ServerIdentity, gasLimit uint64, gasPrice uint64, amount uint64, bytecode []byte, abi string, args ...string) (*TransactionHashResponse, error) {
	request := &DeployRequest{
		GasLimit: gasLimit,
		GasPrice: gasPrice,
		Amount:   amount,
		Bytecode: bytecode,
		Abi:      abi,
		Args:     args,
	}
	response := &TransactionHashResponse{}

	err := c.SendProtobuf(dst, request, response)
	if err != nil {
		return nil, err
	}

	return response, err
}

func (c *Client) ExecuteTransaction(dst *network.ServerIdentity, gasLimit uint64, gasPrice uint64, amount uint64, contractAddress []byte, nonce uint64, abi string, method string, args ...string) (*TransactionHashResponse, error) {
	request := &TransactionRequest{
		GasLimit:        gasLimit,
		GasPrice:        gasPrice,
		Amount:          amount,
		ContractAddress: contractAddress,
		Nonce:           nonce,
		Abi:             abi,
		Method:          method,
		Args:            args,
	}
	response := &TransactionHashResponse{}

	err := c.SendProtobuf(dst, request, response)
	if err != nil {
		return nil, err
	}

	return response, err
}

func (c *Client) FinalizeTransaction(dst *network.ServerIdentity, tx []byte, signature []byte) (*TransactionResponse, error) {
	request := &TransactionFinalizationRequest{
		Transaction: tx,
		Signature:   signature,
	}
	response := &TransactionResponse{}

	err := c.SendProtobuf(dst, request, response)
	if err != nil {
		return nil, err
	}

	return response, err
}
