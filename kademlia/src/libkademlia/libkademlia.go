package libkademlia

// Contains the core kademlia type. In addition to core state, this type serves
// as a receiver for the RPC methods, which is required by that package.

import (
	"container/heap"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"time"
)

const (
	alpha = 3
	b     = 8 * IDBytes
	k     = 20
)

// Kademlia type. You can put whatever state you need in this.
type Kademlia struct {
	NodeID          ID
	SelfContact     Contact
	hash            map[ID][]byte
	hashchannel     chan hashcommand
	updatechannel   chan updatecommand
	findchannel     chan findcommand
	registerchannel chan Client
	table           *RoutingTable
}

type hashcommand struct {
	key           ID
	cmd           int
	value         []byte
	returnchannel chan hashreturn
}

type updatecommand struct {
	contact Contact
}

type findcommand struct {
	clientid ID
	key      ID
	num      int
}

//result for "find values and nodes"
type findresult struct {
	Nodes []Contact
	Value []byte
	err   error
}

//result for "find contact"
type contactresult struct {
	node *Contact
}

//struct for registering in handtable function
type Client struct {
	findchan    chan findresult
	contactchan chan contactresult
	id          ID
	num         int
}

type hashreturn struct {
	value []byte
	ok    bool
}

func NewKademliaWithId(laddr string, nodeID ID) *Kademlia {
	k := new(Kademlia)
	k.NodeID = nodeID
	k.table = NewRoutingTable(nodeID)
	k.updatechannel = make(chan updatecommand)
	k.findchannel = make(chan findcommand)
	k.registerchannel = make(chan Client)
	k.hash = make(map[ID][]byte)
	k.hashchannel = make(chan hashcommand)
	go k.HandleTable()
	go k.HandleHash()
	kRPC := new(KademliaRPC)
	kRPC.kademlia = k
	// Set up RPC server
	// NOTE: KademliaRPC is just a wrapper around Kademlia. This type includes
	// the RPC functions.
	s := rpc.NewServer()
	s.Register(&KademliaRPC{k})
	hostname, port, err := net.SplitHostPort(laddr)
	if err != nil {
		return nil
	}
	s.HandleHTTP(rpc.DefaultRPCPath+port,
		rpc.DefaultDebugPath+port)
	l, err := net.Listen("tcp", laddr)
	if err != nil {
		log.Fatal("Listen: ", err)
	}

	// Run RPC server forever.
	go http.Serve(l, nil)

	// Add self contact
	hostname, port, _ = net.SplitHostPort(l.Addr().String())
	port_int, _ := strconv.Atoi(port)
	ipAddrStrings, err := net.LookupHost(hostname)
	var host net.IP
	for i := 0; i < len(ipAddrStrings); i++ {
		host = net.ParseIP(ipAddrStrings[i])
		if host.To4() != nil {
			break
		}
	}
	k.SelfContact = Contact{k.NodeID, host, uint16(port_int)}
	return k
}

//This is the only function that can have access to the hash table
//If other functions want to insert or get data from the hash table, they need to send a query through channels
//In this way, we provide a thread-safe structure to handle hash table
func (k *Kademlia) HandleHash() {
	for {
		cmd := <-k.hashchannel
		number := cmd.cmd
		switch number {
		case 1:
			k.hash[cmd.key] = cmd.value
		case 2:
			result, ok := k.hash[cmd.key]
			cmd.returnchannel <- hashreturn{result, ok}
		}
	}
}

//This is the only function that can have access to the routing table
//If other functions want to insert or get data from the routing table, they need to send a query through channels
//In this way, we provide a thread-safe structure to handle routing table
func (k *Kademlia) HandleTable() {
	Hashforfind := make(map[ID]chan findresult)
	Hashforcontact := make(map[ID]chan contactresult)
	for {
		select {
		//register for new client
		case client := <-k.registerchannel:
			number := client.num
			switch number {
			case 1:
				Hashforfind[client.id] = client.findchan //store the channel for sending result back
			case 2:
				Hashforcontact[client.id] = client.contactchan //store the channel for sending result back
			}
		case cmd := <-k.updatechannel:
			//update the routingtable
			k.table.UpDate(k, cmd.contact)
		case cmd := <-k.findchannel:
			number := cmd.num
			switch number {
			case 1:
				//find nodes
				nodes := k.table.FindCloset(cmd.key)
				result := findresult{nodes, nil, nil}
				Hashforfind[cmd.clientid] <- result
				delete(Hashforfind, cmd.clientid)
			case 2:
				//find value
				value, _ := k.LocalFindValue(cmd.key)
				nodes := k.table.FindCloset(cmd.key)
				result := findresult{nodes, value, nil}
				Hashforfind[cmd.clientid] <- result
				delete(Hashforfind, cmd.clientid)
			case 3:
				//find specific contact
				node := k.table.FindContact(cmd.key)
				Hashforcontact[cmd.clientid] <- contactresult{node}
				delete(Hashforcontact, cmd.clientid)
			case 4:
				//find alpha contact
				nodes := k.table.FindAlpha(cmd.key)
				result := findresult{nodes, nil, nil}
				Hashforfind[cmd.clientid] <- result
				delete(Hashforfind, cmd.clientid)
			}
		}
	}
}

