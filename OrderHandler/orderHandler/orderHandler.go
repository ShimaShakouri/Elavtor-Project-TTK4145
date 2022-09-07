package orderHandler

import (
	//"Network-go/network/peers"
	"fmt"
	"strings"
	"time"

	. "ConfigsAndTypes/config"
	. "HallAssigner/hallAssigner"
)

var MsgQueue 		[]Message
var currentConf 	[]string
var PrevRxMsgIds 	map[string]int
var myId = ElevIP()

func TransmitOrder(
	SendOrder 			<-chan Order,
	LocalElevUpdate 	<-chan Elevator,
	LocalOrder 			chan<- Order,
	BcastMsg 			chan<- Message,
) {
	TxMsgID 		:= 0
	TxMessageTimer  := time.NewTimer(10 * time.Millisecond)
	packageNotSent 	:= 0
	for {
		select {
		case newOrder := <-SendOrder:
			fmt.Printf("Sending New Order\n")
			elevMsg := new(Elevator)
			txMsg 	:= Message{
				OrderMsg: newOrder,
				ElevMsg:  *elevMsg,
				MsgType:  ORDER,
				MsgId:    TxMsgID,
				ElevId:   myId,
			}
			MsgQueue = append(MsgQueue, txMsg)
			TxMsgID++
		case localElevUpdate := <-LocalElevUpdate:
			fmt.Printf("Local Update\n")
			orderMsg := new(Order)
			txMsg := Message{
				OrderMsg: *orderMsg,
				ElevMsg:  localElevUpdate,
				MsgType:  ELEVSTATUS,
				MsgId:    TxMsgID,
				ElevId:   myId,
			}
			MsgQueue = append(MsgQueue, txMsg)
			TxMsgID++
		case <-TxMessageTimer.C:
			if len(MsgQueue) != 0 {
				txMsg := MsgQueue[0]
				allElevs := AllElevators
				onlineElevs := 0
				elevsConfirmed := 0
				for id, elev := range allElevs {
					if elev.IsOnline {
						onlineElevs++
						for _, ConfirmedId := range currentConf {
							if id == ConfirmedId {
								elevsConfirmed++
								fmt.Printf("Number of Confirmed Elevators %d\n", elevsConfirmed)
							}
						}
					}
				}
				if packageNotSent == LostPackageCounter {
					fmt.Println("PACKAGE NOT SENT")
					txMsg.OrderMsg.Id = myId
					LocalOrder <- txMsg.OrderMsg
					MsgQueue = MsgQueue[1:]
					packageNotSent = 0
					currentConf = make([]string, 0)
				} else {
					if onlineElevs == elevsConfirmed || txMsg.MsgType == ELEVSTATUS {
						if txMsg.MsgType == ELEVSTATUS {
							BcastMsg <- txMsg
						}
						MsgQueue = MsgQueue[1:]
						currentConf = make([]string, 0)
						packageNotSent = 0
					} else {
						BcastMsg <- txMsg
						packageNotSent++
					}
				}
			}
			TxMessageTimer.Reset(10 * time.Millisecond)
		}
	}
}

func ReceiveOrder( 
	RecieveMsg 		   <-chan Message,
	RecieveElevUpdate  chan<- Elevator,
	LocalOrder 		   chan<- Order,
	BcastMsg 		   chan<- Message,
) {
	for {
		select {
		case rxMsg := <-RecieveMsg:
			switch rxMsg.MsgType {
			case ORDER:
				isDuplicate := checkForDuplicate(rxMsg)
				if !isDuplicate && rxMsg.OrderMsg.Id == myId {
					LocalOrder <- rxMsg.OrderMsg
				}
				Confirmation(rxMsg, BcastMsg)
			case ELEVSTATUS:
				RecieveElevUpdate <- rxMsg.ElevMsg
			case CONFIRMATION:
				ArrayId := strings.Split(rxMsg.ElevId, "\nFROM\n")
				toId := ArrayId[0]
				fromId := ArrayId[1]
				fmt.Println("received by:", fromId)
				fmt.Println("sent from:", toId)
				duplicateConfirm := false
				if toId == myId {
					for _, ConfirmedId := range currentConf {
						if ConfirmedId == fromId {
							duplicateConfirm = true
						}
					}
					if !duplicateConfirm {
						currentConf = append(currentConf, fromId)
					}
				}
			}
		}
	}
}

func Confirmation(
	rxMsg 		Message,
	BcastMsg 	chan<- Message,
) {
	txMsg := rxMsg
	txMsg.MsgType = CONFIRMATION
	txMsg.ElevId += "\nFROM\n" + myId
	ArrayId := strings.Split(rxMsg.ElevId, "\nFROM\n")
	Id := ArrayId[0]
	BcastMsg <- txMsg
	fmt.Println("Order is confirmed by: ", Id)
}

func checkForDuplicate(rxMsg Message) bool {
	if prevMsgId, found := PrevRxMsgIds[rxMsg.ElevId]; found {
		if rxMsg.MsgId > prevMsgId {
			PrevRxMsgIds[rxMsg.ElevId] = rxMsg.MsgId
			return false
		}
	} else {
		PrevRxMsgIds[rxMsg.ElevId] = rxMsg.MsgId
		return false
	}
	return true

}