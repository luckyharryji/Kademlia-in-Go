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

// var clientIdToChannel = make(map[string] chan string)

type LocalData map[ID][]byte
// Kademlia type. You can put whatever state you need in this.
type Kademlia struct {
	NodeID      ID
	SelfContact Contact
	RoutingTable *RoutingTable
	DataTable LocalData
}

func NewKademliaWithId(laddr string, nodeID ID) *Kademlia {
	k := new(Kademlia)
	k.NodeID = nodeID

	// TODO: Initialize other state here as you add functionality.

	// Set up RPC server
	// NOTE: KademliaRPC is just a wrapper around Kademlia. This type includes
	// the RPC functions.

	s := rpc.NewServer()
	s.Register(&KademliaRPC{k})
	hostname, port, err := net.SplitHostPort(laddr)
	if err != nil {
		return nil
	}
	s.HandleHTTP(rpc.DefaultRPCPath + port,
		rpc.DefaultDebugPath + port)
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

	// initialize the routing table for the Kademlia
	routingTable := NewRoutingTable()
	k.RoutingTable = routingTable
	// initilize local data storage hashMap
	localData := make(LocalData)
	k.DataTable = localData
	return k
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
	if nodeId == k.SelfContact.NodeID {
		return &k.SelfContact, nil
	}
	nodeOfContact := k.RoutingTable.FindNodeWithId(k.NodeID, nodeId)
	if nodeOfContact != nil{
		return nodeOfContact, nil
	} else {
		return nil, &ContactNotFoundError{nodeId, "Not found"}
	}
}

type CommandFailed struct {
	msg string
}

func (e *CommandFailed) Error() string {
	return fmt.Sprintf("%s", e.msg)
}

func (k *Kademlia) Update(contact Contact) {
	headContact := k.RoutingTable.RecordContact(k.NodeID, contact)
	if headContact != nil {
		pingMessage := PingMessage{Sender: k.SelfContact, MsgID: k.NodeID}
		var pongMessage PongMessage

		address := headContact.Host.String() + ":" + strconv.FormatInt(int64(headContact.Port), 10)
		path := rpc.DefaultRPCPath + strconv.Itoa(int(headContact.Port))
		client, err := rpc.DialHTTPPath("tcp", address, path)
		err = client.Call("KademliaRPC.Ping", pingMessage, &pongMessage)
		if err == nil {
			k.RoutingTable.UpdateFront(k.NodeID, contact)
		}
	}
}

func (k *Kademlia) DoPing(host net.IP, port uint16) (*Contact, error) {
	contact := k.SelfContact
	pingMessage := PingMessage{Sender: contact, MsgID: k.NodeID}
	var pongMessage PongMessage

	address := host.String() + ":" + strconv.FormatInt(int64(port), 10)
	path := rpc.DefaultRPCPath + strconv.Itoa(int(port))
	client, err := rpc.DialHTTPPath("tcp", address, path)
	err = client.Call("KademliaRPC.Ping", pingMessage, &pongMessage)
	if err != nil {
		log.Fatal("Call: ", err)
		return nil, &CommandFailed{
			"Unable to ping " + fmt.Sprintf("%s:%v", host.String(), port)}
	}
	k.Update(pongMessage.Sender)
	return nil, nil
}

func (k *Kademlia) DoStore(contact *Contact, key ID, value []byte) error {
	// TODO: Implement
	clientContact := k.SelfContact
	storePacket := StoreRequest{Sender: clientContact, MsgID: contact.NodeID, Key: key, Value: value}
	var response StoreResult

	address := contact.Host.String() + ":" + strconv.FormatInt(int64(contact.Port), 10)
	path := rpc.DefaultRPCPath + strconv.Itoa(int(contact.Port))
	client, err := rpc.DialHTTPPath("tcp", address, path)
	err = client.Call("KademliaRPC.Store", storePacket, &response)
	if err != nil {
		log.Fatal("Call: ", err)
		return &CommandFailed{"Unable to Store "}
	}
	return nil
}

func (k *Kademlia) DoFindNode(contact *Contact, searchKey ID) ([]Contact, error) {
	// TODO: Implement
	return nil, &CommandFailed{"Not implemented"}
}

func (k *Kademlia) DoFindValue(contact *Contact,
	searchKey ID) (value []byte, contacts []Contact, err error) {
	// TODO: Implement
	return nil, nil, &CommandFailed{"Not implemented"}
}

/*
Important !!!
rewrite to be channel !!
*/
func (k *Kademlia) LocalStoreValue(key ID, value []byte) error {
	k.DataTable[key] = value
	return nil
}

func (k *Kademlia) LocalFindValue(searchKey ID) ([]byte, error) {
	// TODO: Implement
	if data, exist := k.DataTable[searchKey]; exist{
		return data, nil
	} else {
		return []byte(""), &CommandFailed{"Not Found"}
	}
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
