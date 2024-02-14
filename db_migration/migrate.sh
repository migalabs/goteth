source .env

array=( t_block_metrics t_epoch_metrics_summary t_eth2_pubkeys t_finalized_checkpoint t_genesis t_orphans t_pool_summary t_proposer_duties t_reorgs t_transactions t_validator_last_status t_validator_rewards_summary t_withdrawals )
for table in "${array[@]}"
do
        date
        echo "Migrating $table..."
        echo $PS_ENDPOINT, $PS_DB, $table, $PS_USER, $PS_PASSWORD
        clickhouse-client --host $CH_HOST \
        --port $CH_PORT \
        -d $CH_DB \
        --user $CH_USER \
        --password $CH_PASSWORD \
        --query "INSERT INTO $table SELECT * FROM postgresql('$PS_ENDPOINT', '$PS_DB', '$table', '$PS_USER', '$PS_PASSWORD')" \
        --verbose --progress
        echo done
done
date
