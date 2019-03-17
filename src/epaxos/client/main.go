package main

import "log"
import "bytes"
import "net"
import "epaxos/common"
import "github.com/lunixbochs/struc"

func main() {
	localAddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:26666")
	if err != nil {
		log.Fatal(err)
	}
	remoteAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:23333")
	if err != nil {
		log.Fatal(err)
	}
	conn, err := net.DialUDP("udp", localAddr, remoteAddr)
	if err != nil {
		log.Fatal(err)
	}
	var buf bytes.Buffer
	buf.WriteByte(0xca)
	msg := &common.PreAcceptMsg{
		Id: common.InstRef{
			Replica: 233,
			Inst:    666,
		},
		Inst: common.Instance{
			Deps: make([]common.InstRef, 0),
		},
	}
	err = struc.Pack(&buf, msg)
	if err != nil {
		log.Fatal(err)
	}
	_, err = conn.Write(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	log.Print("good")
}

// import "os"
// import "log"
// import "bufio"
// import "net/rpc"
// import "bytes"
// import "epaxos/common"

// func main() {
// 	endpoint := common.GetEnv("EPAXOS_SERVER", "localhost:23333")

// 	client, err := rpc.Dial("tcp", endpoint)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	in := bufio.NewReader(os.Stdin)
// 	for {
// 		line, err := in.ReadString('\n')
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		log.Println(line)
// 		var reply string
// 		err = client.Call("EPaxos.HelloWorld", line, &reply) // TODO
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		log.Println(reply)
// 	}
// }
