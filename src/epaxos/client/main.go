package main

import "os"
import "log"
import "bufio"
import "net/rpc"

func main() {
	client, err := rpc.Dial("tcp", "localhost:23333")
	if err != nil {
		log.Fatal(err)
	}

	in := bufio.NewReader(os.Stdin)
	for {
		line, _, err := in.ReadLine()
		if err != nil {
			log.Fatal(err)
		}
		var reply bool
		err = client.Call("EPaxos.TODO", line, &reply) // TODO
		if err != nil {
			log.Fatal(err)
		}
	}
}
