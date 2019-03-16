package main

import "fmt"
import "time"

func main() {
	fmt.Println("This is server")
	time.Sleep(30 * time.Second)
	fmt.Println("Exiting")
}