func NewKademlia(laddr string) *Kademlia {
	return NewKademliaWithId(laddr, NewRandomID())
}

type ContactNotFoundError struct {
	id  ID
	msg string
}

func (e *ContactNotFoundError) Error() string {
	return fmt.Sprintf("%x %s", e.id, e.msg)
}

func (k *Kademlia) FindContact(nodeId ID) (*Contact, error) {
	// Search through contacts, find specified ID
	// Find contact with provided ID
	if nodeId == k.SelfContact.NodeID {
		return &k.SelfContact, nil
	}
	clientid := NewRandomID() //clientid is the key for register a new thread
	client := Client{findchan: make(chan findresult), contactchan: make(chan contactresult), id: clientid, num: 2}
	k.registerchannel <- client                       //register through channel
	k.findchannel <- findcommand{clientid, nodeId, 3} //send query through channel
	result := <-client.contactchan                    //get result from channel
	if result.node != nil {
		return result.node, nil
	}
	return nil, &ContactNotFoundError{nodeId, "Not found"}
}

type CommandFailed struct {
	msg string
}

func (e *CommandFailed) Error() string {
	return fmt.Sprintf("%s", e.msg)
}

func (k *Kademlia) DoPing(host net.IP, port uint16) (*Contact, error) {
	ping := PingMessage{k.SelfContact, NewRandomID()}
	var pong PongMessage
	address := host.String() + ":" + strconv.Itoa(int(port))
	path := rpc.DefaultRPCPath + strconv.Itoa(int(port))
	client, err := rpc.DialHTTPPath("tcp", address, path)
	if err != nil {
		return nil, &CommandFailed{"HTTP Connect Error"}
	}
	/*
		Use channel to decide time out
	*/
	errorChannel := make(chan error, 1)
	go func() {
		errorChannel <- client.Call("KademliaRPC.Ping", ping, &pong)
	}()
	select {
	case err := <-errorChannel:
		if err != nil {
			log.Fatal("CallDoPing:", err)
			return nil, err
		}
	case <-time.After(10 * time.Second):
		return nil, &CommandFailed{"Time Out"}
	}
	log.Printf("ping msgID:%s\n", ping.MsgID.AsString())
	log.Printf("pong msgID:%s\n", pong.MsgID.AsString())
	if !ping.MsgID.Equals(pong.MsgID) {
		return nil, &CommandFailed{"Wrong MsgID"}
	}
	result := pong.Sender
	k.updatechannel <- updatecommand{result}
	return &result, nil
}

func (k *Kademlia) DoStore(contact *Contact, key ID, value []byte) error {
	// TODO: Implement
	request := StoreRequest{k.SelfContact, NewRandomID(), key, value}
	var result StoreResult
	host := contact.Host.String()
	port := strconv.Itoa(int(contact.Port))
	address := host + ":" + port
	path := rpc.DefaultRPCPath + port
	client, err := rpc.DialHTTPPath("tcp", address, path)
	if err != nil {
		return &CommandFailed{"HTTP Connect Error"}
	}
	errorChannel := make(chan error, 1)
	go func() {
		errorChannel <- client.Call("KademliaRPC.Store", request, &result)
	}()
	select {
	case err := <-errorChannel:
		if err != nil {
			log.Fatal("CallDoStore:", err)
			return err
		}
	case <-time.After(10 * time.Second):
		return &CommandFailed{"Time Out"}
	}
	if !request.MsgID.Equals(result.MsgID) {
		return &CommandFailed{"Wrong MsgID"}
	}
	k.updatechannel <- updatecommand{*contact}
	return err
}

