// All Channels are used in different modules for both Read and Write (No Specified Direction)

package config

import (
	"Driver-go/elevio"
	"Network-go/network/localip"
	"fmt"
	"os"
)

var NumElevs = 0

const (
	NumButtons         = 3
	NumFloors          = 4
	Door_Open_Time     = 3000
	Travel_Time        = 2500
	LostPackageCounter = 30
)

type ElevatorBehavior int

const (
	EB_Idle ElevatorBehavior = iota
	EB_Moving
	EB_Dooropen
)

type Elevator struct {
	Id         string
	Floor      int
	Dir        elevio.MotorDirection
	Behavior   ElevatorBehavior
	IsOnline   bool
	IsMoving   bool
	OrderQueue [NumFloors][NumButtons]bool
}

type Order struct {
	Id     string
	Floor  int
	Button elevio.ButtonType
}


type MessageType int

const (
	ORDER MessageType = iota
	ELEVSTATUS
	CONFIRMATION
)

type Message struct {
	OrderMsg Order
	ElevMsg  Elevator
	MsgType  MessageType
	MsgId    int
	ElevId   string
}

func ElevIP() string {
	localIP, err := localip.LocalIP()
	if err != nil {
		fmt.Println(err)
		localIP = "DISCONNECTED"
	}
	id := fmt.Sprintf("%s-%d", localIP, os.Getpid())
	return id
}