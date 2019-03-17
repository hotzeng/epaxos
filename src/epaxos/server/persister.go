package main

import "io"
import "github.com/lunixbochs/struc"
import "epaxos/common"

func AppendLog(ep *common.EPaxos, rep common.ReplicaID, cmd *common.Command) error {
	var err error
	lst := ep.Array[rep]
	file := lst.LogFile
	lst.Mu.Lock()
	defer lst.Mu.Unlock()
	if err = struc.Pack(file, cmd); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	return nil
}

func RecoverFromLog(ep *common.EPaxos) error {
	var err error
	lst := ep.Array[ep.Self]
	file := lst.LogFile
	lst.Mu.Lock()
	defer lst.Mu.Unlock()
	if _, err = file.Seek(0, 0); err != nil {
		return err
	}
	cmd := &common.Command{}
	for {
		if err = struc.Unpack(file, &cmd); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if cmd.Cmd == common.CmdPut {
			ep.Data[cmd.Key] = cmd.Value
		}
	}
	return nil
}
