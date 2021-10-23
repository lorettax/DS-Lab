## 分布式相关算法原型Lab，包含raft,paxos等。

### 一. raft算法部分

论文复现，完成一个小kv分布式文件服务。

### 服务模型图：

![d03ea63de8357c2ac881e97b985f39a](https://user-images.githubusercontent.com/50191422/138548789-eefdc3d6-6b50-48dc-b14b-6fede57e86c9.png)

#### 实现领导者选举机制，当现在的Leader挂掉以后，必须及时从peers选出一个新的Leader,角色转换机制如Paper中图：

![61b3d0401befac813a40a6b5886b02e](https://user-images.githubusercontent.com/50191422/138554219-6a24ba8a-c0c9-466a-b056-e5c34c3251e9.png)

RSM设计：
    
 Raft实例有两种时间驱动的(time-driven)活动：(1) 领导者必须发送心跳，(2) 以及其他(对等点)自收到领导者消息以来(since hearing from the leader)，如果太长时间过去(if too much time has passed)，开始一次选举。最好使用一个专门的(dedicated)、长期运行的(long-running)goroutine来驱动这两者中的每个活动，而不是将多个活动组合到单个goroutine中。
    
  1. 心跳超时检测。
        
        ```go
        // leader检查距离上次发送心跳的时间（latestIssueTime）是否超过了心跳周期（heartbeatPeriod）
        func (rf *Raft) heartbeatPeriodTick(){...}
        ```
        
 2. 选举超时检测。
        
        ```go
        // 定期检查自最新一次从leader那里收到AppendEntries RPC(包括heartbeat)
        // 或给予candidate 的RequestVote RPC请求的投票的时间（latestHeardTime）以来的时间差，是否超过了
        // 选举超时时间（electionTimeout）.若超时，则往electionTimeoutChan写入数据，以表明可以发起选举。
        func (rf *Raft) electionTimeoutTick(){...}
        ```
        
 3. 主消息处理，处理两种互斥时间。
        
        ```go
        // implement;
        func (rf *Raft) eventLoop(){...}
        ```
        
  4. Raft结构体
        
        ```go
        type Raft struct {
        	mu        sync.Mutex          // Lock to protect shared access to this peer's state
        	peers     []*labrpc.ClientEnd // RPC end points of all peers
        	persister *Persister          // Object to hold this peer's persisted state
        	me        int                 // this peer's index into peers[]
        
        	dead int32
        	applyCh chan ApplyMsg
        	state    int
        	leaderId int
        	applyCond *sync.Cond // 更新commitIndex时，新提交的条目的信号
        	leaderCond    *sync.Cond
        	nonLeaderCond *sync.Cond
        	electionTimeout int
        	heartbeatPeriod int
        	electionTimeoutChan chan bool
        	heartbeatPeriodChan chan bool
        	CurrentTerm int
        	VoteFor     int
        	Log         []LogEntry
        	commitIndex int
        	lastApplied int
        
        	nVotes int
        
        	nextIndex  []int
        	matchIndex []int
        
        	latestHeardTime int64 // 最新的收到leader的AppendEntries RPC(包括heartbeat)  或给予candidate的RequestVote RPC投票的时间
        	latestIssueTime int64 // 最新的leader发送心跳的时间
        }
        ```
 
![image](https://user-images.githubusercontent.com/50191422/138550122-72c2857b-672f-468a-801f-b2d9765d879d.png)    
    
 5. rpc相关：
    
  为了提高发送rpc性能，采用并行发送，迭代peers，为每个peer单独创建一个goroutine发送rpc,在同一个goroutine里进行RPC回复(reply)处理是最简单的，创建投票统计，一但发现获得rpc的统计数超过一般以上，将立刻切换状态，并立即发送一次心跳，防止发起新的选举。
    
    ```go
    func (rf *Raft) startElection() {...}
    ```
    
  rpc超时处理问题。根据Raft算法论文中的Rule for Server规则，如果在网络的请求或者回复中有Term≥当前任期Term, 则更新当前Term为最大Term，并切换Follower状态。
    

#### 日志复制，leader将从Start()接收到的新的指令作为日志条目添加到本次日志后，然后给其它peers发起rpc服务，请求复制entries至follower。

          ![image](https://user-images.githubusercontent.com/50191422/138549717-3b1fe5b2-cec5-4308-8bd3-32d24e5ce937.png)

1. Leader接收到客户端的日志条日之后，先将Log Entry添加到自己的日志当中去，然后发送RPC给其它Peers同意其内容，并完成日志提交。
2. leader在提升commitIndex之前，需要保证本次提交之后的index要大于当前commitIndex,
3. 达到多数条件时必须检测状态机的status是否还处于之前的leader的状态，防止因为peers中有term大于当前请求或回复中的term导致当前状态被切换至Follower状态，而在Follower的状态时又再次收到大部分回复，防止这时错误提升commitIndex。
4. 当Leader在local进行提交entry之后，须发一次心跳告诉peers来提升commitIndex,同时对还没有复制该Entry的peers，在这次心跳采用携带上次提交之后到本次提交之间的entry,再更新peers的状态机。
5. 对一致性检查之后的冲突条目进行日志替换，删除冲突条目，寻找冲突的第一个点，需要向前递减index，直至找到冲突的第一个点。该地方存在一个优化项，如果冲突点需要向前递减的条目非常多，会发生多次网络请求，这个时候可以采用一些压缩手段或其它减少rpc次数来进行优化。
6. 在一致性检查通过之后，收到rpc的peer将自己状态机设为Follower。
7. Rpc处理，extended Raft论文中有提到，如果需要的话，算法可以通过减少被拒绝的追加条目(AppendEntries) RPC的次数来优化。例如，当追加条目(AppendEntries) RPC的请求被拒绝时，跟随者可以包含冲突条目的任期号和它自己存储的那个任期的第一个索引值。借助这个信息，领导者可以减少nextIndex来越过该任期内的所有冲突的日志条目；这样就变为每个任期需要一条追加条目(AppendEntries) RPC而不是每个条目一条，意思为AppendEtries携带的entry必须在发生冲突的点之前，不能在之后。

```go
func (rf *Raft) Start(command interface{}) {...}
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply{...}
...
```

8. 心跳处理，判断nextIndex的值是否大于log的尾后位置，如果是，则心跳不需要携带Entries信息，如果不是，则需要携带期间的entries信息，在一致性检查通过之后，提升commitedIndex。
    
    ```go
    func (rf *Raft) broadcastHeartbeat() {...}
    ...
    ```
    
9. 新增长期守护型groutine来应用日志条目。循环检测commitIndex是否大于lastApplied，如果是，则立即唤醒线程，将期间的数据封装成一个ApplyMsg来发送到applyCh,如果不是，则线程进行休眠即可，等待唤醒。

```go
// long thread 应用commited日志条目
func (rf *Raft) applyEntries() {...}
```

10. 领导者安全特性（任何时候都要保证）
10.1 Raft算法的安全属性是状态机必须要保证的，例如，任意的服务器本地已经完成了一个确定的entry提交动作，那么其余peers也要在同一个位置完成相同的Entry提交。
10.2. 同一个任期内只能有一个Leader被选出。

### 实现Client端到Server的KV存储服务

![593511591328c8f9d62f1348e48889d](https://user-images.githubusercontent.com/50191422/138549897-6b159009-5569-4821-9d74-7cf6cdedc00a.png)


client向server端提交一条日志，server在状态机中成功同步commit，返回一个执行结果给客户端，只要大多服务器存活，哪怕有一些网络分区等故障，依旧保证处理客户端请求。

![4833da889f13f66fec4e37c2d5b648f](https://user-images.githubusercontent.com/50191422/138549904-a92f7564-b7c9-40d2-9b44-3d9ab03ef393.png)

1. 实现线性化语义，即便命令被执行多次，返回请求仍然保持幂等性，客户端保证唯一的clientId,server保证接收到clientId+commandId记录是否被应用以及被应用之后的结果，服务端用map保存，key为clientId，value为command和状态机结果即可。

![6c549e9b25e1afa8cece94c1f1044d4](https://user-images.githubusercontent.com/50191422/138549988-7f1310c3-fdfe-414c-bc64-cc09c8d92f0a.png)

2. 对map的设计需要考虑其大小，集群需要对过期淘汰达成共识。
3. 客户端在范围时间内没有收到rpc响应回复，则对命令进行重新发送。

![1596cd1ca6bc817f2d4df5ac56c709e](https://user-images.githubusercontent.com/50191422/138549910-abc3760b-28f9-4965-b125-843d39d9c704.png)

4. 客户端：
    
    ```go
    func (ck *Clerk) Get(key string) string {...}
    func (ck *Clerk) Put(key string, value string) {...}
    func (ck *Clerk) Append(key string, value string) {...}
    ...
    ```
    
5. 服务端结构体：
    
    ```go
    type KVServer struct {
    	mu      sync.RWMutex
    	dead    int32
    	rf      *raft.Raft
    	applyCh chan raft.ApplyMsg
    	maxRaftState int // snapshot if log grows this big
    	lastApplied  int // record the lastApplied to prevent stateMachine from rollback
    	stateMachine   KVStateMachine                // KV stateMachine
    	lastOperations map[int64]OperationContext    
    	notifyChans    map[int]chan *CommandResponse
    }
    ...
    ```
 
 ##### 更多文档待更新...
 
 ### 资料：
    Raft算法英文在线原版：https://raft.github.io/raft.pdf
    Raft算法中文在线版：https://www.ulunwen.com/archives/229938
    ...
