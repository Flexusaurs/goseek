package tests

import (
	"context"
	"goseek/pkg/actors"
	"sync"
	"testing"
	"time"
)

func TestActorReceivesMessages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sys := actors.NewSystem(ctx)

	var received []string
	var mu sync.Mutex

	actorFunc := func(ctx context.Context, self *actors.PID, msg actors.Message) {
		mu.Lock()
		received = append(received, msg.(string))
		mu.Unlock()
	}

	pid := sys.Spawn(actorFunc, 10)

	pid.Send("hello")
	pid.Send("World")

	//goroutines time to process

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 2 || received[0] != "hello" || received[1] != "world" {
		t.Fatalf("expected [hello world], got %#v", received)
	}
}

func TestSupervisorRestartsCrashedActor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sys := actors.NewSystem(ctx)

	//child logic intentionally crashes
	crashActor := func(ctx context.Context, self *actors.PID, msg actors.Message) {
		if msg.(string) == "crash" {
			panic("boom")
		}
	}

	//create supervisor
	supervisorLogic := actors.Supervisor(
		actors.RestartAlways,
		crashActor,
		5,
		sys,
	)

	superPID := sys.Spawn(supervisorLogic, 10)

	//trigger supervisor startup

	superPID.Send(actors.Started{})

	time.Sleep(20 * time.Millisecond)

	child := superPID.Child()
	if child == nil {
		t.Fatalf("supervisor did not spawn child on Started message")
	}

	child.Send("crash")

	time.Sleep(20 * time.Millisecond)

	newChild := superPID.Child()

	if newChild == nil {
		t.Fatalf("supervisor did not replace child after crash")
	}

	if newChild == child {
		t.Fatalf("supervisor did not restart child, child PID identical after crash")

	}
}

func TestMailboxOverflowReported(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sys := actors.NewSystem(ctx)

	var wg sync.WaitGroup
	wg.Add(1)

	// Supervisor listening for overflow
	supervisor := func(ctx context.Context, self *actors.PID, msg actors.Message) {
		switch msg.(type) {
		case actors.MailboxOverflow:
			wg.Done()
		}
	}

	superPID := sys.Spawn(supervisor, 20)

	// Child with tiny mailbox to force overflow
	child := sys.SpawnWithSupervisor(superPID, func(ctx context.Context, self *actors.PID, msg actors.Message) {
		time.Sleep(time.Hour) // intentionally block so inbox fills
	}, 1)

	// Send 2 messages: 1 consumes slot, 1 overflows
	child.Send("1")
	child.Send("2") // overflow triggers supervisor notification

	wait := make(chan struct{})
	go func() {
		wg.Wait()
		close(wait)
	}()

	select {
	case <-wait:
		// OK
	case <-time.After(time.Second):
		t.Fatalf("expected mailbox overflow to be reported to supervisor")
	}
}
