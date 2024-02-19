package clientapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

func (s *APIClient) RequestBlockRewards(slot phase0.Slot) (spec.BlockRewards, error) {

	uri := s.Api.Address() + "/eth/v1/beacon/rewards/blocks/" + fmt.Sprintf("%d", slot)
	resp, err := http.Get(uri)
	if err != nil {
		log.Fatalln(err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	var rewards spec.BlockRewards
	err = json.Unmarshal(body, &rewards)

	if err != nil {
		log.Fatalf("error parsing block rewards response: %s", err)
	}

	return rewards, err

}
