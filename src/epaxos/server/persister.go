package main

import "io"
import "github.com/lunixbochs/struc"
import "epaxos/common"

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
	cmd := &common.Command{}
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
		if cmd.Cmd == common.CmdPut {
			ep.data[cmd.Key] = cmd.Value
		}
	}
	return nil
}
