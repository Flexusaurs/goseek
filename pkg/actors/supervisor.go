package actors

import (
	"context"
	"fmt"
)

type RestartPolicy int

const (
	RestartNever RestartPolicy = iota
	RestartAlways
)

//supervisor is an actor for receiving crash reports and restarts children proc.

func Supervisor(policy RestartPolicy, childLogic Actor, mailbox int, system *ActorSystem) Actor {
	return func(ctx context.Context, self *PID, msg Message) {
		switch msg.(type) {
		case Started:
			//spawn first child
			self.child = system.SpawnWithSupervisor(self, childLogic, mailbox)

		case ActorCrashed:
			if policy == RestartAlways {
				self.child = system.SpawnWithSupervisor(self, childLogic, mailbox)
			}

		case MailboxOverflow:
			//TODO: overflow logic: log/drop/resize
			_ = msg
			fmt.Println("mailbox overflow")

		}
	}
}

/*
	 func (self *PID) spawnChild(ctx context.Context, child Actor, mailbox int) *PID {
		system := NewSystem(ctx)
		return system.SpawnWithSupervisor(self, child, mailbox)
	}
*/
func (self *PID) SetChild(pid *PID) {
	self.child = pid
}

//var childkey = struct{}{}

func (p *PID) Child() *PID {
	return p.child
}
