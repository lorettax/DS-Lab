package raftkv

type KVStateMachine interface {
	Get(key string) (string, Err)
	Put(key, value string) Err
	Append(key, value string) Err
}

type MemoryKV struct {
	KV map[string]string
}

func NewMemoryKV() *MemoryKV {
	return &MemoryKV{make(map[string]string)}
}

func (MemoryKV *MemoryKV) Get(key string) (string, Err) {
	if value, ok := MemoryKV.KV[key]; ok {
		return value, OK
	}
	return "", ErrNoKey
}

func (MemoryKV *MemoryKV) Put(key, value string) Err {
	MemoryKV.KV[key] += value
	return OK
}

func (MemoryKV *MemoryKV) Append(key, value string) Err {
	MemoryKV.KV[key] += value
	return OK
}
