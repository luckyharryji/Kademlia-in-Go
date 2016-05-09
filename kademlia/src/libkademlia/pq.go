package libkademlia

import ()

type PriorityQueue struct {
	List   []*Contact
	NodeID ID
}

func (pq PriorityQueue) Len() int {
	return len(pq.List)
}

func contactLess(c1, c2 *Contact, key ID) bool {
	dist1 := key.Xor(c1.NodeID)
	dist2 := key.Xor(c2.NodeID)
	return dist1.Less(dist2)
}

func (pq PriorityQueue) Less(i, j int) bool {
	return contactLess(pq.List[i], pq.List[j], pq.NodeID)
}

func (pq PriorityQueue) Swap(i, j int) {
	pq.List[i], pq.List[j] = pq.List[j], pq.List[i]
}

func (pq *PriorityQueue) Pop() interface{} {
	old := pq.List
	n := len(old)
	x := old[n-1]
	pq.List = old[0 : n-1]
	return x
}

func (pq *PriorityQueue) Push(x interface{}) {
	pq.List = append(pq.List, x.(*Contact))
}
