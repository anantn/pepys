package main

import (
	"fmt"
	"./disk"
)

func main() {
	dk, err := disk.New("image")
	if (err != nil) {
		fmt.Printf("could not open disk %v\n", err)
		return
	}
	
	err = dk.CreateSuper()
	if (err != nil) {
		fmt.Printf("could not create superblock %v\n", err)
		return
	}
	
	fmt.Printf("got disk %v\n", dk)
}
