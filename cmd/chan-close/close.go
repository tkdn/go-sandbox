package main

import "fmt"

func main() {
	s := []int{1, 2, 3, 4, 5}
	stream := make(chan int)
	go func() {
		defer close(stream)
		for _, v := range s {
			stream <- v * 2
		}
	}()
	for v := range stream {
		fmt.Printf("%v \n", v)
	}
}
