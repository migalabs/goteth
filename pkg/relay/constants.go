package relay

const (
	mainnetUltraSoundRelay         string = "https://relay.ultrasound.money/"
	mainnetBloxRouteMaxProfitRelay string = "https://0x8b5d2e73e2a3a55c6c87b8b6eb92e0149a125c852751db1422fa951e42a09b82c142c3ea98d0d9930b056a3bc9896b8f@bloxroute.max-profit.blxrbdn.com"
	mainnetAgnosticRelay           string = "https://agnostic-relay.net/"
	mainnetFlashbotsRelay          string = "https://0xac6e77dfe25ecd6110b8e780608cce0dab71fdd5ebea22a16c0205200f2f8e2e3ad3b71d3499c54ad14d6c21b41a37ae@boost-relay.flashbots.net"
	mainnetBloxRouteRegulatedRelay string = "https://0xb0b07cd0abef743db4260b0ed50619cf6ad4d82064cb4fbec9d3ec530f7c5e6793d9f286c4e082c0244ffb9f2658fe88@bloxroute.regulated.blxrbdn.com"
	mainnetAestusRelay             string = "https://0xa15b52576bcbf1072f4a011c0f99f9fb6c66f3e1ff321f11f461d15e31b1cb359caa092c71bbded0bae5b5ea401aab7e@aestus.live"
	mainnetManifoldRelay           string = "https://mainnet-relay.securerpc.com"
	mainnetEdenNetworkRelay        string = "https://0xb3ee7afcf27f1f1259ac1787876318c6584ee353097a50ed84f51a1f21a323b3736f271a895c7ce918c038e4265918be@relay.edennetwork.io"
)

var mainnetRelayList []string = []string{
	mainnetUltraSoundRelay,
	mainnetBloxRouteMaxProfitRelay,
	mainnetAgnosticRelay,
	mainnetFlashbotsRelay,
	mainnetBloxRouteRegulatedRelay,
	mainnetAestusRelay,
	mainnetManifoldRelay,
	mainnetEdenNetworkRelay,
}

const (
	holeskyUltraSoundRelay string = "https://0xb1559beef7b5ba3127485bbbb090362d9f497ba64e177ee2c8e7db74746306efad687f2cf8574e38d70067d40ef136dc@relay-stag.ultrasound.money"
	holeskyBloxRouteRelay  string = "https://0x821f2a65afb70e7f2e820a925a9b4c80a159620582c1766b1b09729fec178b11ea22abb3a51f07b288be815a1a2ff516@bloxroute.holesky.blxrbdn.com"
	holeskyFlashbotsRelay  string = "https://0xafa4c6985aa049fb79dd37010438cfebeb0f2bd42b115b89dd678dab0670c1de38da0c4e9138c9290a398ecd9a0b3110@boost-relay-holesky.flashbots.net"
	holeskyAestusRelay     string = "https://0xab78bf8c781c58078c3beb5710c57940874dd96aef2835e7742c866b4c7c0406754376c2c8285a36c630346aa5c5f833@holesky.aestus.live"
	holeskyTitanRelay      string = "https://0xaa58208899c6105603b74396734a6263cc7d947f444f396a90f7b7d3e65d102aec7e5e5291b27e08d02c50a050825c2f@holesky.titanrelay.xyz"
)

var holeskyRelayList []string = []string{
	holeskyUltraSoundRelay,
	holeskyBloxRouteRelay,
	holeskyFlashbotsRelay,
	holeskyAestusRelay,
	holeskyTitanRelay,
}
