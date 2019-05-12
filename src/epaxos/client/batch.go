package main

import (
	"context"
	"epaxos/common"
	"errors"
	"fmt"
	"github.com/docopt/docopt-go"
	"golang.org/x/sync/semaphore"
	"log"
	"math/rand"
	"reflect"
	"sync"
	"time"
)

type PutStateMachine struct {
	mu     sync.RWMutex // Limit access to ch
	count  int
	mid    int64
	server common.ReplicaID
	key    common.Key
	val    common.Value
	ch     chan common.RequestOKMsg
	retry  int
}

func (ism *PutStateMachine) chMake() {
	ism.mu.Lock()
	defer ism.mu.Unlock()
	ism.ch = make(chan common.RequestOKMsg)
}
func (ism *PutStateMachine) chDrop() {
	go func() {
		for {
			m, ok := <-ism.ch // Consume all pending requests
			if ok {
				log.Printf("Warning: excessive message ##%05d [%d]=0x%016x (%d) to %d: %+v", ism.count, ism.key, ism.val, ism.mid, ism.server, m)
			}
		}
	}()
	ism.mu.Lock() // No more write requests
	defer ism.mu.Unlock()
	close(ism.ch) // tell the fake consumer to stop
	ism.ch = nil
}
func (ism *PutStateMachine) chRead(timeout time.Duration) (*common.RequestOKMsg, bool) {
	ism.mu.RLock()
	defer ism.mu.RUnlock()
	select {
	case msg := <-ism.ch:
		return &msg, true
	case <-time.After(timeout):
		return nil, false
	}
}

func (ep *EPaxosCluster) runISM(ism *PutStateMachine, verbose bool) (time.Duration, bool) {
	rawmsg := common.RequestMsg{
		MId: ism.mid,
		Cmd: common.Command{
			CmdType: common.CmdPut,
			Key:     ism.key,
			Value:   ism.val,
		},
	}
	ism.chMake()
	defer ism.chDrop()
	start := time.Now()
	ep.rpc[ism.server] <- rawmsg
	if verbose {
		log.Printf("Batch PUT: ##%05d [%d]=0x%016x (%d) sent to %d", ism.count, ism.key, ism.val, ism.mid, ism.server)
	}
	good := true
	for ret := 0; ret <= ism.retry; ret++ {
		if msg, ok := ism.chRead(configEPaxos.TimeOut); ok {
			if msg.Err {
				good = false
				log.Printf("Batch PUT: ##%05d [%d]=0x%016x to %d remote error", ism.count, ism.key, ism.val, ism.server)
			} else {
				if verbose {
					log.Printf("Batch PUT: ##%05d [%d]=0x%016x to %d committed", ism.count, ism.key, ism.val, ism.server)
				}
			}
			return time.Now().Sub(start), good
		}
		if ret < ism.retry {
			log.Printf("Batch PUT: ##%05d [%d]=0x%016x to %d timeout, retry %d/%d", ism.count, ism.key, ism.val, ism.server, ret, ism.retry)
			ep.rpc[ism.server] <- rawmsg
		} else {
			log.Printf("Batch PUT: ##%05d [%d]=0x%016x to %d timeout, no more retry", ism.count, ism.key, ism.val, ism.server)
		}
	}
	return time.Now().Sub(start), false
}

