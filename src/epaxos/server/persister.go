package main

import (
	"epaxos/common"
	"github.com/lunixbochs/struc"
	"log"
)

func (ep *EPaxos) appendLogs(rep common.ReplicaID) error {
	my := ep.array[rep]
	my.Mu.Lock()
	defer my.Mu.Unlock()

	file := my.LogFile
	_, err := file.Seek(0, 2)
	if err != nil {
		return err
	}

	var i int
	for i = 0; i < len(my.Pending); i++ {
		obj := my.Pending[i]
		if obj == nil {
			break
		}
		if obj.state != Finished {
			break
		}
		if ep.verbose {
			log.Printf("Persisting %d %d->%d: %v", rep, my.Offset, int(my.Offset)+i, obj.inst)
		}
		err = struc.Pack(file, obj.inst.Cmd)
		if err != nil {
			return err
		}
	}

	my.Offset = common.InstanceID(int(my.Offset) + i)
	my.Pending = my.Pending[i:]
	if ep.verbose {
		log.Printf("Done appendLogs %d, new offset %d", rep, my.Offset)
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