func (k *Kademlia) DoFindNode(contact *Contact, searchKey ID) ([]Contact, error) {
	request := FindNodeRequest{k.SelfContact, NewRandomID(), searchKey}
	var result FindNodeResult
	host := contact.Host.String()
	port := strconv.Itoa(int(contact.Port))
	address := host + ":" + port
	path := rpc.DefaultRPCPath + port
	client, err := rpc.DialHTTPPath("tcp", address, path)
	if err != nil {
		return nil, &CommandFailed{"HTTP Connect Error"}
	}
	errorChannel := make(chan error, 1)
	go func() {
		errorChannel <- client.Call("KademliaRPC.FindNode", request, &result)
	}()
	select {
	case err := <-errorChannel:
		if err != nil {
			log.Fatal("Call DoFindNode Error: ", err)
			return nil, err
		}
	case <-time.After(10 * time.Second):
		return nil, &CommandFailed{"Time Out"}
	}
	if !request.MsgID.Equals(result.MsgID) {
		return nil, &CommandFailed{"Wrong MsgID"}
	}
	if result.Err == nil {
		k.updatechannel <- updatecommand{*contact}
		for _, node := range result.Nodes {
			k.updatechannel <- updatecommand{node}
		}
		return result.Nodes, result.Err
	}
	return nil, result.Err
}

func (k *Kademlia) DoFindValue(contact *Contact,
	searchKey ID) (value []byte, contacts []Contact, err error) {

	request := FindValueRequest{k.SelfContact, NewRandomID(), searchKey}
	var result FindValueResult
	host := contact.Host.String()
	port := strconv.Itoa(int(contact.Port))
	address := host + ":" + port
	path := rpc.DefaultRPCPath + port
	client, err := rpc.DialHTTPPath("tcp", address, path)

	if err != nil {
		return nil, nil, &CommandFailed{"HTTP Connect Error"}
	}

	errorChannel := make(chan error, 1)
	go func() {
		errorChannel <- client.Call("KademliaRPC.FindValue", request, &result)
	}()
	select {
	case err := <-errorChannel:
		if err != nil {
			log.Fatal("Call DoFindValue:", err)
			return nil, nil, err
		}
	case <-time.After(10 * time.Second):
		return nil, nil, &CommandFailed{"Time Out"}
	}

	if !request.MsgID.Equals(result.MsgID) {
		return nil, nil, &CommandFailed{"Wrong MsgID"}
	}

	if result.Err == nil {
		k.updatechannel <- updatecommand{*contact}
		for _, element := range result.Nodes {
			k.updatechannel <- updatecommand{element}
		}
		return result.Value, result.Nodes, nil
	}
	return nil, nil, result.Err
}

func (k *Kademlia) LocalFindValue(searchKey ID) ([]byte, error) {
	req := hashcommand{searchKey, 2, nil, make(chan hashreturn)}
	k.hashchannel <- req
	result := <-req.returnchannel
	if result.ok {
		return result.value, nil
	}
	return []byte(""), &CommandFailed{"No such element"}
}

type iterativeResult struct {
	success    bool
	target     Contact
	activeList []Contact
	value      []byte
	err        error
}

func (k *Kademlia) doFind(contact Contact, key ID, findValue bool, resp chan iterativeResult) {
	if !findValue {
		contacts, err := k.DoFindNode(&contact, key)
		if err != nil {
			resp <- iterativeResult{false, contact, nil, nil, err}
		} else {
			resp <- iterativeResult{true, contact, contacts, nil, nil}
		}
	} else {
		value, contacts, err := k.DoFindValue(&contact, key)
		if err != nil {
			resp <- iterativeResult{false, contact, nil, nil, err}
		} else {
			resp <- iterativeResult{true, contact, contacts, value, nil}
		}
	}
}

type heapRequest struct {
	cmd     int
	channel chan heapResult
	contact []Contact
}

type heapResult struct {
	length   int
	contacts []Contact
}

func (k *Kademlia) HandleHeap(key ID, Req chan heapRequest) {
	pq := &PriorityQueue{k.SelfContact, []Contact{}, key}
	heap.Init(pq)
	for {
		request := <-Req
		cmd := request.cmd
		switch cmd {
		case 1:
			request.channel <- heapResult{pq.Len(), nil}
		case 2:
			heap.Push(pq, request.contact)
		case 3:
			con := []Contact{}
			length := pq.Len()
			for i := 0; i < alpha && pq.Len() > 0; i++ {
				con = append(con, heap.Pop(pq).(Contact))
			}
			request.channel <- heapResult{length, con}
			/*
				case 4:
					ok, ClosetNode := pq.Peek()
					if ok {
						request.channel <- heapResult{pq.Len(), []Contact{ClosetNode}}
					} else {
						request.channel <- heapResult{pq.Len(), nil}
					}
			*/
		}
	}
}

