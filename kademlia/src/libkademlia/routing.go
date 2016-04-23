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
	id ID
	Buckets [b] * BucketList
}

func NewRoutingTable() (table *RoutingTable) {
	table = new(RoutingTable)
	for i := 0; i < 8 * IDBytes; i++ {
		table.Buckets[i] = &BucketList{length: 0}
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
