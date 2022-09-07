package fsm

import (
	. "ConfigsAndTypes/config"
	"Driver-go/elevio"
	. "Requests/requests"
	"fmt"
	"time"
)

func InitElev(port string) {
	var elev Elevator
	elevio.Init("localhost:"+port, NumFloors)
	setAllLights(elev)

	elevio.SetMotorDirection(elevio.MD_Down) 
	for elevio.GetFloor() == -1 {
	}
	elevio.SetMotorDirection(elevio.MD_Stop)
	elevio.SetFloorIndicator(elevio.GetFloor())
}

func RunElevator(
	ElevFloor 			<-chan int,
	LocalOrder 			<-chan Order,
	IsOnline 			<-chan bool,
	LocalElevUpdate 	chan<- Elevator,
	RecieveElevUpdate 	chan<- Elevator,
	MovingElev 			chan<- Elevator,
) {
	elev := Elevator{
		Id:         ElevIP(),
		Floor:      elevio.GetFloor(),
		Dir:        elevio.MD_Stop,
		Behavior:   EB_Idle,
		IsOnline:   false,
		OrderQueue: [NumFloors][NumButtons]bool{},
		IsMoving:   true,
	}

	doorTimeout := time.NewTimer(Door_Open_Time * time.Millisecond)
	doorTimeout.Stop()

	engineFailure := time.NewTimer(3 * time.Second)
	engineFailure.Stop()

	var obstructionCounter = 0 

	for {
		switch elev.Behavior {
		case EB_Idle:
			select {
			case isOnline := <-IsOnline:
				elev.IsOnline = isOnline
			case newOrder := <-LocalOrder:
				elev.Id = newOrder.Id
				if elev.Floor == newOrder.Floor {
					elev.Behavior = EB_Dooropen
					doorTimeout.Reset(Door_Open_Time * time.Millisecond)
				} else {
					elev.OrderQueue[newOrder.Floor][newOrder.Button] = true
					elev.Behavior = EB_Moving
					elev.Dir = RequestChooseDirection(elev)
					engineFailure.Reset(3 * time.Second)
				}
				break
			}
		case EB_Moving:
			select {
			case isOnline := <-IsOnline:
				elev.IsOnline = isOnline
			case newOrder := <-LocalOrder:
				elev.OrderQueue[newOrder.Floor][newOrder.Button] = true
				break
			case newFloor := <-ElevFloor:
				elev.Floor = newFloor
				elev.IsMoving = true
				if RequestShouldStop(elev) {
					elev = RequestClearCurrentFloor(elev, nil)
					elev.Dir = elevio.MD_Stop
					elev.Behavior = EB_Dooropen
					doorTimeout.Reset(Door_Open_Time * time.Millisecond)
					engineFailure.Stop()
				} else {
					engineFailure.Reset((3 * time.Second))
				}
				break
			case <-engineFailure.C:
				fmt.Println("ENGINE FAILURE")
				if elev.IsMoving {
					elev.IsMoving = false
					MovingElev <- elev
				}
				engineFailure.Reset((1 * time.Second))
			}
		case EB_Dooropen:
			select {
			case isOnline := <-IsOnline:
				elev.IsOnline = isOnline
			case newOrder := <-LocalOrder:
				if elev.Floor == newOrder.Floor {
					elev.Behavior = EB_Dooropen
					doorTimeout.Reset(Door_Open_Time * time.Millisecond)
				} else {
					elev.OrderQueue[newOrder.Floor][newOrder.Button] = true
				}
				break
			case <-doorTimeout.C:
				obstructed := elevio.GetObstruction()
				elev.Dir = RequestChooseDirection(elev)
				if obstructed {
					fmt.Println("OBSTRUCTED")
					doorTimeout.Reset(Door_Open_Time * time.Millisecond)
					elev.Behavior = EB_Dooropen
					elev.Dir = elevio.MD_Stop
					obstructionCounter++
					if obstructionCounter == 3 {
						obstructionCounter = 0
						if elev.IsMoving {
							elev.IsMoving = false
							MovingElev <- elev
						}
					}
				} else if elev.Dir == elevio.MD_Stop {
					elev.Behavior = EB_Idle
					elev.IsMoving = true
					engineFailure.Stop()
					obstructionCounter = 0
				} else {
					elev.Behavior = EB_Moving
					elev.IsMoving = true
					engineFailure.Reset((3 * time.Second))
					obstructionCounter = 0
				}
				break
			}
		}

		WriteToBackup(elev)
		elevio.SetFloorIndicator(elev.Floor)
		elevio.SetMotorDirection(elev.Dir)
		elevio.SetDoorOpenLamp(EB_Dooropen == elev.Behavior)

		LocalElevUpdate   <- elev
		RecieveElevUpdate <- elev
	}
}

func setAllLights(elev Elevator) {

	for floor := 0; floor < NumFloors; floor++ {
		for btn := elevio.ButtonType(0); btn < NumButtons; btn++ {
			elevio.SetButtonLamp(btn, floor, elev.OrderQueue[floor][btn])
		}
	}
}