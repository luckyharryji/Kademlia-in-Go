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
	Buckets [b] *BucketList
}

// type ContactForSort struct {
// 	contact Contact
// 	id      ID
// }

type Contacts []ContactForSort

func NewRoutingTable() (table *RoutingTable) {
	table = new(RoutingTable)
	for i := 0; i < 8 * IDBytes; i++ {
		table.BucketLists[i] = &BucketList{length: 0}
	}
	return
}

func (table *RoutingTable) UpDate(selfID ID, contact Contact) {
	id_table := table.SelfContact.NodeID
	id := contact.NodeID
	prefixlen := id.Xor(id_table).PrefixLen()
	bucket := table.BucketLists[prefixlen]
	var element *list.Element
	for e := bucket.Front(); e != nil; e = e.Next() {
		if e.Value.(*Contact).NodeID.Equals(contact.NodeID) {
			element = e
			break
		}
	}
	if element == nil {
		if bucket.Len() <= k {
			bucket.PushFront(contact)
		}
		//error := Ping
		//if error != nil {
		//	bucket.Remove(contact)
		//}
	} else {
		bucket.MoveToFront(element)
	}
}

func CreateShortList(shortlist Contacts, list list.List, count int, id ID) {
	for e := list.Front(); e != nil; e = e.Next() {
		temp := ContactForSort{e.Value.(Contact), id}
		shortlist = append(shortlist, temp)
		count++
	}
}

func (table *RoutingTable) FindCloset(id ID) []Contact {
	prefixlen := id.Xor(table.SelfContact.NodeID).PrefixLen()
	bucket := table.BucketLists[prefixlen]
	var shortlist Contacts
	count := 0
	CreateShortList(shortlist, *bucket, count, id)
	for i := 1; (prefixlen-i >= 0 || prefixlen+i < IDBytes*8) && count <= k; i++ {
		if prefixlen-i >= 0 {
			bucket = table.BucketLists[prefixlen-i]
			CreateShortList(shortlist, *bucket, count, id)
		}
		if prefixlen+i < IDBytes*8 {
			bucket = table.BucketLists[prefixlen+i]
			CreateShortList(shortlist, *bucket, count, id)
		}
	}
	sort.Sort(shortlist)
	result := make([]Contact, k)
	for i := 0; i < k; i++ {
		result = append(result, shortlist[i].contact)
	}
	return result
}

func (list Contacts) Len() int {
	return len(list)
}

func (list Contacts) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

func (list Contacts) Less(i, j int) bool {
	return list[i].contact.NodeID.Xor(list[i].id).Less(list[j].contact.NodeID.Xor(list[j].id))
}
