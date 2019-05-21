package stainless

import (
	"encoding/json"
	"testing"

	"encoding/hex"
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

	expectedAbi := `[{"constant":true,"inputs":[],"name":"test","outputs":[],"payable":false,"stateMutability":"pure","type":"function"},{"inputs":[],"payable":false,"stateMutability":"nonpayable","type":"constructor"}]`

	response, err := client.GenBytecode(ro.List[0], sourceFiles)
	assert.Nil(t, err)

	log.Lvl1("Response:\n", response)

	assert.Contains(t, response.BytecodeObjs, "PositiveUint.sol")

	generated := response.BytecodeObjs["PositiveUint.sol"]

	assert.Equal(t, expectedAbi, generated.Abi)

	// The contents of the bin file does not seem deterministic (last 68 bytes changing?)
	assert.NotEmpty(t, generated.Bin)
}

func Test_Deploy(t *testing.T) {
	local, ro, client := setupTest()
	defer teardownTest(local)

	log.Lvl1("Sending request to service...")

	// Deploy a Candy contract with 100 candies.
	// The expected values are taken from an execution using the BEvmClient.

	candyBytecode, err := hex.DecodeString("608060405234801561001057600080fd5b506040516020806101cb833981018060405281019080805190602001909291905050508060008190555080600181905550600060028190555050610172806100596000396000f30060806040526004361061004c576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff168063a1ff2f5214610051578063ea319f281461007e575b600080fd5b34801561005d57600080fd5b5061007c600480360381019080803590602001909291905050506100a9565b005b34801561008a57600080fd5b5061009361013c565b6040518082815260200191505060405180910390f35b6001548111151515610123576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260058152602001807f6572726f7200000000000000000000000000000000000000000000000000000081525060200191505060405180910390fd5b8060015403600181905550806002540160028190555050565b60006001549050905600a165627a7a723058207721a45f17c0e0f57e255f33575281d17f1a90d3d58b51688230d93c460a19aa0029")
	assert.Nil(t, err)

	candyAbi := `[{"constant":false,"inputs":[{"name":"candies","type":"uint256"}],"name":"eatCandy","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"getRemainingCandies","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"inputs":[{"name":"_candies","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"}]`

	candySupply, err := json.Marshal(100)
	assert.Nil(t, err)

	response, err := client.DeployContract(ro.List[0], 1e7, 1, 0, candyBytecode, candyAbi, string(candySupply))
	assert.Nil(t, err)

	expectedTx, err := hex.DecodeString("7b226e6f6e6365223a22307830222c226761735072696365223a22307831222c22676173223a223078393839363830222c22746f223a6e756c6c2c2276616c7565223a22307830222c22696e707574223a22307836303830363034303532333438303135363130303130353736303030383066643562353036303430353136303230383036313031636238333339383130313830363034303532383130313930383038303531393036303230303139303932393139303530353035303830363030303831393035353530383036303031383139303535353036303030363030323831393035353530353036313031373238303631303035393630303033393630303066333030363038303630343035323630303433363130363130303463353736303030333537633031303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303039303034363366666666666666663136383036336131666632663532313436313030353135373830363365613331396632383134363130303765353735623630303038306664356233343830313536313030356435373630303038306664356235303631303037633630303438303336303338313031393038303830333539303630323030313930393239313930353035303530363130306139353635623030356233343830313536313030386135373630303038306664356235303631303039333631303133633536356236303430353138303832383135323630323030313931353035303630343035313830393130333930663335623630303135343831313131353135313536313031323335373630343035313766303863333739613030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303831353236303034303138303830363032303031383238313033383235323630303538313532363032303031383037663635373237323666373230303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303038313532353036303230303139313530353036303430353138303931303339306664356238303630303135343033363030313831393035353530383036303032353430313630303238313930353535303530353635623630303036303031353439303530393035363030613136353632376137613732333035383230373732316134356631376330653066353765323535663333353735323831643137663161393064336435386235313638383233306439336334363061313961613030323930303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303634222c2276223a22307830222c2272223a22307830222c2273223a22307830222c2268617368223a22307837666631383834633430633664636561653534666361346331356131333063356133663639373032643466336537356665336163373862313735656339356139227d")
	assert.Nil(t, err)

	expectedHash, err := hex.DecodeString("c289e67875d147429d2ffc5cc58e9a1486d581bef5aeca63017ad7855f8dab26")
	assert.Nil(t, err)

	assert.Equal(t, expectedTx, response.Transaction)
	assert.Equal(t, expectedHash, response.TransactionHash)
}

