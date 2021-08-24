package kvpaxos

const (
	OK       = "OK"
	ErrNoKey = "ErrNoKey"
)

type Err string

// Put or Append
type PutAppendArgs struct {
	//  have to add definitions here.
	Key   string
	Value string
	Op    string // "Put" or "Append"
	//  have to add definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
}

type PutAppendReply struct {
	Err Err
}

type GetArgs struct {
	Key string
	//  have to add definitions here.
}

type GetReply struct {
	Err   Err
	Value string
}
