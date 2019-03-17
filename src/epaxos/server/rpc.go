package main

import "log"
import "bytes"
import "net"
import "github.com/lunixbochs/struc"
import "epaxos/common"

func (ep *EPaxos) makeMulticast(msg interface{}, nrep int64) error {
	for i := int64(0); i < nrep; i++ {
		if i == int64(ep.self) {
			ep.inbound <- msg
		} else {
			err := struc.Pack(ep.rpc[i], msg)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (ep *EPaxos) listenUdp(endpoint string) error {
	addr, err := net.ResolveUDPAddr("udp", endpoint)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	buf := make([]byte, 65535)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		if n == 0 {
			log.Println("Ill-formed packet")
			continue
		}
		r := bytes.NewReader(buf[1:n])
		var msg interface{}
		switch buf[0] {
		case 0x01:
			var m common.PreAcceptMsg
			err = struc.Unpack(r, &m)
			msg = m
		default:
			log.Println("Ill-formed packet number")
			continue
		}
		if err != nil {
			log.Println(err)
			continue
		}
		ep.inbound <- msg
	}
}
