package libkademlia

import (
	"container/list"
	"sort"
)

/*
By defination in libkademlia
const (
	alpha = 3
	b     = 8 * IDBytes
	k     = 20
)
*/

type RoutingTable struct {
	id ID
	Buckets [b] * BucketList
}


type Contacts []ContactForSort

func NewRoutingTable() (table *RoutingTable) {
	table = new(RoutingTable)
	for i := 0; i < 8 * IDBytes; i++ {
		table.BucketLists[i] = &BucketList{length: 0}
	}
	return
}

func (table *RoutingTable) FindContact(nodeId ID) (*Contact, error) {
	prefixlen := nodeId.Xor(table.id).PrefixLen()
	locationList := table.Buckets[prefixlen]
	nodeOfContact := locationList.Find(nodeId)
	if nodeOfContact != nil{
		return &nodeOfContact.contact, nil
	} else {
		return nil, &ContactNotFoundError{nodeId, "Not found"}
	}
}
