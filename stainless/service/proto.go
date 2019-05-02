package stainless

import ()

// PROTOSTART
// package stainless;
//
// option java_package = "ch.epfl.dedis.lib.proto";
// option java_outer_classname = "StatusProto";

// VerificationRequest asks the Stainless service to perform verification of contracts
type VerificationRequest struct {
	// Add command to be "verify" or "generate bytecode"
	SourceFiles map[string]string
}

// VerificationResponse is the result of a Stainless verification
type VerificationResponse struct {
	Console string
	Report  string
}
