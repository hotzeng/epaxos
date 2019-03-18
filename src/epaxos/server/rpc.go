package main

import (
	"epaxos/common"
	"fmt"
	"log"
	"net"
	"strconv"
)

func (ep *EPaxos) makeMulticast(msg interface{}, nrep int64) []common.ReplicaID {
	var res []common.ReplicaID
	for i := int64(0); i < nrep; i++ {
		res = append(res, common.ReplicaID(i))
		ep.rpc[i] <- msg
	}
	return res
}

func (ep *EPaxos) forkUdp() error {
	bias, err := strconv.ParseInt(common.GetEnv("EPAXOS_SERVERS_FMT_BIAS", "1"), 10, 64)
	if err != nil {
		return err
	}
	tmp := common.GetEnv("EPAXOS_SERVERS_FMT", "127.0.0.%d:23333")
	go ep.readUdp()
	for i, ch := range ep.rpc {
		if common.ReplicaID(i) != ep.self {
			go ep.writeUdp(fmt.Sprintf(tmp, int64(i)+bias), ch)
		}
	}
	return nil
}

func (ep *EPaxos) replyClient(addr *net.UDPAddr, msg interface{}) error {
	buf, err := common.Pack(msg)
	if err != nil {
		return err
	}
	_, err = ep.udp.WriteTo(buf.Bytes(), addr)
	if err != nil {
		return err
	}
	return nil
}

func (ep *EPaxos) writeUdp(endpoint string, ch chan interface{}) error {
	log.Printf("EPaxos.writeUdp on %s\n", endpoint)
	addr, err := net.ResolveUDPAddr("udp", endpoint)
	if err != nil {
		return err
	}
	for {
		buf, err := common.Pack(<-ch)
		if err != nil {
			log.Println(err)
			continue
		}
		_, err = ep.udp.WriteTo(buf.Bytes(), addr)
		if err != nil {
			log.Println(err)
			continue
		}
	}
}

func (ep *EPaxos) readUdp() error {
	buf := make([]byte, 65536)
	for {
		n, addr, err := ep.udp.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		msg, err := common.Unpack(buf, n)
		if err != nil {
			log.Println(err)
			continue
		}
		switch m := msg.(type) {
		case common.KeepMsg:
			go func() {
				err := ep.replyClient(addr, m)
				if err != nil {
					log.Println(err)
				}
			}()
		case common.RequestMsg:
			go func() {
				r, err := ep.ProcessRequest(m)
				if err != nil {
					log.Println(err)
					err = ep.replyClient(addr, common.RequestOKMsg{Err: true})
				} else {
					err = ep.replyClient(addr, r)
				}
				if err != nil {
					log.Println(err)
				}
			}()
		case common.RequestAndReadMsg:
			go func() {
				r, err := ep.ProcessRequestAndRead(m)
				if err != nil {
					log.Println(err)
					err = ep.replyClient(addr, common.RequestAndReadOKMsg{Err: true})
				} else {
					err = ep.replyClient(addr, r)
				}
				if err != nil {
					log.Println(err)
				}
			}()
		case common.ProbeMsg:
			go ep.recvProbe(&m)
		}
		*ep.inbound <- msg
	}
}
