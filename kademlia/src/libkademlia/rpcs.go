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
	k.kademlia.hashchannel <- hashcommand{req.Key, 1, req.Value, make(chan hashreturn)}
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
	defer close(client.findchan)
	res.Nodes = result.Nodes
	res.Err = result.err
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
	defer close(client.findchan)
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
	Error error
}

// RPC to handle request to get VDO obj
func (k *KademliaRPC) GetVDO(req GetVDORequest, res *GetVDOResult) error {
	res.MsgID = CopyID(req.MsgID)
	update := updatecommand{req.Sender}
	k.kademlia.updatechannel <- update
	id_of_vdo := req.VdoID
	result_channel := make(chan VanashingDataObject)
	empty_vdo_obj := new(VanashingDataObject)
	error_channel := make(chan error)
	find_req := VDO_Obj_For_Command {
		VDO_id: id_of_vdo,
		VDO_Obj: *empty_vdo_obj,
		Cmd_type: 0,
		Query_result_channel: result_channel,
		errors: error_channel,
	}
	k.kademlia.VDOStoreChannel <- find_req
	select {
	case find_result := <- result_channel:
		res.VDO = find_result
	case error_finding := <- error_channel:
		res.Error = error_finding
	}
	return nil
}