func (k *Kademlia) Iterative(key ID, findValue bool) iterativeResult {
	ret := iterativeResult{false, k.SelfContact, nil, nil, nil}
	activeNodes := &PriorityQueue{k.SelfContact, []Contact{}, key}
	nodeSet := make(map[string]bool)
	respChannel := make(chan iterativeResult)
	heapReq := make(chan heapRequest)

	heap.Init(activeNodes)

	go k.HandleHeap(key, heapReq)

	clientid := NewRandomID()
	client := Client{findchan: make(chan findresult), contactchan: make(chan contactresult), id: clientid, num: 1}
	k.registerchannel <- client
	k.findchannel <- findcommand{clientid, key, 4}
	result := <-client.findchan

	heapReq <- heapRequest{2, make(chan heapResult), result.Nodes}

	newReq := heapRequest{1, make(chan heapResult), nil}
	heapReq <- newReq
	heap_result := <-newReq.channel

	if heap_result.length <= 0 {
		ret.err = &CommandFailed{"No node in kBucket"}
		return ret
	}
	interval := 300
	t := time.NewTicker(time.Duration(interval) * time.Millisecond)
	quit := make(chan bool)
	signal := false
	go func() {
		for !signal {
			select {
			case <-quit:
				signal = <-quit
				return
			case <-t.C:
				req := heapRequest{3, make(chan heapResult), nil}
				heapReq <- req
				PopResult := <-req.channel
				for i := 0; i < alpha && i < PopResult.length; i++ {
					contact := PopResult.contacts[i]
					go k.doFind(contact, key, findValue, respChannel)
				}
			}
		}
	}()
	//logic for iteration
	for activeNodes.Len() < 20 {
		result := <-respChannel
		if result.success {
			flag := true
			ok, cnode := activeNodes.Last()
			if ok {
				flag = false
				for _, node := range result.activeList {
					if Closer(key, node, cnode) {
						flag = true
					}
				}
			}
			if findValue && result.value != nil {
				ret.value = result.value
				break
			}
			NeedToPush := []Contact{}
			for _, node := range result.activeList {
				if ok, _ := nodeSet[node.NodeID.AsString()]; !ok {
					NeedToPush = append(NeedToPush, node)
					nodeSet[node.NodeID.AsString()] = true
				}
			}
			if !flag {
				break
			}
			req := heapRequest{2, make(chan heapResult), NeedToPush}
			heapReq <- req
			heap.Push(activeNodes, result.target)
		}
	}
	if findValue && ret.value != nil {
		for activeNodes.Len() > 0 {
			ret.activeList = append(ret.activeList, heap.Pop(activeNodes).(Contact))
		}
		return ret
	}
	//TODO logic for iteration
	for activeNodes.Len() < 20 {
		result := <-respChannel
		if result.success {
			if findValue && result.value != nil {
				ret.value = result.value
				break
			}
			NeedToPush := []Contact{}
			for _, node := range result.activeList {
				if ok, _ := nodeSet[node.NodeID.AsString()]; !ok {
					NeedToPush = append(NeedToPush, node)
					nodeSet[node.NodeID.AsString()] = true
				}
			}
			req := heapRequest{2, make(chan heapResult), NeedToPush}
			heapReq <- req
			heap.Push(activeNodes, result.target)
		}
	}
	quit <- true
	close(respChannel)
	ret.success = true
	return ret
}

func ClosetNode(key ID, c1, c2 Contact) Contact {
	dist1 := key.Xor(c1.NodeID)
	dist2 := key.Xor(c2.NodeID)
	if dist1.Less(dist2) {
		return c1
	}
	return c2
}

func Closer(key ID, c1, c2 Contact) bool {
	dist1 := key.Xor(c1.NodeID)
	dist2 := key.Xor(c2.NodeID)
	return dist1.Less(dist2)
}

// For project 2!
func (k *Kademlia) DoIterativeFindNode(id ID) ([]Contact, error) {
	result := k.Iterative(id, false)
	return nil, result.err
	if result.success {
		return result.activeList, nil
	}
	return nil, &CommandFailed{"Fail Iterative"}
}
func (k *Kademlia) DoIterativeStore(key ID, value []byte) ([]Contact, error) {
	result := k.Iterative(key, false)
	if result.success {
		for _, node := range result.activeList {
			k.DoStore(&node, key, value)
		}
		return result.activeList, nil
	}
	return nil, &CommandFailed{"Fail Iterative"}
}
func (k *Kademlia) DoIterativeFindValue(key ID) (value []byte, err error) {
	result := k.Iterative(key, true)
	if result.success {
		if result.value != nil {
			return result.value, nil
		} else {
			return nil, &CommandFailed{"Cannot find value"}
		}
	}
	return nil, &CommandFailed{"Fail Iterative"}
}

// For project 3!
func (k *Kademlia) Vanish(data []byte, numberKeys byte,
	threshold byte, timeoutSeconds int) (vdo VanashingDataObject) {
	return
}

func (k *Kademlia) Unvanish(searchKey ID) (data []byte) {
	return nil
}
