package hallAssigner

import (
	. "ConfigsAndTypes/config"
	"Driver-go/elevio"
	"Network-go/network/peers"
	"Requests/requests"
	"fmt"
)

var AllElevators 	map[string]Elevator
var SetLights 		map[string]bool
var myId = ElevIP()

func Assigner(
	ElevButtons 		<-chan elevio.ButtonEvent,
	PeerUpdateCh 		<-chan peers.PeerUpdate,
	MovingElev 			<-chan Elevator,
	RecieveElevUpdate 	<-chan Elevator,
	SendOrder 			chan<- Order,
	LocalOrder 			chan<- Order,
	LocalElevUpdate 	chan<- Elevator,
	IsOnline			chan<- bool,
) {
	for {
		select {
		case buttonPress := <-ElevButtons:
			id := "No ID"
			myElev := AllElevators[myId]
			if myElev.IsOnline && NumElevs > 1 {
				if buttonPress.Button == elevio.BT_Cab {
					id = myId
				} else {
					id = costFunction(AllElevators, buttonPress.Button, buttonPress.Floor)
				}
				if !duplicateOrder(buttonPress.Button, buttonPress.Floor) {
					newOrder := Order{Floor: buttonPress.Floor, Button: buttonPress.Button, Id: id}
					SendOrder <- newOrder
				}
			} else {
				newOrder := Order{Floor: buttonPress.Floor, Button: buttonPress.Button, Id: myId}
				LocalOrder <- newOrder
				fmt.Println("SINGLE ELEVATOR MODE")
			}
		case updatedElev := <-RecieveElevUpdate:
			AllElevators[updatedElev.Id] = updatedElev
			setAllLights()
		case myElev := <-MovingElev:
			AllElevators[myElev.Id] = myElev
			setAllLights()
			reassignOrders(myElev, SendOrder)
		case peer := <-PeerUpdateCh:
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", peer.Peers)
			fmt.Printf("  New:      %q\n", peer.New)
			fmt.Printf("  Lost:     %q\n", peer.Lost)
			if peer.New != "" {
				if elev, foundPeer := AllElevators[peer.New]; foundPeer {
					elev.IsOnline = true
					AllElevators[peer.New] = elev
				} else {
					elev := Elevator{
						Id:         peer.New,
						Floor:      0,
						Dir:        elevio.MD_Stop,
						Behavior:   EB_Idle,
						IsOnline:   true,
						OrderQueue: [NumFloors][NumButtons]bool{},
						IsMoving:   true,
					}
					AllElevators[peer.New] = elev
				}
				if peer.New == myId {
					IsOnline <- true
				}
				LocalElevUpdate <- AllElevators[myId]
				NumElevs++
				fmt.Printf("Number of Elevators: %d\n", NumElevs)
			}
			if len(peer.Lost) > 0 {
				for _, lostPeer := range peer.Lost {
					elev := AllElevators[lostPeer]
					elev.IsOnline = false
					if lostPeer == myId {
						IsOnline <- false
					}
					AllElevators[lostPeer] = elev
					NumElevs--
					fmt.Printf("Number of Elevators: %d\n", NumElevs)
					reassignOrders(elev, SendOrder)
				}
			}
		}
	}
}

func costFunction(
	allElevators 	map[string]Elevator, 
	btn 			elevio.ButtonType, 
	floor 			int,
	) string {
	minTime := -1
	minId := "Undefined"
	for id, elev := range allElevators {
		if elev.IsOnline && elev.IsMoving {
			time := timeToServeRequest(elev, btn, floor)
			if time < minTime || minTime == -1 {
				minTime = time
				minId = id
			}
		}
	}
	return minId

}

func sendReassignedOrder(
	absentElev 	Elevator,
	SendOrder 	chan<- Order,
) {
	for floor := 0; floor < NumFloors; floor++ {
		for btn := 0; btn < NumButtons-1; btn++ {
			if absentElev.OrderQueue[floor][btn] && !duplicateOrder(elevio.ButtonType(btn), floor) {
				id := costFunction(AllElevators, elevio.ButtonType(btn), floor)
				newOrder := Order{Floor: floor, Button: elevio.ButtonType(btn), Id: id}
				SendOrder <- newOrder
			}
		}
	}
}

func reassignOrders(
	absentElev 	Elevator,
	SendOrder 	chan<- Order,
) {
	switch absentElev.IsMoving {
	case true:
		for id, elev := range AllElevators {
			if elev.IsOnline {
				if id == myId {
					sendReassignedOrder(absentElev, SendOrder)
				}
				break
			}
		}
	case false:
		sendReassignedOrder(absentElev, SendOrder)
	}

}

func timeToServeRequest(
	elev Elevator, 
	btn elevio.ButtonType, 
	floor int,
	) int {
	elev.OrderQueue[floor][btn] = true
	arrivedAtRequest := false
	ifEqual := func(innerBtn elevio.ButtonType, innerFloor int) {
		if innerBtn == btn && innerFloor == floor {
			arrivedAtRequest = true
		}
	}
	duration := 0
	switch elev.Behavior {
	case EB_Idle:
		elev.Dir = requests.RequestChooseDirection(elev)
		if elev.Dir == elevio.MD_Stop {
			return duration
		}
		break
	case EB_Moving:
		duration += Travel_Time / 2
		elev.Floor += int(elev.Dir)
		break
	case EB_Dooropen:
		duration -= Door_Open_Time / 2
	}
	for {
		if requests.RequestShouldStop(elev) {
			elev = requests.RequestClearCurrentFloor(elev, ifEqual)
			if arrivedAtRequest {
				return duration
			}
			duration += Door_Open_Time
			elev.Dir = requests.RequestChooseDirection(elev)
		}
		elev.Floor += int(elev.Dir)
		duration += Travel_Time
	}
}

func setAllLights() {
	var lightsOff bool
	for floor := 0; floor < NumFloors; floor++ {
		for btn := 0; btn < NumButtons; btn++ {
			for id, elev := range AllElevators {
				SetLights[id] = false
				if btn == elevio.BT_Cab && id != myId {
					continue
				}

				if elev.OrderQueue[floor][btn] && ((elev.IsOnline && elev.IsMoving) || (!elev.IsOnline && id == myId) || (!elev.IsMoving && btn == elevio.BT_Cab)) {
					SetLights[id] = true
					elevio.SetButtonLamp(elevio.ButtonType(btn), floor, true)
				}
			}
			lightsOff = true
			for _, val := range SetLights {
				if val == true {
					lightsOff = false
				}
			}
			if lightsOff {
				elevio.SetButtonLamp(elevio.ButtonType(btn), floor, false)
			}
		}
	}
}

func duplicateOrder(
	btn elevio.ButtonType, 
	floor int,
	) bool {
	for id, elev := range AllElevators {
		if btn == elevio.BT_Cab && id != myId {
			continue
		}
		if elev.OrderQueue[floor][btn] && elev.IsOnline && elev.IsMoving {
			return true
		}
	}
	return false
}
