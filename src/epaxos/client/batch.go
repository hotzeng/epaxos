package main

import (
	"context"
	"epaxos/common"
	"errors"
	"github.com/docopt/docopt-go"
	"golang.org/x/sync/semaphore"
	"log"
	"math/rand"
	"sync"
	"time"
)

func (ep *EPaxosCluster) cmdBatchPut(argv []string) error {
	usage := `usage: client batch-put [options] <server> <keys>
options:
	-h, --help
	-v, --verbose        be verbose
	-N <nops>            number of operations, infinite = -1 [default: -1]
	-S <servers>         servers to choose from [default: 1]
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

	var mu sync.Mutex
	dic := make(map[int64]chan common.RequestOKMsg)
	chx := make(chan interface{})
	for _, ch := range ep.inbound {
		go func(ch chan interface{}) {
			for {
				select {
				case m := <-ch:
					chx <- m
				}
			}
		}(ch)
	}
	go func() {
		for {
			select {
			case msg := <-chx:
				if m, ok := msg.(common.RequestOKMsg); ok {
					func() {
						mu.Lock()
						defer mu.Unlock()
						if ch, ok := dic[m.MId]; ok {
							ch <- m
						} else {
							log.Printf("Received erroneous RequestOKMsg")
						}
					}()
				}
			}
		}
	}()

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

		sid := int64(server+serverL) % configEPaxos.NReps
		kid := key + keyL

		rawmsg := common.RequestMsg{
			MId: mid,
			Cmd: common.Command{
				Cmd:   common.CmdPut,
				Key:   common.Key(kid),
				Value: common.Value(mid),
			},
		}
		ch := make(chan common.RequestOKMsg)
		func() {
			mu.Lock()
			defer mu.Unlock()
			dic[mid] = ch
		}()
		ep.rpc[sid] <- rawmsg
		if verbose {
			log.Printf("Batch PUT: ##%05d [%d]=0x%016x sent to %d", count, kid, mid, sid)
		}

		wg.Add(1)
		go func(count, kid int, mid, sid int64) {
			defer wg.Done()
			defer func() {
				mu.Lock()
				defer mu.Unlock()
				delete(dic, mid)
			}()
			defer sem.Release(1)
			for ret := 0; ret <= retry; ret++ {
				select {
				case msg := <-ch:
					if msg.Err {
						good = false
						log.Printf("Batch PUT: ##%05d [%d]=0x%016x to %d remote error", count, kid, mid, sid)
					} else {
						if verbose {
							log.Printf("Batch PUT: ##%05d [%d]=0x%016x to %d committed", count, kid, mid, sid)
						}
					}
					return
				case <-time.After(configEPaxos.TimeOut):
				}
				if ret < retry {
					log.Printf("Batch PUT: ##%05d [%d]=0x%016x to %d timeout, retry %d/%d", count, kid, mid, sid, ret, retry)
				} else {
					log.Printf("Batch PUT: ##%05d [%d]=0x%016x to %d timeout, no more retry", count, kid, mid, sid)
				}
			}
			good = false
		}(count, kid, mid, sid)

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
	if !good {
		return errors.New("remote errors")
	}
	return nil
}
