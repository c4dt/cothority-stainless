// Package stainless is a service for executing stainless verification and
// Ethereum bytecode generation on smart contracts written in a subset of
// Scala.
package stainless

// FIXME: Add info into README regarding what to install on the server, i.e.
// stainless-smart and solcjs@0.4.24

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
	"strings"
	"time"
)

const (
	stainlessCmd = "stainless-smart"
	solCompiler  = "solcjs"
	reportName   = "report.json"
	cacheDir     = "/tmp/stainless-cache-dir"
	timeout      = 60 * time.Second
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

func createSourceFiles(dir string, sourceFiles map[string]string) ([]string, error) {
	var filenames []string

	for filename, contents := range sourceFiles {
		err := ioutil.WriteFile(filepath.Join(dir, filename), []byte(contents), 0644)
		if err != nil {
			return nil, err
		}
		filenames = append(filenames, filename)
	}

	return filenames, nil
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
	filenames, err := createSourceFiles(dir, sourceFiles)
	if err != nil {
		return "", "", err
	}

	// Build stainless arguments
	args := append([]string{
		"--json",
		fmt.Sprintf("--cache-dir=%s", cacheDir),
	}, filenames...)

	// Build command
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, stainlessCmd, args...)
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

func compileToSolidity(dir string, sourceFilenames []string) ([]string, error) {
	// % stainless-smart --solidity *scala

	// Build stainless arguments
	args := append([]string{
		"--solidity",
	}, sourceFilenames...)

	// Build command
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, stainlessCmd, args...)
	cmd.Dir = dir

	// Execute command and retrieve stdout
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error; stdout = \n%s", out)
		// return nil, err
	}

	// Find produced Solidity files
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var solidityFilenames []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sol") {
			solidityFilenames = append(solidityFilenames, file.Name())
		}
	}

	return solidityFilenames, nil
}

func compileToBytecode(dir string, sourceFilenames []string, destDir string) error {
	// % solcjs --bin --abi --output-dir OUT_DIR [SOLIDITY_FILE...]

	// Each SOLIDITY_FILE needs to be given with full path due to
	// https://github.com/ethereum/solc-js/issues/114
	var sourceFilepaths []string
	for _, f := range sourceFilenames {
		sourceFilepaths = append(sourceFilepaths, filepath.Join(dir, f))
	}

	// Build Solidity compiler arguments
	args := append([]string{
		"--bin",
		"--abi",
		"--output-dir", destDir,
	}, sourceFilepaths...)

	// Build command
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, solCompiler, args...)
	cmd.Dir = dir

	// Execute command and retrieve stdout
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error; stdout = \n%s", out)
		return err
	}

	return nil
}

func findGeneratedFile(dir string, solFile string, suffix string) (string, error) {
	pattern := fmt.Sprintf(`*%s_sol*.%s`, solFile[:len(solFile)-4], suffix)

	genFiles, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return "", err
	}
	if len(genFiles) != 1 {
		return "", fmt.Errorf("Expected 1 generated '%s' file, got %v", suffix, genFiles)
	}

	contents, err := ioutil.ReadFile(genFiles[0])
	if err != nil {
		return "", err
	}

	return string(contents), nil
}

func genBytecode(sourceFiles map[string]string) (map[string]*BytecodeObj, error) {
	// Create temporary working directory for isolated execution
	dir, err := ioutil.TempDir("", "stainless-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	// Create source files in working directory
	sourceFilenames, err := createSourceFiles(dir, sourceFiles)
	if err != nil {
		return nil, err
	}

	solFilenames, err := compileToSolidity(dir, sourceFilenames)
	if err != nil {
		return nil, err
	}

	bytecodeDir := filepath.Join(dir, "out")
	err = compileToBytecode(dir, solFilenames, bytecodeDir)
	if err != nil {
		return nil, err
	}

	// Build bytecode map
	bc := make(map[string]*BytecodeObj)

	for _, solFile := range solFilenames {
		abiFile, err := findGeneratedFile(bytecodeDir, solFile, "abi")
		if err != nil {
			return nil, err
		}

		binFile, err := findGeneratedFile(bytecodeDir, solFile, "bin")
		if err != nil {
			return nil, err
		}

		bc[solFile] = &BytecodeObj{Abi: string(abiFile), Bin: string(binFile)}
	}

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
