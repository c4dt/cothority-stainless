package stainless

import (
	"encoding/json"
	"testing"

	"fmt"
	"github.com/stretchr/testify/assert"
	"go.dedis.ch/kyber/v3/suites"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
)

var tSuite = suites.MustFind("Ed25519")

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func setupTest() (*onet.LocalTest, *onet.Roster, *Client) {
	local := onet.NewTCPTest(tSuite)

	// Generate 1 host, don't connect, process messages, don't register the
	// tree or entitylist
	_, ro, _ := local.GenTree(1, false)

	client := &Client{Client: local.NewClient(ServiceName)}

	return local, ro, client
}

func teardownTest(local *onet.LocalTest) {
	local.CloseAll()
}

func parseReport(report string) (valid int, invalid int, err error) {
	jsonData := []byte(report)

	var v interface{}
	err = json.Unmarshal(jsonData, &v)
	if err != nil {
		return
	}

	// The JSON schema of the report is a bit convoluted...
	verif := v.(map[string]interface{})["stainless"].([]interface{})[0].([]interface{})[1].([]interface{})[0].([]interface{})
	for _, elem := range verif {
		status := elem.(map[string]interface{})["status"].(map[string]interface{})
		for s := range status {
			switch s {
			case "Valid", "ValidFromCache":
				valid++
				break
			case "Invalid":
				invalid++
				break
			default:
				err = fmt.Errorf("Unknown status: '%s'", s)
				return
			}
		}
	}

	return
}

func Test_NoSource(t *testing.T) {
	local, ro, client := setupTest()
	defer teardownTest(local)

	log.Lvl1("Sending request to service...")
	sourceFiles := map[string]string{}

	response, err := client.Verify(ro.List[0], sourceFiles)
	assert.Nil(t, err)

	log.Lvl1("Response:\n", response)

	valid, invalid, err := parseReport(response.Report)
	assert.Nil(t, err)

	assert.Equal(t, 0, valid)
	assert.Equal(t, 0, invalid)
}

func Test_FailCompilation(t *testing.T) {
	local, ro, client := setupTest()
	defer teardownTest(local)

	log.Lvl1("Sending request to service...")
	sourceFiles := map[string]string{"p.scala": "garbage"}

	_, err := client.Verify(ro.List[0], sourceFiles)
	assert.NotNil(t, err)
}

func Test_BasicContract(t *testing.T) {
	local, ro, client := setupTest()
	defer teardownTest(local)

	log.Lvl1("Sending request to service...")
	sourceFiles := map[string]string{
		"BasicContract1.scala": `
import stainless.smartcontracts._
import stainless.annotation._

object BasicContract1 {
    case class BasicContract1(
        val other: Address
    ) extends Contract {
        @view
        def foo = {
            other
        }
    }
}`,
	}

	response, err := client.Verify(ro.List[0], sourceFiles)
	assert.Nil(t, err)

	log.Lvl1("Response:\n", response)

	valid, invalid, err := parseReport(response.Report)
	assert.Nil(t, err)

	assert.Equal(t, 0, valid)
	assert.Equal(t, 0, invalid)
}

func Test_VerificationPass(t *testing.T) {
	local, ro, client := setupTest()
	defer teardownTest(local)

	log.Lvl1("Sending request to service...")
	sourceFiles := map[string]string{
		"PositiveUint.scala": `
import stainless.smartcontracts._
import stainless.annotation._
import stainless.lang.StaticChecks._

object PositiveUint {
    case class PositiveUint() extends Contract {
            @solidityPure
         def test(@ghost a: Uint256) = {
            assert(a >= Uint256.ZERO)
         }
    }
}`,
	}

	response, err := client.Verify(ro.List[0], sourceFiles)
	assert.Nil(t, err)
	log.ErrFatal(err)

	log.Lvl1("Response:\n", response)

	valid, invalid, err := parseReport(response.Report)
	assert.Nil(t, err)

	assert.Equal(t, 1, valid)
	assert.Equal(t, 0, invalid)
}

func Test_VerificationFail(t *testing.T) {
	local, ro, client := setupTest()
	defer teardownTest(local)

	log.Lvl1("Sending request to service...")
	sourceFiles := map[string]string{
		"Overflow.scala": `
import stainless.smartcontracts._

object Test {
  def f(a: Uint256, b: Uint256) = {
    assert(a + b >= a)
  }
}`,
	}

	response, err := client.Verify(ro.List[0], sourceFiles)
	assert.Nil(t, err)
	log.ErrFatal(err)

	log.Lvl1("Response:\n", response)

	valid, invalid, err := parseReport(response.Report)
	assert.Nil(t, err)

	assert.Equal(t, 0, valid)
	assert.Equal(t, 1, invalid)
}

func Test_BytecodeGen(t *testing.T) {
	local, ro, client := setupTest()
	defer teardownTest(local)

	log.Lvl1("Sending request to service...")
	sourceFiles := map[string]string{
		"PositiveUint.scala": `
import stainless.smartcontracts._
import stainless.annotation._
import stainless.lang.StaticChecks._

object PositiveUint {
    case class PositiveUint() extends Contract {
            @solidityPure
         def test(@ghost a: Uint256) = {
            assert(a >= Uint256.ZERO)
         }
    }
}`,
	}

	abiFile := `[{"constant":true,"inputs":[],"name":"test","outputs":[],"payable":false,"stateMutability":"pure","type":"function"},{"inputs":[],"payable":false,"stateMutability":"nonpayable","type":"constructor"}]`

	response, err := client.GenBytecode(ro.List[0], sourceFiles)
	assert.Nil(t, err)

	log.Lvl1("Response:\n", response)

	assert.Contains(t, response.BytecodeObjs, "PositiveUint.sol")

	generated := response.BytecodeObjs["PositiveUint.sol"]

	assert.Equal(t, abiFile, generated.Abi)

	// The contents of the bin file does not seem deterministic (last 68 bytes changing?)
	assert.NotEmpty(t, generated.Bin)
}
