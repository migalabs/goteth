package relay

const (
	mainnetAestusRelay             string = "https://0xa15b52576bcbf1072f4a011c0f99f9fb6c66f3e1ff321f11f461d15e31b1cb359caa092c71bbded0bae5b5ea401aab7e@aestus.live"
	mainnetAgnosticRelay           string = "https://0xa7ab7a996c8584251c8f925da3170bdfd6ebc75d50f5ddc4050a6fdc77f2a3b5fce2cc750d0865e05d7228af97d69561@agnostic-relay.net"
	mainnetBloxRouteMaxProfitRelay string = "https://0x8b5d2e73e2a3a55c6c87b8b6eb92e0149a125c852751db1422fa951e42a09b82c142c3ea98d0d9930b056a3bc9896b8f@bloxroute.max-profit.blxrbdn.com"
	mainnetBloxRouteRegulatedRelay string = "https://0xb0b07cd0abef743db4260b0ed50619cf6ad4d82064cb4fbec9d3ec530f7c5e6793d9f286c4e082c0244ffb9f2658fe88@bloxroute.regulated.blxrbdn.com"
	mainnetTitanGlobalRelay        string = "https://0x8c4ed5e24fe5c6ae21018437bde147693f68cda427cd1122cf20819c30eda7ed74f72dece09bb313f2a1855595ab677d@global.titanrelay.xyz"
	mainnetTitanRegionalRelay      string = "https://0x8c4ed5e24fe5c6ae21018437bde147693f68cda427cd1122cf20819c30eda7ed74f72dece09bb313f2a1855595ab677d@regional.titanrelay.xyz"
	mainnetFlashbotsRelay          string = "https://0xac6e77dfe25ecd6110b8e780608cce0dab71fdd5ebea22a16c0205200f2f8e2e3ad3b71d3499c54ad14d6c21b41a37ae@boost-relay.flashbots.net"
	mainnetManifoldRelay           string = "https://0x98650451ba02064f7b000f5768cf0cf4d4e492317d82871bdc87ef841a0743f69f0f1eea11168503240ac35d101c9135@mainnet-relay.securerpc.com"
	mainnetUltraSoundRelay         string = "https://0xa1559ace749633b997cb3fdacffb890aeebdb0f5a3b6aaa7eeeaf1a38af0a8fe88b9e4b1f61f236d2e64d95733327a62@relay.ultrasound.money"
	mainnetWenmergeRelay           string = "https://0x8c7d33605ecef85403f8b7289c8058f440cbb6bf72b055dfe2f3e2c6695b6a1ea5a9cd0eb3a7982927a463feb4c3dae2@relay.wenmerge.com"
	mainnetProofRelay              string = "https://0xa44f64faca0209764461b2abfe3533f9f6ed1d51844974e22d79d4cfd06eff858bb434d063e512ce55a1841e66977bfd@proof-relay.ponrelay.com"
)

var mainnetRelayList []string = []string{
	mainnetAestusRelay,
	mainnetAgnosticRelay,
	mainnetBloxRouteMaxProfitRelay,
	mainnetBloxRouteRegulatedRelay,
	mainnetTitanGlobalRelay,
	mainnetTitanRegionalRelay,
	mainnetFlashbotsRelay,
	mainnetManifoldRelay,
	mainnetUltraSoundRelay,
	mainnetWenmergeRelay,
	mainnetProofRelay,
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

const (
	hoodiBloxRouteRelay string = "https://0x821f2a65afb70e7f2e820a925a9b4c80a159620582c1766b1b09729fec178b11ea22abb3a51f07b288be815a1a2ff516@bloxroute.hoodi.blxrbdn.com"
	hoodiFlashbotsRelay string = "https://0xafa4c6985aa049fb79dd37010438cfebeb0f2bd42b115b89dd678dab0670c1de38da0c4e9138c9290a398ecd9a0b3110@boost-relay-hoodi.flashbots.net"
	hoodiUltraSoundRelay string = "https://0xb1559beef7b5ba3127485bbbb090362d9f497ba64e177ee2c8e7db74746306efad687f2cf8574e38d70067d40ef136dc@relay-hoodi.ultrasound.money"
	hoodiTitanRelay     string = "https://0xaa58208899c6105603b74396734a6263cc7d947f444f396a90f7b7d3e65d102aec7e5e5291b27e08d02c50a050825c2f@hoodi.titanrelay.xyz"
	hoodiAestusRelay    string = "https://0x98f0ef62f00780cf8eb06701a7d22725b9437d4768bb19b363e882ae87129945ec206ec2dc16933f31d983f8225772b6@hoodi.aestus.live"
)

var hoodiRelayList []string = []string{
	hoodiBloxRouteRelay,
	hoodiFlashbotsRelay,
	hoodiUltraSoundRelay,
	hoodiTitanRelay,
	hoodiAestusRelay,
}

const (
	sepoliaFlashbotsRelay string = "https://0xafa4c6985aa049fb79dd37010438cfebeb0f2bd42b115b89dd678dab0670c1de38da0c4e9138c9290a398ecd9a0b3110@boost-relay-sepolia.flashbots.net"
)

var sepoliaRelayList []string = []string{
	sepoliaFlashbotsRelay,
}
