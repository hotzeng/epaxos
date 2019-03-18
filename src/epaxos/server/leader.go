package main

import (
	"epaxos/common"
)

func (ep *EPaxos) ProcessRequest(req common.RequestMsg) (common.RequestOKMsg, error) {
	// TODO
	return common.RequestOKMsg{}, nil
}

func (ep *EPaxos) ProcessRequestAndRead(req common.RequestAndReadMsg) (common.RequestAndReadOKMsg, error) {
	// TODO
	return common.RequestAndReadOKMsg{}, nil
}
