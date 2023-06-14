package db

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
	"github.com/jackc/pgx/v4/pgxpool"
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
)

type PostgresDBService struct {
	// Control Variables
	ctx           context.Context
	connectionUrl string // the url might not be necessary (better to remove it?¿)
	psqlPool      *pgxpool.Pool
	wgDBWriters   sync.WaitGroup

	writeChan chan Model // Receive tasks to persist
	stop      bool
	workerNum int
}

// Connect to the PostgreSQL Database and get the multithread-proof connection
// from the given url-composed credentials
func ConnectToDB(ctx context.Context, url string, workerNum int) (*PostgresDBService, error) {
	// spliting the url to don't share any confidential information on wlogs
	wlog.Infof("Connecting to postgres DB %s", url)
	if strings.Contains(url, "@") {
		wlog.Debugf("Connecting to PostgresDB at %s", strings.Split(url, "@")[1])
	}
	psqlPool, err := pgxpool.Connect(ctx, url)
	if err != nil {
		return nil, err
	}
	if strings.Contains(url, "@") {
		wlog.Infof("PostgresDB %s succesfully connected", strings.Split(url, "@")[1])
	}

	psqlDB := &PostgresDBService{
		ctx:           ctx,
		connectionUrl: url,
		psqlPool:      psqlPool,
		writeChan:     make(chan Model, workerNum),
		workerNum:     workerNum,
	}
	// init the psql db
	psqlDB.makeMigrations()
	go psqlDB.runWriters()
	return psqlDB, err
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

	err = p.createTransactionsTable()
	if err != nil {
		return err
	}

	return nil
}

func (p *PostgresDBService) Finish() {
	p.stop = true
	p.wgDBWriters.Wait()
	wlog.Infof("Routines finished...")
	wlog.Infof("closing connection to database server...")
	p.psqlPool.Close()
	wlog.Infof("connection to database server closed...")
	close(p.writeChan)
}

func (p *PostgresDBService) runWriters() {

	wlog.Info("Launching Beacon State Writers")
	wlog.Infof("Launching %d Beacon State Writers", p.workerNum)
	for i := 0; i < p.workerNum; i++ {
		p.wgDBWriters.Add(1)
		go func(dbWriterID int) {
			defer p.wgDBWriters.Done()
			batcher := NewQueryBatch(p.ctx, p.psqlPool, MAX_BATCH_QUEUE)
			wlogWriter := wlog.WithField("DBWriter", dbWriterID)
			ticker := time.NewTicker(utils.RoutineFlushTimeout)
		loop:
			for {
				select {
				case task := <-p.writeChan:

					var err error
					persis := NewPersistable()

					switch task.Type() {
					case spec.BlockModel:
						q, args := BlockOperation(task.(spec.AgnosticBlock))
						persis.query = q
						persis.values = append(persis.values, args...)
					case spec.EpochModel:
						q, args := EpochOperation(task.(spec.Epoch))
						persis.query = q
						persis.values = append(persis.values, args...)
					case spec.PoolSummaryModel:
						q, args := PoolOperation(task.(spec.PoolSummary))
						persis.query = q
						persis.values = append(persis.values, args...)
					case spec.ProposerDutyModel:
						q, args := ProposerDutyOperation(task.(spec.ProposerDuty))
						persis.query = q
						persis.values = append(persis.values, args...)
					case spec.ValidatorLastStatusModel:
						q, args := ValidatorLastStatusOperation(task.(spec.ValidatorLastStatus))
						persis.query = q
						persis.values = append(persis.values, args...)
					case spec.ValidatorRewardsModel:
						q, args := ValidatorOperation(task.(spec.ValidatorRewards))
						persis.query = q
						persis.values = append(persis.values, args...)
					case spec.WithdrawalModel:
						q, args := WithdrawalOperation(task.(spec.Withdrawal))
						persis.query = q
						persis.values = append(persis.values, args...)
					case spec.TransactionsModel:
						q, args := TransactionOperation(task.(*spec.AgnosticTransaction))
						persis.query = q
						persis.values = append(persis.values, args...)
					default:
						err = fmt.Errorf("could not figure out the type of write task")
						wlog.Errorf("could not process incoming task, %s", err)
					}
					// ckeck if there is any new query to add
					if !persis.isEmpty() {
						batcher.AddQuery(persis)
					}
					// check if we can flush the batch of queries
					if batcher.IsReadyToPersist() {
						err := batcher.PersistBatch()
						if err != nil {
							wlogWriter.Errorf("Error processing batch", err.Error())
						}
					}

				case <-p.ctx.Done():
					break loop

				case <-ticker.C:
					// if limit reached or no more queue and pending tasks
					if batcher.IsReadyToPersist() || (len(p.writeChan) == 0 && batcher.Len() > 0) {
						wlog.Tracef("flushing batcher")
						err := batcher.PersistBatch()
						if err != nil {
						}
					}

					if p.stop && len(p.writeChan) == 0 {
						break loop
					}
				}
			}
		}(i)
	}

}

func (p *PostgresDBService) Persist(w Model) {
	p.writeChan <- w
}

type Model interface { // simply to enforce a Model interface
	// For now we simply support insert operations
	Type() spec.ModelType // whether insert is activated for this model
}
