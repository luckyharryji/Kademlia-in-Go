package libkademlia

import ()

type BucketNode struct {
	contact    Contact
	next, prev *BucketNode
}

type BucketList struct {
	head, tail *BucketNode
	length     int
}

func (list *BucketList) First() *BucketNode {
	return list.head
}

func (node *BucketNode) Next() *BucketNode {
	return node.next
}

func (node *BucketNode) Prev() *BucketNode {
	return node.prev
}

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

func (list *BucketList) DeleteFrontInsert(contact Contact) *BucketList {
	frontNode := list.head
	list.head = frontNode.next
	list.head.prev = nil
	list.Push(contact)
	return list
}

func (list *BucketList) MoveToTail(contact Contact) *BucketList {
	node := list.Find(contact)
	node.prev = node.next
	list.length -= 1
	list.Push(contact)
	return list
}
