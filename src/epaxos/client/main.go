package main

import "os"
import "bufio"
import "net/rpc"
import "log"
import "epaxos/common"

func main() {
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
		log.Println(line)
		var reply string
		err = client.Call("EPaxos.HelloWorld", line, &reply) // TODO
		if err != nil {
			log.Fatal(err)
		}
		log.Println(reply)
	}
}
