package clientapi

import "time"

func (s APIClient) RequestGenesis() time.Time {
	genesis, err := s.Api.GenesisTime(s.ctx)
	if err != nil {
		log.Panicf("could not get genesis time: %s", err)
	}

	return genesis
}
