package main

import (
	"epaxos/common"
	"fmt"
	"log"
	"net"
)

func (ep *EPaxosCluster) forkUdp() error {
	for i, ch := range ep.rpc {
		endpoint := fmt.Sprintf(
			configEPaxos.ServerFmt,
			int64(i)+configEPaxos.ServerBias,
		)
		addr, err := net.ResolveUDPAddr("udp", endpoint)
		if err != nil {
			return err
		}
		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			return err
		}
		go ep.writeUdp(conn, ch)
		go ep.readUdp(conn, ep.inbound[i])
	}
	return nil
}

func (ep *EPaxosCluster) writeUdp(conn *net.UDPConn, ch chan interface{}) error {
	for {
		buf, err := common.Pack(<-ch)
		if err != nil {
			log.Println(err)
			continue
		}
		_, err = conn.Write(buf.Bytes())
		if err != nil {
			log.Println(err)
			continue
		}
	}
}

func (ep *EPaxosCluster) readUdp(conn *net.UDPConn, ch chan interface{}) error {
	buf := make([]byte, 65536)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return err
		}
		msg, err := common.Unpack(buf, n)
		if err != nil {
			log.Println(err)
			continue
		}
		ch <- msg
	}
}
