package main

import "os"
import "bufio"
import "net/rpc"
import "log"
import "strings"
import "strconv"
import "epaxos/common"

var VERSION string

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("This is epaxos-server, version %s", VERSION)
	endpoint := common.GetEnv("EPAXOS_SERVER", "localhost:23333")

	client, err := rpc.Dial("tcp", endpoint)
	if err != nil {
		log.Fatal(err)
	}

	in := bufio.NewReader(os.Stdin)
	for {
		line, err := in.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		line = line[0 : len(line)-1]
		var reply string
		if line == "ready" {
			err = client.Call("EPaxos.ReadyProbe", "hello", &reply)
			if err != nil {
				log.Println(err)
			}
			log.Println(reply)
		} else if strings.HasPrefix(line, "send ") {
			id, err := strconv.ParseInt(line[5:], 10, 64)
			if err != nil {
				log.Println(err)
				continue
			}
			err = client.Call("EPaxos.SendProbe", id, &reply)
			if err != nil {
				log.Println(err)
				continue
			}
			log.Println(reply)
		} else {
			log.Println("ready or send #")
		}
	}
}
