package libkademlia

import ()

type RoutingTable struct {
	SelfId      ID
	BucketLists [8 * IDBytes]*BucketList
}

func NewRoutingTable(id ID) (table *RoutingTable) {
	table = new(RoutingTable)
	table.SelfId = id
	for i := 0; i < 8*IDBytes; i++ {
		table.BucketLists[i] = new(BucketList)
	}
	return
}

func (table *RoutingTable) UpDate(k *Kademlia, contact Contact) {
	id_table := table.SelfId
	id := contact.NodeID
	prefixlen := id.Xor(id_table).PrefixLen()
	bucket := table.BucketLists[prefixlen]
	element := bucket.Find(contact)
	if element == nil {
		if bucket.length <= 20 {
			bucket.Push(contact)
		} else {
			firstnode := bucket.First()
			_, err := k.DoPing(firstnode.contact.Host, firstnode.contact.Port)
			if err != nil {
				bucket.DeleteFrontInsert(contact)
			}
		}
	} else {
		bucket.MoveToTail(contact)
	}
}

func (table *RoutingTable) FindContact(nodeid ID) *Contact {
	prefixlen := nodeid.Xor(table.SelfId).PrefixLen()
	bucket := table.BucketLists[prefixlen]
	for n := bucket.First(); n != nil; n = n.Next() {
		if n.contact.NodeID.Equals(nodeid) {
			return &n.contact
		}
	}
	return nil
}

func (table *RoutingTable) FindCloset(nodeid ID) []Contact {
	prefixlen := nodeid.Xor(table.SelfId).PrefixLen()
	bucket := table.BucketLists[prefixlen]
	result := make([]Contact, 0)
	count := 0
	for n := bucket.First(); n != nil; n = n.Next() {
		result = append(result, n.contact)
		count++
	}
	i := 1
	for count < 20 && (prefixlen-i >= 0 || prefixlen+i < 160) {
		if prefixlen-i >= 0 {
			bucket = table.BucketLists[prefixlen-i]
			for e := bucket.First(); e != nil; e = e.Next() {
				result = append(result, e.contact)
				count++
			}
		}
		if prefixlen+i < 160 {
			bucket = table.BucketLists[prefixlen+i]
			for e := bucket.First(); e != nil; e = e.Next() {
				result = append(result, e.contact)
				count++
			}
		}
		i++
	}
	if count > 20 {
		temp := make([]Contact, 0)
		for i := 0; i < 20; i++ {
			temp = append(temp, result[i])
		}
		return temp
	}
	return result
}
