package raftkv

import "fmt"

type CommandRequest struct {
	Key       string
	Value     string
	Op        OperationOp
	ClientId  int64
	CommandId int64
}

func (request CommandRequest) String() string {
	return fmt.Sprintf("{Key:%v,Value:%v,Op:%v,ClientId:%v,CommandId:%v}", request.Key, request.Value, request.Op, request.ClientId, request.CommandId)
}

type CommandResponse struct {
	Err   Err
	Value string
}

func (response CommandResponse) String() string {
	return fmt.Sprintf("{Err:%v,Value:%v}", response.Err, response.Value)
}
