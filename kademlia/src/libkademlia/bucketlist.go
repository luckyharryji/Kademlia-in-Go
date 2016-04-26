package libkademlia

/*
New lib created for bucketlist based on Golang double list

BucketNode contains information of contact and its location in the list
BucketList contains length info and the structure of the list
*/

type BucketNode struct {
	contact    Contact
	next, prev *BucketNode
}

type BucketList struct {
	head, tail *BucketNode
	length     int
}

/*
Basic manipulation of the Node
*/
func (list *BucketList) First() *BucketNode {
	return list.head
}

func (node *BucketNode) Next() *BucketNode {
	return node.next
}

func (node *BucketNode) Prev() *BucketNode {
	return node.prev
}

/* Push: Add new node to the list when new contact communicate with itself.
	Decide if list is empty before adding.
	Length of the list plus 1.

	Args:
		contact(Contact): new contact for the list.
	Return:
		*BuckerList: the BuckerList after inserting the contact node.
*/
func (list *BucketList) Push(contact Contact) *BucketList {
	n := &BucketNode{contact: contact}
	if list.head == nil {
		list.head = n
	} else {
		list.tail.next = n
		n.prev = list.tail
	}
	list.tail = n
	list.length += 1
	return list
}

/*
	Find: Try to find the node in the list which match the given contact

	Args:
		contact(Contact)
	Return:
		*BucketNode/nil if contact info does not exist
*/
func (list *BucketList) Find(contact Contact) *BucketNode {
	found := false
	var ret *BucketNode = nil
	for n := list.First(); n != nil && !found; n = n.Next() {
		if n.contact.NodeID.Equals(contact.NodeID) {
			found = true
			ret = n
		}
	}
	return ret
}

/*
	DeleteFrontInsert: Delete the front of the bucketlist and insert the new
	contact info to the tail of the list.

	This function is used when the server decide the head node in the list is
	inactive/dead and the new node needs to come in to be recorded.
*/
func (list *BucketList) DeleteFrontInsert(contact Contact) *BucketList {
	frontNode := list.head
	list.head = frontNode.next
	list.head.prev = nil
	list.Push(contact)
	return list
}

/*
	MoveToTail: move the existing node in the list to the tail

	This function is used when a client showed before contact with the server
	again.

	If the node is the head/last node in the list, return immediately.

	Args:
		contact(Contact)
	Return:
		*BucketLists
*/
func (list *BucketList) MoveToTail(contactNode *BucketNode) *BucketList {
	nodeOfContact := contactNode
	previousNode := nodeOfContact.Prev()
	afterNode := nodeOfContact.Next()
	if previousNode == nil || afterNode == nil {
		return list
	}
	previousNode.next = afterNode
	afterNode.prev = previousNode
	list.length -= 1
	list.Push(nodeOfContact.contact)
	return list
}
