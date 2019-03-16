package main

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
	// TODO
	return nil
}
