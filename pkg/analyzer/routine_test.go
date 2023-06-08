package analyzer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cortze/eth-cl-state-analyzer/pkg/clientapi"
	"github.com/magiconair/properties/assert"
)

func TestStateAnalyzerRoutineClosure(t *testing.T) {

	pCtx := context.Background()
	ctx, cancel := context.WithCancel(pCtx)

	// generate the httpAPI client
	cli, err := clientapi.NewAPIClient(ctx, "http://localhost:5052", 50*time.Second)
	if err != nil {
		t.Errorf("could not create beacon node requester: %s", err)
	}

	analyzer, err := NewStateAnalyzer(ctx, cli, 6000000, 6000100, "", 1, 1, "historical", "", false, "epoch,block", true)

	if err != nil {
		t.Errorf("could not create state analyzer: %s", err)
	}
	requestDone := make(chan struct{})
	ticker := time.NewTicker(time.Second * 60)

	go func() {
		analyzer.Run()
		requestDone <- struct{}{}
	}()

	fmt.Printf("waiting for 10 seconds...\n")
	time.Sleep(time.Second * 10)
	go analyzer.Close() // trigger finishing the anayzer run process

	fmt.Printf("waiting for finish signal or cancel...\n")
	select {
	case <-requestDone:
		assert.Equal(t, 1, 1) // the analyzer finished and we receive the signal on the channel
	case <-ticker.C:
		fmt.Printf("cancelling...") // after 60 seconds we have not received any finish signal, trigger context cancel
		cancel()
		<-requestDone
		t.Errorf("routine did not finish in time...")
	}
}
