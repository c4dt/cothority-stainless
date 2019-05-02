// Package stainless is a service for executing stainless verification and
// Ethereum bytecode generation on smart contracts written in a subset of
// Scala.
package stainless

import (
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"

	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	stainlessProg     = "stainless-smart"
	stainlessCacheDir = "/tmp/stainless-cache-dir"
	stainlessTimeout  = 60 * time.Second
)

// ServiceName is the name to refer to the Stainless service.
const ServiceName = "Stainless"

func init() {
	onet.RegisterNewService(ServiceName, newStainlessService)
	network.RegisterMessage(&VerificationRequest{})
	network.RegisterMessage(&VerificationResponse{})
}

// Stainless is the service that performs stainless operations.
type Stainless struct {
	*onet.ServiceProcessor
}

func stainlessVerify(sourceFiles map[string]string) (string, string, error) {
	// Handle explicitely the case of no source file
	if len(sourceFiles) == 0 {
		return "", "", nil
	}

	// Ensure Stainless cache directory exists
	err := os.MkdirAll(stainlessCacheDir, 0755)
	if err != nil {
		return "", "", err
	}

	// Create temporary working directory for isolated execution
	dir, err := ioutil.TempDir("", "stainless-")
	if err != nil {
		return "", "", err
	}
	defer os.RemoveAll(dir)

	// Create source files in working directory
	var filenames []string
	for filename, contents := range sourceFiles {
		err = ioutil.WriteFile(filepath.Join(dir, filename), []byte(contents), 0644)
		if err != nil {
			return "", "", err
		}
		filenames = append(filenames, filename)
	}

	ctx, cancel := context.WithTimeout(context.Background(), stainlessTimeout)
	defer cancel()

	// Build stainless arguments
	args := append([]string{
		"--json",
		fmt.Sprintf("--cache-dir=%s", stainlessCacheDir),
	}, filenames...)

	// Build command
	cmd := exec.CommandContext(ctx, stainlessProg, args...)
	cmd.Dir = dir

	// Execute command and retrieve console output
	console, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("%s\nConsole:\n%s", err.Error(), console)
	}

	// Read JSON report
	report, err := ioutil.ReadFile(filepath.Join(dir, "report.json"))
	if err != nil {
		log.LLvl4("Error reading JSON report:", err)
		return "", "", err
	}

	return string(console), string(report), nil
}

// Verify performs a Stainless contract verification
func (service *Stainless) Verify(req *VerificationRequest) (network.Message, error) {
	console, report, err := stainlessVerify(req.SourceFiles)
	if err != nil {
		return nil, err
	}

	log.Lvl4("Returning", console, report)

	return &VerificationResponse{
		Console: console,
		Report:  report,
	}, nil
}

// newStainlessService creates a new service that is built for Status
func newStainlessService(context *onet.Context) (onet.Service, error) {
	service := &Stainless{
		ServiceProcessor: onet.NewServiceProcessor(context),
	}
	err := service.RegisterHandler(service.Verify)
	if err != nil {
		return nil, err
	}

	return service, nil
}
