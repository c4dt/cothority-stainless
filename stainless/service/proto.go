package stainless

import ()

// PROTOSTART
// package stainless;
//
// option java_package = "ch.epfl.dedis.lib.proto";
// option java_outer_classname = "StatusProto";

// VerificationRequest asks the Stainless service to perform verification of contracts
type VerificationRequest struct {
	SourceFiles map[string]string
}

// VerificationResponse is the result of a Stainless verification
type VerificationResponse struct {
	Console string
	Report  string
}

// BytecodeGenRequest asks the Stainless service to generate Ethereum bytecode
// of contracts
type BytecodeGenRequest struct {
	SourceFiles map[string]string
}

// BytecodeObj is the combination of the binary code and the ABI
type BytecodeObj struct {
	Abi string
	Bin string
}

// BytecodeGenResponse is the result of a Stainless bytecode generation
type BytecodeGenResponse struct {
	BytecodeObjs map[string]*BytecodeObj
}

type DeployRequest struct {
	GasLimit uint64
	GasPrice uint64
	Amount   uint64
	Bytecode []byte
	Abi      string   // JSON-encoded
	Args     []string // JSON-encoded
}

type TransactionRequest struct {
	GasLimit        uint64
	GasPrice        uint64
	Amount          uint64
	ContractAddress []byte
	Nonce           uint64
	Abi             string // JSON-encoded
	Method          string
	Args            []string // JSON-encoded
}

type TransactionHashResponse struct {
	Transaction     []byte
	TransactionHash []byte
}

type TransactionFinalizationRequest struct {
	Transaction []byte
	Signature   []byte
}

type TransactionResponse struct {
	Transaction []byte
}
