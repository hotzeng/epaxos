package common

import (
	"bytes"
	"errors"
	"github.com/lunixbochs/struc"
)

func Pack(msg interface{}) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	var err error
	switch m := msg.(type) {
	case PreAcceptMsg:
		buf.WriteByte(0x01)
		err = struc.Pack(&buf, &m)
	case PreAcceptOKMsg:
		buf.WriteByte(0x11)
		err = struc.Pack(&buf, &m)
	case AcceptMsg:
		buf.WriteByte(0x02)
		err = struc.Pack(&buf, &m)
	case AcceptOKMsg:
		buf.WriteByte(0x12)
		err = struc.Pack(&buf, &m)
	case CommitMsg:
		buf.WriteByte(0x03)
		err = struc.Pack(&buf, &m)
	case PrepareMsg:
		buf.WriteByte(0x04)
		err = struc.Pack(&buf, &m)
	case PrepareOKMsg:
		buf.WriteByte(0x14)
		err = struc.Pack(&buf, &m)
	case TryPreAcceptMsg:
		buf.WriteByte(0x05)
		err = struc.Pack(&buf, &m)
	case RequestMsg:
		buf.WriteByte(0x81)
		err = struc.Pack(&buf, &m)
	case RequestOKMsg:
		buf.WriteByte(0x91)
		err = struc.Pack(&buf, &m)
	case RequestAndReadMsg:
		buf.WriteByte(0x82)
		err = struc.Pack(&buf, &m)
	case RequestAndReadOKMsg:
		buf.WriteByte(0x92)
		err = struc.Pack(&buf, &m)
	case KeepMsg:
		buf.WriteByte(0xfe)
		err = struc.Pack(&buf, &m)
	case ProbeMsg:
		buf.WriteByte(0xff)
		err = struc.Pack(&buf, &m)
	default:
		return nil, errors.New("Unknown msg type")
	}
	return &buf, err
}

func Unpack(buf []byte, n int) (interface{}, error) {
	if n == 0 {
		return nil, errors.New("Ill-formed packet")
	}
	r := bytes.NewReader(buf[1:n])
	switch buf[0] {
	case 0x01:
		var m PreAcceptMsg
		err := struc.Unpack(r, &m)
		return m, err
	case 0x11:
		var m PreAcceptOKMsg
		err := struc.Unpack(r, &m)
		return m, err
	case 0x02:
		var m AcceptMsg
		err := struc.Unpack(r, &m)
		return m, err
	case 0x12:
		var m AcceptOKMsg
		err := struc.Unpack(r, &m)
		return m, err
	case 0x03:
		var m CommitMsg
		err := struc.Unpack(r, &m)
		return m, err
	case 0x04:
		var m PrepareMsg
		err := struc.Unpack(r, &m)
		return m, err
	case 0x14:
		var m PrepareOKMsg
		err := struc.Unpack(r, &m)
		return m, err
	case 0x05:
		var m TryPreAcceptMsg
		err := struc.Unpack(r, &m)
		return m, err
	case 0x15:
		var m TryPreAcceptOKMsg
		err := struc.Unpack(r, &m)
		return m, err
	case 0x81:
		var m RequestMsg
		err := struc.Unpack(r, &m)
		return m, err
	case 0x91:
		var m RequestOKMsg
		err := struc.Unpack(r, &m)
		return m, err
	case 0x82:
		var m RequestAndReadMsg
		err := struc.Unpack(r, &m)
		return m, err
	case 0x92:
		var m RequestAndReadOKMsg
		err := struc.Unpack(r, &m)
		return m, err
	case 0xfe:
		var m KeepMsg
		err := struc.Unpack(r, &m)
		return m, err
	case 0xff:
		var m ProbeMsg
		err := struc.Unpack(r, &m)
		return m, err
	default:
		return nil, errors.New("Ill-formed packet number")
	}
}
