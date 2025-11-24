package actors

type Stop struct{}
type Ping struct{}
type Started struct{}

type ActorCrashed struct {
	Actor *PID
	Err   error
}

type MailboxOverflow struct {
	Actor *PID
	Msg   Message
}
