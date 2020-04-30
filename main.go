package main

import (
	"github.com/mpenick/robot/control"
	"log"
	"time"
)

func main() {
	ctrl, err := control.NewControl()
	if err != nil {
		log.Fatalf("unable to create robot control: %v", err)
	}
	ctrl.Green(1)
	ctrl.Motor2(0x89)
	//control.Motor2(0x02)
	log.Println("after")
	<-time.After(5 * time.Second)
}
