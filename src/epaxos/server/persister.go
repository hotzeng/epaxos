package main

import (
	"epaxos/common"
	"github.com/lunixbochs/struc"
	"io"
)

func (ep *EPaxos) appendLog(rep common.ReplicaID, cmd *common.Command) error {
	lst := ep.array[rep]
	file := lst.LogFile
	lst.Mu.Lock()
	defer lst.Mu.Unlock()
	err := struc.Pack(file, cmd)
	if err != nil {
		return err
	}
	err = file.Sync()
	if err != nil {
		return err
	}
	return nil
}

func (ep *EPaxos) recoverFromLog() error {
	// TODO: seems totally wrong
	lst := ep.array[ep.self]
	file := lst.LogFile
	lst.Mu.Lock()
	defer lst.Mu.Unlock()
	_, err := file.Seek(0, 0)
	if err != nil {
		return err
	}
	ep.lastInst = 0
	ep.data = make(map[common.Key]common.Value)
	var cmd common.Command
	for {
		err = struc.Unpack(file, &cmd)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		lst.Offset++
		ep.lastInst++
		if cmd.CmdType == common.CmdPut {
			ep.data[cmd.Key] = cmd.Value
		}
	}
	return nil
}
