package db

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Static postgres queries, for each modification in the tables, the table needs to be reseted
var (
	// wlogrus associated with the postgres db
	PsqlType = "postgres-db"
	wlog     = logrus.WithField(
		"module", PsqlType,
	)
	MAX_BATCH_QUEUE       = 1000
	MAX_EPOCH_BATCH_QUEUE = 1
	batchFlushingTimeout  = time.Duration(1 * time.Second)
)

type PostgresDBService struct {
	// Control Variables
	ctx           context.Context
	cancel        context.CancelFunc
	connectionUrl string // the url might not be necessary (better to remove it?¿)
	psqlPool      *pgxpool.Pool

	writeChan        chan Model // Receive tasks to persist
	endProcess       int32
	FinishSignalChan chan struct{}
	workerNum        int
}

// Connect to the PostgreSQL Database and get the multithread-proof connection
// from the given url-composed credentials
func ConnectToDB(ctx context.Context, url string, chanLength int, workerNum int) (*PostgresDBService, error) {
	mainCtx, cancel := context.WithCancel(ctx)
	// spliting the url to don't share any confidential information on wlogs
	wlog.Infof("Connecting to postgres DB %s", url)
	if strings.Contains(url, "@") {
		wlog.Debugf("Connecting to PostgresDB at %s", strings.Split(url, "@")[1])
	}
	psqlPool, err := pgxpool.Connect(mainCtx, url)
	if err != nil {
		return nil, err
	}
	if strings.Contains(url, "@") {
		wlog.Infof("PostgresDB %s succesfully connected", strings.Split(url, "@")[1])
	}
	// filter the type of network that we are filtering

	psqlDB := &PostgresDBService{
		ctx:              mainCtx,
		cancel:           cancel,
		connectionUrl:    url,
		psqlPool:         psqlPool,
		writeChan:        make(chan Model, chanLength),
		endProcess:       0,
		FinishSignalChan: make(chan struct{}, 1),
		workerNum:        workerNum,
	}
	// init the psql db
	err = psqlDB.init(ctx, psqlDB.psqlPool)
	if err != nil {
		return psqlDB, errors.Wrap(err, "error initializing the tables of the psqldb")
	}
	go psqlDB.runWriters()
	return psqlDB, err
}

// Close the connection with the PostgreSQL
func (p *PostgresDBService) Close() {
	p.psqlPool.Close()
}

func (p *PostgresDBService) init(ctx context.Context, pool *pgxpool.Pool) error {
	// create the tables
	err := p.createRewardsTable()
	if err != nil {
		return err
	}

	err = p.createEpochMetricsTable()
	if err != nil {
		return err
	}

	err = p.createProposerDutiesTable()
	if err != nil {
		return err
	}

	err = p.createBlockMetricsTable()
	if err != nil {
		return err
	}

	err = p.createStatusTable()
	if err != nil {
		return err
	}

	err = p.createPoolsTable()
	if err != nil {
		return err
	}

	err = p.createWithdrawalsTable()
	if err != nil {
		return err
	}

	err = p.createLastStatusTable()
	if err != nil {
		return err
	}

	return nil
}

func (p *PostgresDBService) DoneTasks() {
	atomic.AddInt32(&p.endProcess, int32(1))
	wlog.Infof("Received finish signal")
}

func (p *PostgresDBService) runWriters() {
	var wgDBWriters sync.WaitGroup
	wlog.Info("Launching Beacon State Writers")
	wlog.Infof("Launching %d Beacon State Writers", p.workerNum)
	for i := 0; i < p.workerNum; i++ {
		wgDBWriters.Add(1)
		go func(dbWriterID int) {
			batch := pgx.Batch{}
			defer wgDBWriters.Done()
			wlogWriter := wlog.WithField("DBWriter", dbWriterID)
			// batch flushing ticker
			ticker := time.NewTicker(batchFlushingTimeout)
		loop:
			for {

				select {

				case task := <-p.writeChan:

					wlogWriter.Tracef("Received new write task")
					var q string
					var args []interface{}
					var err error

					switch task.Type() {
					case spec.BlockModel:
						q, args = BlockOperation(task.(spec.AgnosticBlock))
					case spec.EpochModel:
						q, args = EpochOperation(task.(spec.Epoch))
					case spec.PoolSummaryModel:
						q, args = PoolOperation(task.(spec.PoolSummary))
					case spec.ProposerDutyModel:
						q, args = ProposerDutyOperation(task.(spec.ProposerDuty))
					case spec.ValidatorLastStatusModel:
						q, args = ValidatorLastStatusOperation(task.(spec.ValidatorLastStatus))
					case spec.ValidatorRewardsModel:
						q, args = ValidatorOperation(task.(spec.ValidatorRewards))
					case spec.WithdrawalModel:
						q, args = WithdrawalOperation(task.(spec.Withdrawal))
					default:
						err = fmt.Errorf("could not figure out the type of write task")
					}

					if err != nil {
						wlog.Errorf("could not process incoming task, %s", err)
					} else {
						batch.Queue(q, args...)
					}

				case <-p.ctx.Done():
					wlogWriter.Info("shutdown detected, closing persister")
					break loop
				case <-ticker.C:
					// if limit reached or no more queue and pending tasks
					if batch.Len() > MAX_BATCH_QUEUE ||
						(len(p.writeChan) == 0 && batch.Len() > 0) {

						wlog.Tracef("Sending batch to be stored...")

						err := p.ExecuteBatch(batch)
						if err != nil {
							wlogWriter.Errorf("Error processing batch", err.Error())
						}
						batch = pgx.Batch{}
					}

					if p.endProcess >= 1 && len(p.writeChan) == 0 {
						wlogWriter.Info("shutdown detected, closing persister")
						break loop
					}
				}
			}
			wlogWriter.Debugf("DB Writer finished...")

		}(i)
	}

	wgDBWriters.Wait()

}

func (p PostgresDBService) ExecuteBatch(batch pgx.Batch) error {

	snapshot := time.Now()
	tx, err := p.psqlPool.Begin(p.ctx)
	if err != nil {
		panic(err)
	}

	batchResults := tx.SendBatch(p.ctx, &batch)

	var qerr error
	var rows pgx.Rows
	batchIdx := 0
	for qerr == nil {
		rows, qerr = batchResults.Query()
		rows.Close()
		batchIdx += 1
	}
	if qerr.Error() != "no result" {
		wlog.Errorf("Error executing batch, error: %s", qerr.Error())
	} else {
		wlog.Tracef("Batch process time: %f, batch size: %d", time.Since(snapshot).Seconds(), batch.Len())
	}

	return tx.Commit(p.ctx)

}

func (p *PostgresDBService) Persist(w Model) {
	p.writeChan <- w
}

type Model interface { // simply to enforce a Model interface
	// For now we simply support insert operations
	Type() spec.ModelType // whether insert is activated for this model
}
