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
	executable = "stainless-smart"
	reportName = "report.json"
	cacheDir   = "/tmp/stainless-cache-dir"
	timeout    = 60 * time.Second
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

func verify(sourceFiles map[string]string) (string, string, error) {
	// Ensure Stainless cache directory exists
	err := os.MkdirAll(cacheDir, 0755)
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

	// Build stainless arguments
	args := append([]string{
		"--json",
		fmt.Sprintf("--cache-dir=%s", cacheDir),
	}, filenames...)

	// Build command
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = dir

	// Execute command and retrieve console output
	console, execErr := cmd.Output()

	// If no report was produced, a serious error happened
	reportFile := filepath.Join(dir, reportName)
	if _, err := os.Stat(reportFile); os.IsNotExist(err) {
		return "", "", fmt.Errorf("%s\nConsole:\n%s", execErr.Error(), console)
	}

	// Read JSON report
	report, err := ioutil.ReadFile(filepath.Join(dir, "report.json"))
	if err != nil {
		log.LLvl4("Error reading JSON report:", err)
		return "", "", err
	}
	// If the report is empty, verification could not proceed normally
	if string(report) == "{}" {
		return "", "", fmt.Errorf("Error in Stainless execution -- Console:\n%s", console)
	}

	// Verification was performed, and its results are contained in the report
	return string(console), string(report), nil
}

func genBytecode(sourceFiles map[string]string) (map[string]BytecodeObj, error) {
	bc := make(map[string]BytecodeObj)

	return bc, nil
}

// Verify performs a Stainless contract verification
func (service *Stainless) Verify(req *VerificationRequest) (network.Message, error) {
	console, report, err := verify(req.SourceFiles)
	if err != nil {
		return nil, err
	}

	log.Lvl4("Returning", console, report)

	return &VerificationResponse{
		Console: console,
		Report:  report,
	}, nil
}

// GenBytecode generates bytecode from Stainless contracts
func (service *Stainless) GenBytecode(req *BytecodeGenRequest) (network.Message, error) {
	bytecodeObjs, err := genBytecode(req.SourceFiles)
	if err != nil {
		return nil, err
	}

	log.Lvl4("Returning", bytecodeObjs)

	return &BytecodeGenResponse{
		BytecodeObjs: bytecodeObjs,
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
	err = service.RegisterHandler(service.GenBytecode)
	if err != nil {
		return nil, err
	}

	return service, nil
}
