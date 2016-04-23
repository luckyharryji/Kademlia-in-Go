package libkademlia

// import (
// 	"container/list"
// 	"sort"
// )

/*
This lib tend to implement the RoutingTable structure for each server.

ToDo: the table should be initialize with the kademlia object
*/

/*
By defination in libkademlia
const (
	alpha = 3
	b     = 8 * IDBytes
	k     = 20
)
*/

type RoutingTable struct {
	Buckets [b] * BucketList
}

func NewRoutingTable() (table *RoutingTable) {
	table = new(RoutingTable)
	for i := 0; i < 8 * IDBytes; i++ {
		table.Buckets[i] = &BucketList{length: 0}
	}
	return
}

func (table *RoutingTable) FindListWithId(selfId, nodeId ID) *BucketList {
	prefixlen := selfId.Xor(nodeId).PrefixLen()
	return table.Buckets[prefixlen]
}

func (table *RoutingTable) FindNodeWithId(selfId, nodeId ID) *Contact {
	nodeOfContact := table.FindListWithId(selfId, nodeId).Find(nodeId)
	if nodeOfContact != nil {
		return &nodeOfContact.contact
	} else {
		return nil
	}
}

func (table *RoutingTable) UpdateFront(selfId ID, contact Contact) error {
	table.FindListWithId(selfId, contact.NodeID).DeleteFrontInsert(contact)
	return nil
}

func (table *RoutingTable) RecordContact(selfId ID, contact Contact) *Contact {
	locationList := table.FindListWithId(selfId, contact.NodeID)
	nodeOfContact := locationList.Find(contact.NodeID)
	if nodeOfContact != nil {
		previousNode := nodeOfContact.prev
		afterNode := nodeOfContact.next
		previousNode.next = afterNode
		afterNode.prev = previousNode
		locationList.Push(contact)
		return nil
	} else if locationList.length < k {
		locationList.Push(contact)
		return nil
	} else {
		// ToDo: ping the top of the list
		return &locationList.head.contact
	}
	return nil
}
