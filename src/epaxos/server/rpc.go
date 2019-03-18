package main

import (
	"bytes"
	"epaxos/common"
	"fmt"
	"github.com/lunixbochs/struc"
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

func (ep *EPaxos) writeUdp(endpoint string, ch chan interface{}) error {
	log.Printf("EPaxos.writeUdp on %s\n", endpoint)
	addr, err := net.ResolveUDPAddr("udp", endpoint)
	if err != nil {
		return err
	}
	for {
		var buf bytes.Buffer
		var err error
		msg := <-ch
		switch msg.(type) {
		case common.PreAcceptMsg:
			buf.WriteByte(0x01)
			m := msg.(common.PreAcceptMsg)
			err = struc.Pack(&buf, &m)
		case common.PreAcceptOKMsg:
			buf.WriteByte(0x11)
			m := msg.(common.PreAcceptOKMsg)
			err = struc.Pack(&buf, &m)
		case common.AcceptMsg:
			buf.WriteByte(0x02)
			m := msg.(common.AcceptMsg)
			err = struc.Pack(&buf, &m)
		case common.AcceptOKMsg:
			buf.WriteByte(0x12)
			m := msg.(common.AcceptOKMsg)
			err = struc.Pack(&buf, &m)
		case common.CommitMsg:
			buf.WriteByte(0x03)
			m := msg.(common.CommitMsg)
			err = struc.Pack(&buf, &m)
		case common.PrepareMsg:
			buf.WriteByte(0x04)
			m := msg.(common.PrepareMsg)
			err = struc.Pack(&buf, &m)
		case common.PrepareOKMsg:
			buf.WriteByte(0x14)
			m := msg.(common.PrepareOKMsg)
			err = struc.Pack(&buf, &m)
		case common.TryPreAcceptMsg:
			buf.WriteByte(0x05)
			m := msg.(common.TryPreAcceptMsg)
			err = struc.Pack(&buf, &m)
		case common.TryPreAcceptOKMsg:
			buf.WriteByte(0x15)
			m := msg.(common.TryPreAcceptOKMsg)
			err = struc.Pack(&buf, &m)
		case common.ProbeMsg:
			buf.WriteByte(0xff)
			m := msg.(common.ProbeMsg)
			err = struc.Pack(&buf, &m)
		default:
			log.Println("Unknown msg type")
			continue
		}
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
		n, _, err := ep.udp.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		if n == 0 {
			log.Println("Ill-formed packet")
			continue
		}
		log.Print("Got raw message:")
		log.Print(buf[0:n])
		r := bytes.NewReader(buf[1:n])
		var msg interface{}
		switch buf[0] {
		case 0x01:
			var m common.PreAcceptMsg
			err = struc.Unpack(r, &m)
			msg = m
		case 0x11:
			var m common.PreAcceptOKMsg
			err = struc.Unpack(r, &m)
			msg = m
		case 0x02:
			var m common.AcceptMsg
			err = struc.Unpack(r, &m)
			msg = m
		case 0x12:
			var m common.AcceptOKMsg
			err = struc.Unpack(r, &m)
			msg = m
		case 0x03:
			var m common.CommitMsg
			err = struc.Unpack(r, &m)
			msg = m
		case 0x04:
			var m common.PrepareMsg
			err = struc.Unpack(r, &m)
			msg = m
		case 0x14:
			var m common.PrepareOKMsg
			err = struc.Unpack(r, &m)
			msg = m
		case 0x05:
			var m common.TryPreAcceptMsg
			err = struc.Unpack(r, &m)
			msg = m
		case 0x15:
			var m common.TryPreAcceptOKMsg
			err = struc.Unpack(r, &m)
			msg = m
		case 0xff:
			var m common.ProbeMsg
			err = struc.Unpack(r, &m)
			if err != nil {
				log.Println(err)
				continue
			}
			ep.recvProbe(&m)
			continue
		default:
			log.Println("Ill-formed packet number")
			continue
		}
		if err != nil {
			log.Println(err)
			continue
		}
		*ep.inbound <- msg
	}
}
