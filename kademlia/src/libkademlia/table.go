package libkademlia

import ()

type RoutingTable struct {
	SelfId      ID
	BucketLists [b]*BucketList
}

func NewRoutingTable(id ID) (table *RoutingTable) {
	table = new(RoutingTable)
	table.SelfId = id
	for i := 0; i < 8 * IDBytes; i++ {
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
		bucket.MoveToTail(element)
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

func CreateShortList(contactList *[]Contact, list *BucketList, count *int) {
	for e := list.head; e != nil; e = e.Next() {
		if *count < k {
			*contactList = append(*contactList, e.contact)
			*count++
		} else {
			return
		}
	}
}

func (table *RoutingTable) FindCloset(id ID) []Contact {
	prefixlen := id.Xor(table.SelfId).PrefixLen()
	bucket := table.BucketLists[prefixlen]
	var shortlist []Contact
	count := 0
	CreateShortList(&shortlist, bucket, &count)
	for i := 1; (prefixlen - i >= 0 || prefixlen + i < b) && count <= k; i++ {
		if prefixlen - i >= 0 {
			bucket = table.BucketLists[prefixlen-i]
			CreateShortList(&shortlist, bucket, &count)
		}
		if prefixlen + i < b {
			bucket = table.BucketLists[prefixlen+i]
			CreateShortList(&shortlist, bucket, &count)
		}
	}
	return shortlist
}