func Test_Transaction(t *testing.T) {
	local, ro, client := setupTest()
	defer teardownTest(local)

	log.Lvl1("Sending request to service...")

	// Call eatCandy(10) on a Candy contract deployed at 0x8cdaf0cd259887258bc13a92c0a6da92698644c0.
	// The expected values are taken from an execution using the BEvmClient.

	contractAddress, err := hex.DecodeString("8cdaf0cd259887258bc13a92c0a6da92698644c0")
	assert.Nil(t, err)

	candyAbi := `[{"constant":false,"inputs":[{"name":"candies","type":"uint256"}],"name":"eatCandy","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"getRemainingCandies","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"inputs":[{"name":"_candies","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"}]`

	candiesToEat, err := json.Marshal(10)
	assert.Nil(t, err)

	nonce := uint64(1) // First call right after deployment

	response, err := client.ExecuteTransaction(ro.List[0], 1e7, 1, 0, contractAddress, nonce, candyAbi, "eatCandy", string(candiesToEat))
	assert.Nil(t, err)

	expectedTx, err := hex.DecodeString("7b226e6f6e6365223a22307831222c226761735072696365223a22307831222c22676173223a223078393839363830222c22746f223a22307838636461663063643235393838373235386263313361393263306136646139323639383634346330222c2276616c7565223a22307830222c22696e707574223a223078613166663266353230303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303061222c2276223a22307830222c2272223a22307830222c2273223a22307830222c2268617368223a22307865343264343137386465303032323636386433326637383033666564353637376437343666393238666465386430656339303532656432306138616466343362227d")
	assert.Nil(t, err)

	expectedHash, err := hex.DecodeString("e13b1cfe8797fa11bd7929158008033e585d302a6f4cb11cfcf2b0a8bebec3fd")
	assert.Nil(t, err)

	assert.Equal(t, expectedTx, response.Transaction)
	assert.Equal(t, expectedHash, response.TransactionHash)
}

func Test_FinalizeTx(t *testing.T) {
	local, ro, client := setupTest()
	defer teardownTest(local)

	log.Lvl1("Sending request to service...")

	// Finalize a transaction combining the unsigned transaction and the signature.
	// The expected values are taken from an execution using the BEvmClient.

	// Unsigned transaction of Candy.eatCandy(10) (see Test_Transaction())
	unsignedTx, err := hex.DecodeString("7b226e6f6e6365223a22307831222c226761735072696365223a22307831222c22676173223a223078393839363830222c22746f223a22307838636461663063643235393838373235386263313361393263306136646139323639383634346330222c2276616c7565223a22307830222c22696e707574223a223078613166663266353230303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303061222c2276223a22307830222c2272223a22307830222c2273223a22307830222c2268617368223a22307865343264343137386465303032323636386433326637383033666564353637376437343666393238666465386430656339303532656432306138616466343362227d")
	assert.Nil(t, err)

	// Signature done with private key 0xc87509a1c067bbde78beb793e6fa76530b6382a4c0241e5e4a9ec0a0f44dc0d3
	signature, err := hex.DecodeString("aa0b243e4ad97b6cb7c2a016567aa02b2e7bed159c221b7089b60688527f6e88679c9dfcb1ceb2477a36753645b564c2a14a7bc757f46b9b714c49a4c93ea0a401")
	assert.Nil(t, err)

	expectedTx, err := hex.DecodeString("7b226e6f6e6365223a22307831222c226761735072696365223a22307831222c22676173223a223078393839363830222c22746f223a22307838636461663063643235393838373235386263313361393263306136646139323639383634346330222c2276616c7565223a22307830222c22696e707574223a223078613166663266353230303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303061222c2276223a2230783163222c2272223a22307861613062323433653461643937623663623763326130313635363761613032623265376265643135396332323162373038396236303638383532376636653838222c2273223a22307836373963396466636231636562323437376133363735333634356235363463326131346137626337353766343662396237313463343961346339336561306134222c2268617368223a22307834633966336134343361663030326438373839666235616239393261376631346639396134303762616532613332643464653830313037366365613065353631227d")
	assert.Nil(t, err)

	response, err := client.FinalizeTransaction(ro.List[0], unsignedTx, signature)
	assert.Nil(t, err)

	assert.Equal(t, expectedTx, response.Transaction)
}
