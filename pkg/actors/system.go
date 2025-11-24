package actors

import (
	"context"
	"errors"
)

type ActorSystem struct {
	ctx context.Context
}

func NewSystem(ctx context.Context) *ActorSystem {
	return &ActorSystem{ctx: ctx}
}

func (s *ActorSystem) Spawn(actor Actor, mailboxSize int) *PID {
	return s.SpawnWithSupervisor(nil, actor, mailboxSize)
}

func (s *ActorSystem) SpawnWithSupervisor(supervisor *PID, actor Actor, mailboxSize int) *PID {
	pid := &PID{
		inbox:      make(chan Message, mailboxSize),
		supervisor: supervisor, // initial supervisor
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// report to supervisor if present (best-effort)
				if supervisor != nil {
					supervisor.Send(ActorCrashed{
						Actor: pid,
						Err:   errors.New("actor crashed"),
					})
				}
			}
		}()

		for {
			select {
			case <-s.ctx.Done():
				return
			case msg := <-pid.inbox:
				actor(s.ctx, pid, msg)
			}
		}
	}()

	return pid
}
