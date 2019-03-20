package main

import (
	"epaxos/common"
	"github.com/lunixbochs/struc"
	"log"
)

func (ep *EPaxos) mustAppendLogs(rep common.ReplicaID) {
	err := ep.appendLogs(rep)
	if err != nil {
		log.Fatalf("Can't append logs: %v", err)
	}
}

func (ep *EPaxos) appendLogs(rep common.ReplicaID) error {
	my := ep.array[rep]
	my.Mu.Lock()
	defer my.Mu.Unlock()
	self := rep == ep.self

	if ep.verbose {
		log.Printf("Start appendLogs on %d: offset %v, pending %v", rep, my.Offset, len(my.Pending))
	}

	file := my.LogFile
	_, err := file.Seek(0, 2)
	if err != nil {
		return err
	}

	var i int
	for i = 0; i < len(my.Pending); i++ {
		id := common.InstanceID(int(my.Offset) + i)
		obj := my.Pending[i]
		if obj == nil {
			break
		}
		if obj.state != Finished {
			break
		}
		if ep.verbose {
			log.Printf("Persisting appendLogs on #%02d.%d: %+v", rep, id, obj.inst)
		}
		err = struc.Pack(file, obj.inst.Cmd)
		if err != nil {
			return err
		}
		if self {
			ism := func() *InstStateMachine {
				ep.ismsL.RLock()
				defer ep.ismsL.RUnlock()
				return ep.isms[id]
			}()
			if ep.verbose {
				log.Printf("Notifying appendLogs on #%02d.%d", rep, id)
			}
			ism.cmChan <- true
		}
	}

	if i == 0 {
		if ep.verbose {
			log.Printf("Done appendLogs on %d, untouched offset %d, still %d pending", rep, my.Offset, len(my.Pending))
		}
	} else {
		my.Offset = common.InstanceID(int(my.Offset) + i)
		my.Pending = my.Pending[i:]
		if ep.verbose {
			log.Printf("Done appendLogs on %d, %d persisted and new offset is %d, still %d pending", rep, i, my.Offset, len(my.Pending))
		}
	}
	return nil
}

func (ep *EPaxos) retrieveLog(ref common.InstRef) (common.Command, error) {
	var cmd common.Command
	my := ep.array[ref.Replica]
	my.Mu.Lock()
	defer my.Mu.Unlock()

	file := my.LogFile
	_, err := file.Seek(int64(ref.Inst)*common.COMMAND_SIZE, 0)
	if err != nil {
		return cmd, err
	}
	err = struc.Unpack(file, &cmd)
	if err != nil {
		return cmd, err
	}
	return cmd, nil
}
