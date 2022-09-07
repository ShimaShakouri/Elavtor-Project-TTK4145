package main

import (
	. "ConfigsAndTypes/config"
	"Driver-go/elevio"
	. "FSM/fsm"
	. "HallAssigner/hallAssigner"
	"Network-go/network/bcast"
	"Network-go/network/peers"
	. "OrderHandler/orderHandler"
	. "Requests/requests"
	"flag"
)

func main() {

	var port string
	flag.StringVar(&port, "p", "15657", "port server")
	flag.Parse()
	InitElev(port)

	AllElevators 	  = make(map[string]Elevator)
	SetLights      	  = make(map[string]bool)
	PrevRxMsgIds 	  = make(map[string]int)

	//Order Channels
	SendOrder 		  := make(chan Order)
	RecieveElevUpdate := make(chan Elevator)
	LocalOrder		  := make(chan Order)
	LocalElevUpdate   := make(chan Elevator)

	//Hardware Channels
	ElevButtons    	  := make(chan elevio.ButtonEvent)
	ElevFloor		  := make(chan int)
	Obstruction 	  := make(chan bool)

	//Network Channels
	PeerUpdateCh 	  := make(chan peers.PeerUpdate)
	PeerTxEnable 	  := make(chan bool)
	BcastMsg 	 	  := make(chan Message)
	RecieveMsg 		  := make(chan Message)
	IsOnline		  := make(chan bool)
	MovingElev 	  	  := make(chan Elevator)

	// Goroutines of Driver-go
	go elevio.PollButtons(ElevButtons)
	go elevio.PollFloorSensor(ElevFloor)
	go elevio.PollObstructionSwitch(Obstruction)

	// Goroutines of Network-go
	go bcast.Receiver(44444, RecieveMsg)
	go bcast.Transmitter(44444, BcastMsg)
	go peers.Receiver(55555, PeerUpdateCh)
	go peers.Transmitter(55555, ElevIP(), PeerTxEnable)

	// Goroutine of hallAssigner
	go Assigner(
		ElevButtons,
		PeerUpdateCh,
		MovingElev,
		RecieveElevUpdate,
		SendOrder,
		LocalOrder,
		LocalElevUpdate,
		IsOnline,
	)

	// Goroutine of orderHandler
	go ReceiveOrder(
		RecieveMsg,
		RecieveElevUpdate,
		LocalOrder,
		BcastMsg,
	)
	go TransmitOrder(
		SendOrder,
		LocalElevUpdate,
		LocalOrder,
		BcastMsg,
	)

	// Goroutine of FSM
	go RunElevator(
		ElevFloor,
		LocalOrder,
		IsOnline,
		RecieveElevUpdate,
		LocalElevUpdate,
		MovingElev,
	)
	go ReadFromBackup(ElevButtons)
	select {}
}
