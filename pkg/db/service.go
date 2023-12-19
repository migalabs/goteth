package db

import (
	"context"

	"sync"
	"time"

	"github.com/ClickHouse/ch-go"
	"github.com/ClickHouse/ch-go/proto"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/sirupsen/logrus"
)

// Static postgres queries, for each modification in the tables, the table needs to be reseted
var (
	// wlogrus associated with the postgres db
	modName  = "db"
	PsqlType = "clickhouse-db"
	log      = logrus.WithField(
		"module", modName,
	)
	MAX_BATCH_QUEUE       = 1000
	MAX_EPOCH_BATCH_QUEUE = 1
)

type DBServiceOption func(*DBService) error

type DBService struct {
	// Control Variables
	ctx           context.Context
	connectionUrl string // the url might not be necessary (better to remove it?¿)

	lowLevelClient  *ch.Client  // for bulk loads, mainly insert
	highLevelClient driver.Conn // for side tasks, like Select and Delete

	monitorMetrics map[string]DBMonitorMetrics // map table and metrics
	lowMu          sync.Mutex
	highMu         sync.Mutex
}

func New(ctx context.Context, url string, options ...DBServiceOption) (*DBService, error) {
	var err error
	pService := &DBService{
		ctx:            ctx,
		connectionUrl:  url,
		monitorMetrics: make(map[string]DBMonitorMetrics),
	}

	for _, o := range options {
		err := o(pService)
		if err != nil {
			return pService, err
		}
	}

	return pService, err
}

// Connect to the PostgreSQL Database and get the multithread-proof connection
// from the given url-composed credentials
func (s *DBService) Connect() error {
	err := s.ConnectLowLevel()
	if err != nil {
		return err
	}

	err = s.ConnectHighLevel()
	if err != nil {
		return err
	}
	return nil

}

func WithUrl(url string) DBServiceOption {
	return func(s *DBService) error {
		s.connectionUrl = url
		return nil
	}
}

func (p *DBService) Finish() {

	p.lowLevelClient.Close()
	p.highLevelClient.Close()
	log.Infof("Routines finished...")
	log.Infof("closing connection to database server...")
	log.Infof("connection to database server closed...")
}

type DBMonitorMetrics struct {
	Rows        int           // how many rows were persisted in the last copy
	PersistTime time.Duration // how much time to persist the last copy
	RowRate     float64       // rows per second transmitted
}

func (d *DBMonitorMetrics) UpdateValues(rows int, time time.Duration) {
	d.Rows = rows
	d.PersistTime = time

	d.RowRate = float64(rows) / time.Seconds()
}

type PersistObject interface {
	Table() string
	Query() string
	Input() proto.Input
	Columns() int
	Rows() int
}

type DeleteObject interface {
	Table() string
	Query() string
	Args() []any
}
