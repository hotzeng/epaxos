package main

import (
	"epaxos/common"
	"errors"
	"fmt"
	"log"
	"net/rpc"
	"sync"
)

func probeAll(verbose bool) error {
	if verbose {
		log.Printf("Start probeAll for %d", configEPaxos.NReps)
	}
	var wg sync.WaitGroup
	wg.Add(int(configEPaxos.NReps))
	good := true
	for i := int64(0); i < configEPaxos.NReps; i++ {
		id := common.ReplicaID(i)
		go func() {
			defer wg.Done()
			err := probeOne(verbose, id)
			if err != nil {
				log.Println(err)
				good = false
			}
		}()
	}
	wg.Wait()
	if verbose {
		log.Print("Done probeAll")
	}
	return nil
}

func probeOne(verbose bool, id common.ReplicaID) error {
	endpoint := fmt.Sprintf(configEPaxos.ServerFmt, id)
	if verbose {
		log.Printf("Start probeOne %s", endpoint)
	}

	client, err := rpc.Dial("tcp", endpoint)
	if err != nil {
		return err
	}

	var reply string
	err = client.Call("EPaxos.ReadyProbe", "hello", &reply)
	if err != nil {
		return err
	}
	if verbose {
		log.Println(reply)
	}

	var wg sync.WaitGroup
	wg.Add(int(configEPaxos.NReps))
	good := true
	for i := int64(0); i < configEPaxos.NReps; i++ {
		id2 := common.ReplicaID(i)
		go func() {
			defer wg.Done()
			err := probeOneOne(verbose, client, id, id2)
			if err != nil {
				log.Println(err)
				good = false
			}
		}()
	}
	if !good {
		return errors.New("Various error happend")
	}
	wg.Wait()
	if verbose {
		log.Printf("Done probeOne %s", endpoint)
	}
	return nil
}

func probeOneOne(verbose bool, client *rpc.Client, id1, id2 common.ReplicaID) error {
	var reply string
	err := client.Call("EPaxos.SendProbe", id2, &reply)
	if err != nil {
		return err
	}
	if verbose {
		log.Println(reply)
	}
	return nil
}
