package main

import (
	"epaxos/common"
)

func (ep *EPaxos) ProcessRequest(req common.RequestMsg) (common.RequestOKMsg, error) {
	// TODO
	return common.RequestOKMsg{MId: req.MId}, nil
}

func (ep *EPaxos) ProcessRequestAndRead(req common.RequestAndReadMsg) (common.RequestAndReadOKMsg, error) {
	// TODO
	return common.RequestAndReadOKMsg{MId: req.MId}, nil
}
