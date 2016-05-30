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

	// Xiangyu: for vanish
	VDOStoreChannel chan VDO_Obj_For_Command
	VDOStorage		map[ID]VanashingDataObject

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

type VDO_Obj_For_Command struct {
	VDO_id	 ID
	VDO_Obj	 VanashingDataObject
	Cmd_type int
	Query_result_channel chan VanashingDataObject
	errors	 chan error
	// Time_out  time.Second
	// xiangyu: Remaining issue
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

	// xiangyu: store for VDO
	k.VDOStoreChannel = make(chan VDO_Obj_For_Command)
	go k.HandleVDOStore()

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

// Use Channel to store VDO locally
func (k *Kademlia) HandleVDOStore() {
	for obj := range k.VDOStoreChannel {
		switch obj.Cmd_type{
		case 1:
			k.VDOStorage[obj.VDO_id] = obj.VDO_Obj
		case -1:
			delete(k.VDOStorage, obj.VDO_id) // xiangyu: what if does not existst
		case 0:
			if vdo_obj, ok := k.VDOStorage[obj.VDO_id]; ok {
				obj.Query_result_channel <- vdo_obj
			} else {
				obj.errors <- &CommandFailed{"VDO ID does not exists"}
			}
			// fmt.Println("Return here")
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
				value, err := k.LocalFindValue(cmd.key)
				nodes := k.table.FindCloset(cmd.key)
				result := findresult{nodes, value, err}
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
	defer close(client.contactchan)
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
	defer client.Close()
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
	if !ping.MsgID.Equals(pong.MsgID) {
		return nil, &CommandFailed{"Wrong MsgID"}
	}
	result := pong.Sender
	k.updatechannel <- updatecommand{result}
	return &result, nil
}

/*
InternalPing: for ping the LRU contact to decide whether replace it
ping without send update query
*/
func (k *Kademlia) InternalPing(host net.IP, port uint16) (*Contact, error) {
	ping := PingMessage{k.SelfContact, NewRandomID()}
	var pong PongMessage
	address := host.String() + ":" + strconv.Itoa(int(port))
	path := rpc.DefaultRPCPath + strconv.Itoa(int(port))
	client, err := rpc.DialHTTPPath("tcp", address, path)
	defer client.Close()
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
	if !ping.MsgID.Equals(pong.MsgID) {
		return nil, &CommandFailed{"Wrong MsgID"}
	}
	result := pong.Sender
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
	defer client.Close()
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
	defer client.Close()
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
		return result.Nodes, nil
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
	defer client.Close()
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

/*
If it is an IterativeFindNode or IterativeStore we set findValue to be false.
If it is an IterativeFindValue we set findValue to be true.
If findValue is false then we do DoFindNode, if findValue is true then we DoFindValue.
*/

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
			if value != nil {
				resp <- iterativeResult{true, contact, contacts, value, nil}
			} else {
				resp <- iterativeResult{true, contact, contacts, nil, nil}
			}
		}
	}
}

type heapRequest struct {
	cmd      int
	len      int
	channel  chan heapResult
	contacts []Contact
}

type heapResult struct {
	length   int
	contacts []Contact
}

func (k *Kademlia) HandleHeap(key ID, Req chan heapRequest) {
	/*
		Handle The Heap List created for return shortlist:
		Deal with individual Goroutine

		Input:
			@key: the object nodeID which tends to find
			@Req(chan): record the channel used for update heap
	*/
	pq := &PriorityQueue{k.SelfContact, []Contact{}, key}
	heap.Init(pq)
	nodeSet := make(map[string]bool)
	for {
		request := <-Req
		cmd := request.cmd
		switch cmd {
		case 1:
			// request for length
			request.channel <- heapResult{pq.Len(), nil}
		case 2:
			// request to push new element into heap
			for _, node := range request.contacts {
				if _, ok := nodeSet[node.NodeID.AsString()]; !ok {
					// ignore when self ID is returned
					if !node.NodeID.Equals(k.NodeID) {
						heap.Push(pq, node)
						nodeSet[node.NodeID.AsString()] = true
					}
				}
			}
			request.channel <- heapResult{pq.Len(), nil}
		case 3:
			// Pop the close element from the heap
			con := []Contact{}
			length := pq.Len()
			for i := 0; i < request.len && pq.Len() > 0; i++ {
				con = append(con, heap.Pop(pq).(Contact))
			}
			request.channel <- heapResult{length, con}
		}
	}
}

func (k *Kademlia) Iterative(key ID, findValue bool) iterativeResult {
	ret := iterativeResult{false, k.SelfContact, nil, nil, nil}
	activeNodes := &PriorityQueue{k.SelfContact, []Contact{}, key}
	respChannel := make(chan iterativeResult)
	heapReq := make(chan heapRequest)

	heap.Init(activeNodes)

	go k.HandleHeap(key, heapReq)

	clientid := NewRandomID()
	client := Client{findchan: make(chan findresult), contactchan: make(chan contactresult), id: clientid, num: 1}
	k.registerchannel <- client
	k.findchannel <- findcommand{clientid, key, 4}
	result := <-client.findchan

	// add the initial 20 nodes into potential heap
	pushReq := heapRequest{2, 0, make(chan heapResult), result.Nodes}
	heapReq <- pushReq

	<-pushReq.channel

	newReq := heapRequest{1, 0, make(chan heapResult), nil}
	heapReq <- newReq
	heap_result := <-newReq.channel

	if heap_result.length <= 0 {
		ret.err = &CommandFailed{"No node in kBucket"}
		return ret
	}
	//logic for iteration
outerloop:
	for activeNodes.Len() < 20 {
		count := 0
		req := heapRequest{3, 3, make(chan heapResult), nil}
		heapReq <- req
		PopResult := <-req.channel
		for _, node := range PopResult.contacts {
			go k.doFind(node, key, findValue, respChannel)
			count++
		}
	innerloop:
		for {
			select {
			case result := <-respChannel:
				count--
				if result.success {
					if findValue && result.value != nil {
						ret.value = result.value
						ret.success = true
						_, ret.target = activeNodes.Peek() //Find the closet node which doesn't return the value
						ret.activeList = nil
						return ret
						//If it is IterativeFindValue and we successfully find the value.
					}
					flag := true
					ok, cnode := activeNodes.Peek()
					//Get the closet node in the shortlist
					heap.Push(activeNodes, result.target)
					if ok {
						flag = false
						// decide if one of the return node is closer to the object node
						for _, node := range result.activeList {
							if Closer(key, node, cnode) {
								flag = true
							}
						}
					}
					if !flag {
						break outerloop
					}
					req := heapRequest{2, 0, make(chan heapResult), result.activeList}
					heapReq <- req
					<-req.channel
					//Push the return result into potential heap
				}

				if count == 0 {
					break innerloop
				}
			case <-time.After(300 * time.Millisecond):
				fmt.Println("Time out")
				break innerloop
			}
		}
	}
	//If the closet node is not updated:
OuterLoop:
	for activeNodes.Len() < 20 {
		length := 20 - activeNodes.Len()
		count := 0
		req := heapRequest{3, length, make(chan heapResult), nil}
		heapReq <- req
		PopResult := <-req.channel
		for _, node := range PopResult.contacts {
			go k.doFind(node, key, findValue, respChannel)
			count++
		}
	InnerLoop:
		for {
			result := <-respChannel
			if result.success {
				if findValue && result.value != nil {
					ret.value = result.value
					ret.activeList = nil
					ret.success = true
					_, ret.target = activeNodes.Peek() //Find the closet node which does not return the value
					return ret
					//If it is an IterativeFindValue and we find the value
				}
				heap.Push(activeNodes, result.target)
			}
			count--
			if count == 0 {
				break InnerLoop
			}
			requestForLengthReq := heapRequest{1, 0, make(chan heapResult), nil}
			heapReq <- requestForLengthReq
			length_result := <-requestForLengthReq.channel
			if length_result.length <= 0 {
				break OuterLoop
			}
		}
	}
	if findValue && ret.value == nil {
		ret.activeList = nil
		ret.success = true
		_, ret.target = activeNodes.Peek()
		return ret
	}
	for activeNodes.Len() > 0 {
		ret.activeList = append(ret.activeList, heap.Pop(activeNodes).(Contact))
	}
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
	if result.success {
		return result.activeList, nil
	}
	return nil, result.err
}
func (k *Kademlia) DoIterativeStore(key ID, value []byte) ([]Contact, error) {
	result := k.Iterative(key, false)
	if result.success {
		for _, node := range result.activeList {
			k.DoStore(&node, key, value)
		}
		return result.activeList, nil
	}
	return nil, result.err
}
func (k *Kademlia) DoIterativeFindValue(key ID) (value []byte, err error) {
	result := k.Iterative(key, true)
	if result.success {
		if result.value != nil {
			k.DoStore(&result.target, key, result.value) //Store the value to the closet node which doesn't return the value
			//fmt.Println(key.Xor(result.target.NodeID).PrefixLen())
			//fmt.Println(key.Xor(key).PrefixLen())
			fmt.Println("Store value to the closet node :" + result.target.NodeID.AsString())
			return result.value, nil
		} else {
			return nil, &CommandFailed{"Cannot find value! Closet Node :" + result.target.NodeID.AsString()}
		}
	}
	return nil, result.err
}

// For project 3!
func (k *Kademlia) StoreVdoObj(VdoID ID, vdo VanashingDataObject){
	store_req := VDO_Obj_For_Command {
		VDO_id: VdoID,
		VDO_Obj: vdo,
		Cmd_type: 1,
		Query_result_channel: nil,
		errors: nil,
	}
	k.VDOStoreChannel <- store_req
}

func (k *Kademlia) DoFindVdoFromSingleContact(contact Contact, vdoID ID) (vdo *VanashingDataObject) {
	find_request_id := NewRandomID()
	get_vdo_request := GetVDORequest {
		Sender: k.SelfContact,
		VdoID: vdoID,
		MsgID: find_request_id,
	}
	get_vdo_result := new(GetVDOResult)
	host := contact.Host.String()
	port := strconv.Itoa(int(contact.Port))
	address := host + ":" + port
	path := rpc.DefaultRPCPath + port
	client, err := rpc.DialHTTPPath("tcp", address, path)
	defer client.Close()
	if err != nil {
		// Connection Error
		return
	}
	errorChannel := make(chan error, 1)
	go func() {
		errorChannel <- client.Call("KademliaRPC.GetVDO", get_vdo_request, get_vdo_result)
	}()
	select {
	case err := <-errorChannel:
		if err != nil {
			return
		}
	case <-time.After(10 * time.Second):
		return
	}
	update := updatecommand{contact}
	k.updatechannel <- update
	if get_vdo_result.Error != nil{
		return nil
	}
	return &get_vdo_result.VDO
}

func (k *Kademlia) FindVdoFromContactList(contact_list []Contact, vdoID ID) (vdo VanashingDataObject){
	for _, node_obj :=  range contact_list {
		vdo_from_node := k.DoFindVdoFromSingleContact(node_obj, vdoID)
		if vdo_from_node != nil{
			return *vdo_from_node
		}
	}
	return
}

func (k *Kademlia) LocalFindVdo(nodeID ID, vdoID ID) (vdo *VanashingDataObject) {
	local_node, err := k.FindContact(nodeID)
	if err != nil {
		return nil
	}
	local_find_vdo := k.DoFindVdoFromSingleContact(*local_node, vdoID)
	return local_find_vdo
}

func (k *Kademlia) IterativeFindVdo(nodeID ID, vdoID ID) (vdo *VanashingDataObject) {
	nodes_candidates_with_vdo, err := k.DoIterativeFindNode(nodeID)
	if err != nil{
		// error when finding
		return nil
	}
	if nodes_candidates_with_vdo == nil || len(nodes_candidates_with_vdo) == 0 {
		// No node is found
		return nil
	}
	// Xiangyu denote:
	// Based on Piazza discussion, only call RPC once if the result of
	// iterative find node have the same contact node id with the input
	for _, node_obj := range nodes_candidates_with_vdo {
		if node_obj.NodeID.Equals(nodeID){
			vdo_from_iterative_node := k.LocalFindVdo(node_obj.NodeID, vdoID)
			if vdo_from_iterative_node != nil {
				return vdo_from_iterative_node
			}
		}
	}
	return nil
	// vdo_from_list := k.FindVdoFromContactList(nodes_candidates_with_vdo, vdoID)
	// return &vdo_from_list
}

func (k *Kademlia) Republish(timeoutSeconds int, vdo_obj VanashingDataObject, vdoID ID) {
	epoch := 1
	ForLoop:
		for {
			select{
			case <-time.After(time.Duration(timeoutSeconds) * time.Second):
				break ForLoop
			case <-time.After(time.Duration(400 * epoch) * time.Minute):
				data := k.UnvanishData(vdo_obj)
				vdo_refresh := k.VanishDataWithRepublish(data, vdo_obj.NumberKeys, vdo_obj.Threshold, epoch)
				k.StoreVdoObj(vdoID, vdo_refresh)
				epoch += 1
			}
		}
}

func (k *Kademlia) Vanish(vdoID ID, data []byte, numberKeys byte, threshold byte, timeoutSeconds int) (vdo VanashingDataObject) {
	// Xiangyu: remaining handling timeout
	vdo_after_vanish := k.VanishData(data, numberKeys, threshold, timeoutSeconds)
	k.StoreVdoObj(vdoID, vdo_after_vanish)
	go k.Republish(timeoutSeconds, vdo_after_vanish, vdoID)
	return vdo_after_vanish
}

func (k *Kademlia) Unvanish(nodeID ID, vdoID ID)(data []byte) {
	var vdo_obj VanashingDataObject
	local_vdo := k.LocalFindVdo(nodeID, vdoID)
	if local_vdo != nil {
		vdo_obj = *local_vdo
	} else {
		remote_vdo := k.IterativeFindVdo(nodeID, vdoID)
		if remote_vdo != nil {
			vdo_obj = *remote_vdo
		}
	}
	if &vdo_obj == nil {
		return nil
	}
	if vdo_obj.NumberKeys == byte(0) {
		return nil
	} else {
		return k.UnvanishData(vdo_obj)
	}
}
