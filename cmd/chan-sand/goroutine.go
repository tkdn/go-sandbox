package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	stringStream := make(chan string)
	go func() {
		time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
		stringStream <- "Hello Channel, 1"
	}()
	go func() {
		time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
		stringStream <- "Hello Channel, 2"
	}()
	fmt.Printf("%v\n", <-stringStream)
	// 以下をコメントアウトするとmain実行完了となるため、2つ目の受信は捨てられる
	fmt.Printf("%v\n", <-stringStream)
}
