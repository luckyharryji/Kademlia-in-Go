package libkademlia

type RoutingTable struct {
	SelfId      ID
	BucketLists [b]*BucketList
}

/*
	Create a new K-Buckets Object
*/
func NewRoutingTable(id ID) (table *RoutingTable) {
	table = new(RoutingTable)
	table.SelfId = id
	for i := 0; i < 8*IDBytes; i++ {
		table.BucketLists[i] = new(BucketList)
	}
	return
}

/*
	Update K-Buckets

	If did not find contact:
		- if length of the list is less than 20
		- else ping the head of the list
			- if the head is still alive, ingore the new contact
			- else, delete the head and insert new contact to the tail
	else:
	 	- move the contact to the end
*/
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
			//If the node is not alive, update the table
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

/*
	Find the k-nearest node for the given contact
	Manipulate with pointer type

	CreateShortList will add the node in the list to the result list
	FindCloest will loop until: get all nodes/get 20 nodes
*/
func CreateShortList(contactList *[]Contact, list *BucketList, count *int, id ID) {
	for e := list.head; e != nil; e = e.Next() {
		if *count < k {
			if !id.Equals(e.contact.NodeID) {
				*contactList = append(*contactList, e.contact)
				*count++
			}
		} else {
			return
		}
	}
}

func (table *RoutingTable) FindAlpha(id ID) []Contact {
	prefixlen := id.Xor(table.SelfId).PrefixLen()
	bucket := table.BucketLists[prefixlen]
	var shortlist []Contact
	count := 0
	CreateShortList(&shortlist, bucket, &count, id)
	for i := 1; (prefixlen-i >= 0 || prefixlen+i < b) && count <= 20; i++ {
		if prefixlen-i >= 0 {
			bucket = table.BucketLists[prefixlen-i]
			CreateShortList(&shortlist, bucket, &count, id)
		}
		if prefixlen+i < b {
			bucket = table.BucketLists[prefixlen+i]
			CreateShortList(&shortlist, bucket, &count, id)
		}
	}
	return shortlist
}

func (table *RoutingTable) FindCloset(id ID) []Contact {
	prefixlen := id.Xor(table.SelfId).PrefixLen()
	bucket := table.BucketLists[prefixlen]
	var shortlist []Contact
	count := 0
	CreateShortList(&shortlist, bucket, &count, id)
	for i := 1; (prefixlen-i >= 0 || prefixlen+i < b) && count <= k; i++ {
		if prefixlen-i >= 0 {
			bucket = table.BucketLists[prefixlen-i]
			CreateShortList(&shortlist, bucket, &count, id)
		}
		if prefixlen+i < b {
			bucket = table.BucketLists[prefixlen+i]
			CreateShortList(&shortlist, bucket, &count, id)
		}
	}
	return shortlist
}
