package db

import (
	"github.com/ClickHouse/ch-go/proto"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

var (
	blockRewardsTable       = "t_block_rewards"
	insertBlockRewardsQuery = `
	INSERT INTO %s (
		f_slot,
		f_reward_fees,
		f_burnt_fees,
		f_cl_manual_reward,
		f_cl_api_reward,
		f_relays,
		f_builder_pubkey,
		f_bid_commission)
		VALUES`
)

func blockRewardsInput(blocks []BlockReward) proto.Input {
	// one object per column
	var (
		f_slot             proto.ColUInt64
		f_reward_fees      proto.ColUInt64
		f_burnt_fees       proto.ColUInt64
		f_cl_manual_reward proto.ColUInt64
		f_cl_api_reward    proto.ColUInt64
		f_relays           = new(proto.ColStr).Array()
		f_builder_pubkey   = new(proto.ColStr).Array()
		f_bid_commission   proto.ColUInt64
	)

	for _, blockReward := range blocks {

		f_slot.Append(uint64(blockReward.Slot))
		f_reward_fees.Append(blockReward.RewardFees)
		f_burnt_fees.Append(blockReward.BurntFees)
		f_cl_manual_reward.Append(uint64(blockReward.CLManualReward))
		f_cl_api_reward.Append(uint64(blockReward.CLApiReward))
		f_relays.Append(blockReward.Relays)
		f_builder_pubkey.Append(blockReward.BuilderPubkeys)
		f_bid_commission.Append(blockReward.BidCommision)
	}

	return proto.Input{

		{Name: "f_slot", Data: f_slot},
		{Name: "f_reward_fees", Data: f_reward_fees},
		{Name: "f_burnt_fees", Data: f_burnt_fees},
		{Name: "f_cl_manual_reward", Data: f_cl_manual_reward},
		{Name: "f_cl_api_reward", Data: f_cl_api_reward},
		{Name: "f_relays", Data: f_relays},
		{Name: "f_builder_pubkey", Data: f_builder_pubkey},
		{Name: "f_bid_commission", Data: f_bid_commission},
	}
}

func (p *DBService) PersistBlockRewards(data []BlockReward) error {
	persistObj := PersistableObject[BlockReward]{
		input: blockRewardsInput,
		table: blockRewardsTable,
		query: insertBlockRewardsQuery,
	}

	for _, item := range data {
		persistObj.Append(item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting block rewards: %s", err.Error())
	}
	return err
}

type BlockReward struct {
	Slot           phase0.Slot
	CLManualReward phase0.Gwei // Gwei
	CLApiReward    phase0.Gwei // Gwei
	RewardFees     uint64      // Gwei
	BurntFees      uint64      // Gwei
	Relays         []string
	BuilderPubkeys []string
	BidCommision   uint64
}
