package stainless

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.dedis.ch/kyber/v3/suites"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
)

var tSuite = suites.MustFind("Ed25519")

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func NewTestClient(l *onet.LocalTest) *Client {
	return &Client{Client: l.NewClient(ServiceName)}
}

func Test_NoSource(t *testing.T) {
	local := onet.NewTCPTest(tSuite)

	// Generate 1 host, don't connect, process messages, don't register the
	// tree or entitylist
	_, ro, _ := local.GenTree(1, false)
	defer local.CloseAll()

	client := NewTestClient(local)

	log.Lvl1("Sending request to service...")
	sourceFiles := map[string]string{}

	response, err := client.Verify(ro.List[0], sourceFiles)
	log.ErrFatal(err)

	assert.Empty(t, response.Console)
	assert.Empty(t, response.Report)
}

func Test_BasicContract(t *testing.T) {
	local := onet.NewTCPTest(tSuite)

	// Generate 1 host, don't connect, process messages, don't register the
	// tree or entitylist
	_, ro, _ := local.GenTree(1, false)
	defer local.CloseAll()

	client := NewTestClient(local)

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
	log.ErrFatal(err)

	log.Lvl1("Response:\n", response)

	assert.NotEmpty(t, response.Console)
	assert.NotEmpty(t, response.Report)
}
