package actors

import (
	"context"
	"fmt"
)

//all types set for any for now, TODO: msg types

type Message any

type PID struct {
	inbox      chan Message
	supervisor *PID
	child      *PID
}

type Actor func(ctx context.Context, self *PID, msg Message)

func (p *PID) Send(msg Message) {
	select {
	case p.inbox <- msg:
	default:
		//
		if p.supervisor != nil {
			p.supervisor.Send(MailboxOverflow{
				Actor: p,
				Msg:   msg,
			})
			fmt.Printf("Mailbox full for actor %p, dropping message %T\n", p, msg)
		}
	}
}
