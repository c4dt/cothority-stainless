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
	BytecodeObjs map[string]BytecodeObj
}