func (ep *EPaxosCluster) cmdBatchPut(argv []string) error {
	usage := `usage: client batch-put [options] <server> <keys>
options:
	-h, --help
	-v, --verbose        be verbose
	-N <nops>            number of operations, infinite = -1 [default: -1]
	-S <servers>         servers to choose from [default: 1]
	-L, --latency        write individual latency to stdout
	--key-bias <key>     minimum key value [default: 0]
	--random-server      choose servers randomly each time
	--random-key         choose keys randomly each time
	--max-retry <retry>  retry limit for single operation [default: 65536]
	--pipeline <pipe>    on-flight operations limit [default: 1]
	`
	args, _ := docopt.ParseArgs(usage, argv, "epaxos-client version "+VERSION)

	verbose, err := args.Bool("--verbose")
	if err != nil {
		return err
	}

	nops, err := args.Int("-N")
	if err != nil {
		return err
	}

	serverL, err := args.Int("<server>")
	if err != nil {
		return err
	}

	serverN, err := args.Int("-S")
	if err != nil {
		return err
	}

	latency, err := args.Bool("--latency")
	if err != nil {
		return err
	}

	keyL, err := args.Int("--key-bias")
	if err != nil {
		return err
	}

	keyN, err := args.Int("<keys>")
	if err != nil {
		return err
	}

	serverR, err := args.Bool("--random-server")
	if err != nil {
		return err
	}

	keyR, err := args.Bool("--random-key")
	if err != nil {
		return err
	}

	retry, err := args.Int("--max-retry")
	if err != nil {
		return err
	}

	pipe, err := args.Int("--pipeline")
	if err != nil {
		return err
	}

	mid := rand.Int63()
	sem := semaphore.NewWeighted(int64(pipe))
	ctx := context.TODO()

	var mu sync.RWMutex
	dic := make(map[int64]*PutStateMachine)
	chx := make(chan interface{}, pipe)
	lat := make(chan int64, 1024)
	latDone := make(chan bool)
	for i, ch := range ep.inbound {
		go func(i int, ch chan interface{}) {
			for {
				m := <-ch
				if verbose {
					log.Printf("<-- #%02d  %s:%+v", i, reflect.TypeOf(m), m)
				}
				chx <- m
			}
		}(i, ch)
	}
	go func() {
		for {
			msg := <-chx
			if m, ok := msg.(common.RequestOKMsg); ok {
				ism := func() *PutStateMachine {
					mu.RLock()
					defer mu.RUnlock()
					if ism, ok := dic[m.MId]; ok {
						return ism
					} else {
						log.Printf("Received erroneous RequestOKMsg: %+v", m)
						return nil
					}
				}()
				if ism != nil {
					func() {
						ism.mu.RLock()
						defer ism.mu.RUnlock()
						if ism.ch != nil {
							ism.ch <- m
						} else {
							log.Printf("Warning: attempt to write to nil chan at %d", m.MId)
						}
					}()
				}
			}
		}
	}()
	if latency {
		go func() {
			for {
				if l, ok := <-lat; ok {
					fmt.Printf("%d\n", l)
				} else {
					break
				}
			}
			latDone <- true
		}()
	}

	var wg sync.WaitGroup
	good := true
	count := 0
	server := 0
	key := 0
	for nops == -1 || nops > 0 {
		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}

		if serverR {
			server = rand.Intn(serverN)
		}
		if keyR {
			key = rand.Intn(keyN)
		}

		ism := &PutStateMachine{
			count:  count,
			mid:    mid,
			server: common.ReplicaID(int64(server+serverL) % configEPaxos.NReps),
			key:    common.Key(key + keyL),
			val:    common.Value(mid),
			retry:  retry,
		}
		func() {
			mu.Lock()
			defer mu.Unlock()
			dic[mid] = ism
		}()
		wg.Add(1)

		go func(mid int64) {
			defer wg.Done()
			defer func(mid int64) {
				mu.Lock()
				defer mu.Unlock()
				delete(dic, mid)
			}(mid)
			defer sem.Release(1)
			dur, ok := ep.runISM(ism, verbose)
			if !ok {
				good = false
			}
			if latency {
				lat <- dur.Nanoseconds()
			}
		}(mid)

		mid++
		count++

		if !serverR {
			server = (server + 1) % serverN
		}
		if !keyR {
			key = (key + 1) % keyN
		}
		if nops > 0 {
			nops--
		}
	}
	wg.Wait()
	if latency {
		close(lat)
		<-latDone
	}
	if !good {
		return errors.New("remote errors")
	}
	return nil
}
