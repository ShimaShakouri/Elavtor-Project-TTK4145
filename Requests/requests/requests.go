package requests

import (
	. "ConfigsAndTypes/config"
	"Driver-go/elevio"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	. "time"
)

func RequestChooseDirection(elev Elevator) elevio.MotorDirection {
	if RequestAbove(elev) {
		return elevio.MD_Up
	} else if RequestBelow(elev) {
		return elevio.MD_Down
	} else {
		return elevio.MD_Stop
	}
}

func RequestAbove(elev Elevator) bool {
	for floor := elev.Floor + 1; floor < NumFloors; floor++ {
		for btn := elevio.ButtonType(0); btn < NumButtons; btn++ {
			if elev.OrderQueue[floor][btn] {
				return true
			}
		}
	}
	return false
}

func RequestBelow(elev Elevator) bool {
	for floor := 0; floor < elev.Floor; floor++ {
		for btn := elevio.ButtonType(0); btn < NumButtons; btn++ {
			if elev.OrderQueue[floor][btn] {
				return true
			}
		}
	}
	return false
}

func RequestShouldStop(elev Elevator) bool {
	switch elev.Dir {
	case elevio.MD_Down:
		return elev.OrderQueue[elev.Floor][elevio.BT_HallDown] ||
			elev.OrderQueue[elev.Floor][elevio.BT_Cab] ||
			!RequestBelow(elev)
	case elevio.MD_Up:
		return elev.OrderQueue[elev.Floor][elevio.BT_HallUp] ||
			elev.OrderQueue[elev.Floor][elevio.BT_Cab] ||
			!RequestAbove(elev)
	default:
		return true
	}
}

func RequestClearCurrentFloor(
	elev			 	Elevator, 
	onClearedRequest 	func(elevio.ButtonType, int),
	) Elevator {
	elev.OrderQueue[elev.Floor][elevio.BT_Cab] = false
	haveFunction := !(onClearedRequest == nil)
	switch elev.Dir {
	case elevio.MD_Up:
		if haveFunction {
			onClearedRequest(elevio.BT_HallUp, elev.Floor)
		}
		elev.OrderQueue[elev.Floor][elevio.BT_HallUp] = false
		if !RequestAbove(elev) {
			if haveFunction {
				onClearedRequest(elevio.BT_HallDown, elev.Floor)
			}
			elev.OrderQueue[elev.Floor][elevio.BT_HallDown] = false
		}
		break
	case elevio.MD_Down:
		if haveFunction {
			onClearedRequest(elevio.BT_HallDown, elev.Floor)
		}
		elev.OrderQueue[elev.Floor][elevio.BT_HallDown] = false
		if !RequestBelow(elev) {
			if haveFunction {
				onClearedRequest(elevio.BT_HallUp, elev.Floor)
			}
			elev.OrderQueue[elev.Floor][elevio.BT_HallUp] = false
		}
		break

	default: // in case elevio.MD_Stop or anything undefined
		if haveFunction {
			onClearedRequest(elevio.BT_HallUp, elev.Floor)
			onClearedRequest(elevio.BT_HallDown, elev.Floor)
		}
		elev.OrderQueue[elev.Floor][elevio.BT_HallUp] = false
		elev.OrderQueue[elev.Floor][elevio.BT_HallDown] = false
		break
	}
	return elev
}

func WriteToBackup(elev Elevator) {
	filename := "cabBackup.txt"
	f, err := os.Create(filename)
	if err != nil {
		return
	}
	caborders := make([]bool, 0)
	for _, row := range elev.OrderQueue {
		caborders = append(caborders, row[NumButtons-1])
	}
	cabordersString := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(caborders)), " "), "[]")
	_, err = f.WriteString(cabordersString)
	defer f.Close()
}

func ReadFromBackup(
	HwButtons 	chan<- elevio.ButtonEvent,
	) {
	filename := "cabBackup.txt"
	f, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	CabOrder := make([]bool, 0)
	if err == nil {
		s := strings.Split(string(f), " ")
		for _, item := range s {
			result, _ := strconv.ParseBool(item)
			CabOrder = append(CabOrder, result)
		}
	}
	Sleep(20 * Millisecond)
	for f, order := range CabOrder {
		if order {
			backupOrder := elevio.ButtonEvent{Floor: f, Button: elevio.BT_Cab}
			HwButtons <- backupOrder
			Sleep(20 * Millisecond)
		}
	}
}
