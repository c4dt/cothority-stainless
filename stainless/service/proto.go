package stainless

import ()

// PROTOSTART
// FIXME

// Request is what the Stainless service is expected to receive from clients.
type Request struct {
	// Add command to be "verify" or "generate bytecode"
	SourceFiles map[string]string
}

// Response is what the Stainless service will reply to clients.
type Response struct {
	Console string
	Report  map[string]interface{}
}
