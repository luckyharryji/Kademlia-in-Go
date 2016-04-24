package libkademlia

// Contains the core kademlia type. In addition to core state, this type serves
// as a receiver for the RPC methods, which is required by that package.

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
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
	updatechannel   chan updatecommand
	findchannel     chan findcommand
	registerchannel chan Client
	table           *RoutingTable
}

type updatecommand struct {
	contact Contact
}

type findcommand struct {
	clientid ID
	key      ID
	num      int
}

type findresult struct {
	Nodes []Contact
	Value []byte
	err   error
}

type contactresult struct {
	node *Contact
}

type Client struct {
	findchan    chan findresult
	contactchan chan contactresult
	id          ID
	num         int
}

func NewKademliaWithId(laddr string, nodeID ID) *Kademlia {
	k := new(Kademlia)
	k.NodeID = nodeID
	k.table = NewRoutingTable(nodeID)
	k.updatechannel = make(chan updatecommand)
	k.findchannel = make(chan findcommand)
	k.registerchannel = make(chan Client)
	k.hash = make(map[ID][]byte)
	go k.HandleTable()
	// TODO: Initialize other state here as you add functionality.
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
				Hashforfind[client.id] = client.findchan
			case 2:
				Hashforcontact[client.id] = client.contactchan
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
			case 2:
				//find value
				value, _ := k.LocalFindValue(cmd.key)
				nodes := k.table.FindCloset(cmd.key)
				result := findresult{nodes, value, nil}
				Hashforfind[cmd.clientid] <- result
			case 3:
				//find specific contact
				node := k.table.FindContact(cmd.key)
				Hashforcontact[cmd.clientid] <- contactresult{node}
			}
		default:
			continue
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
	// TODO: Search through contacts, find specified ID
	// Find contact with provided ID
	if nodeId == k.SelfContact.NodeID {
		return &k.SelfContact, nil
	}
	clientid := NewRandomID()
	client := Client{findchan: make(chan findresult), contactchan: make(chan contactresult), id: clientid, num: 2}
	k.registerchannel <- client
	k.findchannel <- findcommand{clientid, nodeId, 3}
	result := <-client.contactchan
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
	err = client.Call("KademliaRPC.Ping", ping, &pong)
	if err != nil {
		log.Fatal("CallDoPing:", err)
	}
	log.Printf("ping msgID:%s\n", ping.MsgID.AsString())
	log.Printf("pong msgID:%s\n", pong.MsgID.AsString())
	if err != nil {
		return nil, err
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
	err = client.Call("KademliaRPC.Store", request, &result)
	if err != nil {
		log.Fatal("CallDoStore:", err)
	}
	k.updatechannel <- updatecommand{*contact}
	return err
}

func (k *Kademlia) DoFindNode(contact *Contact, searchKey ID) ([]Contact, error) {
	// TODO: Implement
	request := FindNodeRequest{k.SelfContact, NewRandomID(), searchKey}
	var result FindNodeResult
	host := contact.Host.String()
	port := strconv.Itoa(int(contact.Port))
	address := host + ":" + port
	path := rpc.DefaultRPCPath + port
	client, err := rpc.DialHTTPPath("tcp", address, path)
	err = client.Call("KademliaRPC.FindNode", request, &result)
	if err != nil {
		log.Fatal("Call DoFindNode", err)
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
	// TODO: Implement
	request := FindValueRequest{k.SelfContact, NewRandomID(), searchKey}
	var result FindValueResult
	host := contact.Host.String()
	port := strconv.Itoa(int(contact.Port))
	address := host + ":" + port
	path := rpc.DefaultRPCPath + port
	client, err := rpc.DialHTTPPath("tcp", address, path)
	err = client.Call("KademliaRPC.FindValue", request, &result)
	if err != nil {
		log.Fatal("Call DoFindValue:", err)
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
	// TODO: Implement
	result, ok := k.hash[searchKey]
	if ok {
		return result, nil
	}
	return []byte(""), &CommandFailed{"No such element"}
}

// For project 2!
func (k *Kademlia) DoIterativeFindNode(id ID) ([]Contact, error) {
	return nil, &CommandFailed{"Not implemented"}
}
func (k *Kademlia) DoIterativeStore(key ID, value []byte) ([]Contact, error) {
	return nil, &CommandFailed{"Not implemented"}
}
func (k *Kademlia) DoIterativeFindValue(key ID) (value []byte, err error) {
	return nil, &CommandFailed{"Not implemented"}
}

// For project 3!
func (k *Kademlia) Vanish(data []byte, numberKeys byte,
	threshold byte, timeoutSeconds int) (vdo VanashingDataObject) {
	return
}

func (k *Kademlia) Unvanish(searchKey ID) (data []byte) {
	return nil
}
