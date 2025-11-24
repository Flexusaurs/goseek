package actors

import (
	"context"
	"fmt"
	"sync"
)

//all types set for any for now, TODO: msg types

type Message any

type PID struct {
	inbox      chan Message
	supervisor *PID
	child      *PID
	mu         sync.RWMutex
}

type Actor func(ctx context.Context, self *PID, msg Message)

func (p *PID) Send(msg Message) {
	select {
	case p.inbox <- msg:
	default:
		// mailbox full: notify supervisor (non-blocking)
		p.mu.RLock()
		sup := p.supervisor
		p.mu.RUnlock()
		if sup != nil {
			// best-effort notify
			sup.Send(MailboxOverflow{
				Actor: p,
				Msg:   msg,
			})
		}
		// Log for debugging — keep it lightweight
		fmt.Printf("Mailbox full for actor %p, dropping message %T\n", p, msg)
	}
}

func (p *PID) Child() *PID {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.child
}

// SetChild sets the child PID in a concurrency-safe way.
func (p *PID) SetChild(child *PID) {
	p.mu.Lock()
	p.child = child
	p.mu.Unlock()
}

// setSupervisor sets supervisor pointer (internal use).
func (p *PID) setSupervisor(sup *PID) {
	p.mu.Lock()
	p.supervisor = sup
	p.mu.Unlock()
}
