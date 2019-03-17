package main

import "fmt"
import "log"
import "bytes"
import "strconv"
import "net"
import "github.com/lunixbochs/struc"
import "epaxos/common"

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
		go ep.writeUdp(fmt.Sprintf(tmp, int64(i)+bias), ch)
	}
	return nil
}

func (ep *EPaxos) writeUdp(endpoint string, ch chan interface{}) error {
	log.Printf("EPaxos.writeUdp on %s\n", endpoint)
	addr, err := net.ResolveUDPAddr("udp", endpoint)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	for {
		msg := <-ch
		switch msg.(type) {
		case common.PreAcceptMsg:
			buf.WriteByte(0x01)
		case common.PreAcceptOKMsg:
			buf.WriteByte(0x11)
		case common.AcceptMsg:
			buf.WriteByte(0x02)
		case common.AcceptOKMsg:
			buf.WriteByte(0x12)
		case common.CommitMsg:
			buf.WriteByte(0x03)
		case common.PrepareMsg:
			buf.WriteByte(0x04)
		case common.PrepareOKMsg:
			buf.WriteByte(0x14)
		case common.TryPreAcceptMsg:
			buf.WriteByte(0x05)
		case common.TryPreAcceptOKMsg:
			buf.WriteByte(0x15)
		case common.ProbeMsg:
			buf.WriteByte(0xff)
		default:
			log.Println("Unknown msg type")
			continue
		}
		err := struc.Pack(&buf, msg)
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
	buf := make([]byte, 65535)
	for {
		n, _, err := ep.udp.ReadFromUDP(buf)
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
			log.Printf("Received ProbeMsg from %d\n", m.Replica)
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
