package libkademlia

// Contains definitions mirroring the Kademlia spec. You will need to stick
// strictly to these to be compatible with the reference implementation and
// other groups' code.

import (
	"net"
)

type KademliaRPC struct {
	kademlia *Kademlia
}

// Host identification.
type Contact struct {
	NodeID ID
	Host   net.IP
	Port   uint16
}

///////////////////////////////////////////////////////////////////////////////
// PING
///////////////////////////////////////////////////////////////////////////////
type PingMessage struct {
	Sender Contact
	MsgID  ID
}

type PongMessage struct {
	MsgID  ID
	Sender Contact
}

func (k *KademliaRPC) Ping(ping PingMessage, pong *PongMessage) error {
	pong.Sender = k.kademlia.SelfContact
	pong.MsgID = CopyID(ping.MsgID)
	update := updatecommand{ping.Sender}
	k.kademlia.updatechannel <- update
	return nil
}

///////////////////////////////////////////////////////////////////////////////
// STORE
///////////////////////////////////////////////////////////////////////////////
type StoreRequest struct {
	Sender Contact
	MsgID  ID
	Key    ID
	Value  []byte
}

type StoreResult struct {
	MsgID ID
	Err   error
}

func (k *KademliaRPC) Store(req StoreRequest, res *StoreResult) error {
	// TODO: Implement.
	k.kademlia.hash[req.Key] = req.Value
	res.MsgID = CopyID(req.MsgID)
	update := updatecommand{req.Sender}
	k.kademlia.updatechannel <- update
	return nil
}

///////////////////////////////////////////////////////////////////////////////
// FIND_NODE
///////////////////////////////////////////////////////////////////////////////
type FindNodeRequest struct {
	Sender Contact
	MsgID  ID
	NodeID ID
}

type FindNodeResult struct {
	MsgID ID
	Nodes []Contact
	Err   error
}

func (k *KademliaRPC) FindNode(req FindNodeRequest, res *FindNodeResult) error {
	res.MsgID = CopyID(req.MsgID)
	update := updatecommand{req.Sender}
	k.kademlia.updatechannel <- update
	clientid := NewRandomID()
	client := Client{findchan: make(chan findresult), contactchan: make(chan contactresult), id: clientid, num: 1}
	k.kademlia.registerchannel <- client
	k.kademlia.findchannel <- findcommand{clientid, req.NodeID, 1}
	result := <-client.findchan
	res.Nodes = result.Nodes
	return nil
}

///////////////////////////////////////////////////////////////////////////////
// FIND_VALUE
///////////////////////////////////////////////////////////////////////////////
type FindValueRequest struct {
	Sender Contact
	MsgID  ID
	Key    ID
}

// If Value is nil, it should be ignored, and Nodes means the same as in a
// FindNodeResult.
type FindValueResult struct {
	MsgID ID
	Value []byte
	Nodes []Contact
	Err   error
}

func (k *KademliaRPC) FindValue(req FindValueRequest, res *FindValueResult) error {
	res.MsgID = CopyID(req.MsgID)
	update := updatecommand{req.Sender}
	k.kademlia.updatechannel <- update
	clientid := NewRandomID()
	client := Client{findchan: make(chan findresult), contactchan: make(chan contactresult), id: clientid, num: 1}
	k.kademlia.registerchannel <- client
	k.kademlia.findchannel <- findcommand{clientid, req.Key, 2}
	result := <-client.findchan
	res.Value = result.Value
	res.Nodes = result.Nodes
	res.Err = nil
	return nil
}

// For Project 3

type GetVDORequest struct {
	Sender Contact
	VdoID  ID
	MsgID  ID
}

type GetVDOResult struct {
	MsgID ID
	VDO   VanashingDataObject
}

func (k *KademliaRPC) GetVDO(req GetVDORequest, res *GetVDOResult) error {
	// TODO: Implement.
	return nil
}
