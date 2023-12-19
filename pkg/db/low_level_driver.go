package db

import (
	"context"
	"strings"
	"time"

	"github.com/ClickHouse/ch-go"
)

// Connect to the PostgreSQL Database and get the multithread-proof connection
// from the given url-composed credentials
func (s *DBService) ConnectLowLevel() error {
	ctx := context.Background()

	opts := ParseChUrlIntoOptionsLowLevel(s.connectionUrl)
	lowLevelConn, err := ch.Dial(ctx, opts)
	if err == nil {
		s.lowLevelClient = lowLevelConn
		s.makeMigrations()
	}

	return err

}

func ParseChUrlIntoOptionsLowLevel(url string) ch.Options {
	var user string
	var password string
	var database string

	protocolAndDetails := strings.Split(url, "://")
	// protocol := protocolAndDetails[0]
	details := protocolAndDetails[1]

	credentialsAndEndpoint := strings.Split(details, "@")
	credentials := credentialsAndEndpoint[0]
	endpoint := credentialsAndEndpoint[1]

	hostPortAndPathParams := strings.Split(endpoint, "/")
	fqdn := hostPortAndPathParams[0]
	pathParams := hostPortAndPathParams[1]

	pathAndParams := strings.Split(pathParams, "?")
	database = pathAndParams[0]
	// params := pathAndParams[1]

	user = strings.Split(credentials, ":")[0]
	password = strings.Split(credentials, ":")[1]

	return ch.Options{
		Address:  fqdn,
		Database: database,
		User:     user,
		Password: password}
}

func (p *DBService) Persist(obj PersistObject) error {

	query := obj.Query()
	input := obj.Input()
	table := obj.Table()
	startTime := time.Now()

	p.lowMu.Lock()
	err := p.lowLevelClient.Do(p.ctx, ch.Query{
		Body:  query,
		Input: input,
	})
	p.lowMu.Unlock()
	elapsedTime := time.Since(startTime)

	if err == nil {
		log.Infof("table %s persisted %d rows in %fs", table, obj.Rows(), elapsedTime.Seconds())

		if _, ok := p.monitorMetrics[table]; !ok {
			p.monitorMetrics[table] = DBMonitorMetrics{}
		}

		metrics := p.monitorMetrics[table]
		metrics.UpdateValues(obj.Rows(), elapsedTime)
		p.monitorMetrics[table] = metrics
	}

	return err
}
