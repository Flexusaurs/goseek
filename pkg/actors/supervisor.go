package actors

import (
	"context"
)

type RestartPolicy int

const (
	RestartNever RestartPolicy = iota
	RestartAlways
)

// Supervisor returns an actor which spawns and (optionally) restarts a single child actor.
// policy: RestartAlways will respawn the child on ActorCrashed.
func Supervisor(policy RestartPolicy, childLogic Actor, mailbox int, system *ActorSystem) Actor {
	return func(ctx context.Context, self *PID, msg Message) {
		switch m := msg.(type) {
		case Started:
			// spawn first child
			child := system.SpawnWithSupervisor(self, childLogic, mailbox)
			self.SetChild(child)
		case ActorCrashed:
			// note: m contains the crash info if needed
			if policy == RestartAlways {
				child := system.SpawnWithSupervisor(self, childLogic, mailbox)
				self.SetChild(child)
			}
		case MailboxOverflow:
			// optional: handle overflow (log/resize/notify). keep noop here.
			_ = m
		}
	}
}
