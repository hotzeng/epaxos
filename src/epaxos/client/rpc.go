package main

import (
	"epaxos/common"
	"fmt"
	"log"
	"net"
	"reflect"
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
		conn.SetWriteBuffer(configEPaxos.WBuffer)
		conn.SetReadBuffer(configEPaxos.RBuffer)
		id := common.ReplicaID(i)
		go ep.writeUdp(id, conn, ch)
		go ep.readUdp(conn, ep.inbound[i])
	}
	return nil
}

func (ep *EPaxosCluster) writeUdp(id common.ReplicaID, conn *net.UDPConn, ch chan interface{}) error {
	for {
		msg := <-ch
		if configEPaxos.Verbose {
			t := reflect.TypeOf(msg)
			log.Printf("--> #%02d  %s:%+v", id, t, msg)
		}
		buf, err := common.Pack(msg)
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
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		msg, err := common.Unpack(buf, n)
		if err != nil {
			log.Println(err)
			continue
		}
		if configEPaxos.Verbose {
			t := reflect.TypeOf(msg)
			log.Printf("<-- %s  %s:%+v", addr, t, msg)
		}
		ch <- msg
	}
}
