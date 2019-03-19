package main

import (
	"epaxos/common"
	"fmt"
	"log"
	"net"
	"reflect"
	"strconv"
)

func (ep *EPaxos) makeMulticast(msg interface{}, nrep int64) []common.ReplicaID {
	var res []common.ReplicaID
    if ep.verbose == false {
        fmt.Printf("enter makeMulticast\n")
    }
	for i := int64(0); i < nrep; i++ {
        if i == int64(ep.self) {
            nrep++
            continue
        }
		res = append(res, common.ReplicaID(i))
		ep.rpc[i] <- msg
        if ep.verbose == false {
            fmt.Printf("msg send to ep.rpc[%d]\n", i)
        }
	}
	return res
}

func (ep *EPaxos) forkUdp() error {
	bias, err := strconv.ParseInt(common.GetEnv("EPAXOS_SERVERS_FMT_BIAS", "1"), 10, 64)
	if err != nil {
		return err
	}
	tmp := common.GetEnv("EPAXOS_SERVERS_FMT", "127.0.0.%d:23333")
	for i, ch := range ep.rpc {
		if common.ReplicaID(i) != ep.self {
			id := common.ReplicaID(i)
			go ep.writeUdp(id, fmt.Sprintf(tmp, int64(i)+bias), ch)
		}
	}
	return ep.readUdp()
}

func (ep *EPaxos) replyClient(addr *net.UDPAddr, msg interface{}) error {
	if ep.verbose {
		t := reflect.TypeOf(msg)
		if m, ok := msg.(common.ClientMsg); ok {
			log.Printf("--> 0x%016x  %s:%+v", m.GetSender(), t, msg)
		} else {
			log.Printf("--> ???%s  %s:%+v", addr, t, msg)
		}
	}
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

func (ep *EPaxos) writeUdp(id common.ReplicaID, endpoint string, ch chan interface{}) error {
	log.Printf("EPaxos.writeUdp on %s\n", endpoint)
	addr, err := net.ResolveUDPAddr("udp", endpoint)
	if err != nil {
		return err
	}
	for {
		msg := <-ch
		if ep.verbose {
			t := reflect.TypeOf(msg)
			log.Printf("--> #%02d  %s:%+v", id, t, msg)
		}
		buf, err := common.Pack(msg)
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
		if ep.verbose {
			t := reflect.TypeOf(msg)
			if m, ok := msg.(common.ServerMsg); ok {
				log.Printf("<-- #%02d  %s:%+v", m.GetSender(), t, msg)
			} else if m, ok := msg.(common.ClientMsg); ok {
				log.Printf("<-- 0x%016x  %s:%+v", m.GetSender(), t, msg)
			} else {
				log.Printf("<-- ???%s  %s:%+v", addr, t, msg)
			}
		}
		switch m := msg.(type) {
		case common.KeepMsg:
			go func() {
				log.Printf("Probe: Got KeepMsg %d, will reply", m.MId)
				err := ep.replyClient(addr, m)
				if err != nil {
					log.Println(err)
				}
			}()
			continue
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
			continue
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
			continue
		case common.ProbeReqMsg:
			go func() {
				err := ep.sendProbe(m.Replica)
				if err != nil {
					log.Println(err)
					err = ep.replyClient(addr, common.ProbeReqMsg{
						MId:     m.MId,
						Replica: common.ReplicaID(-1),
					})
				} else {
					err = ep.replyClient(addr, m)
				}
				if err != nil {
					log.Println(err)
				}
			}()
			continue
		case common.ProbeMsg:
			go ep.recvProbe(&m)
			continue
		}
		*ep.inbound <- msg
	}
}
