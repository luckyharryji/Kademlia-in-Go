package libkademlia

import ()

type PriorityQueue struct {
	SelfContact Contact
	List        []Contact
	NodeID      ID
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
	return contactLess(&pq.List[i], &pq.List[j], pq.NodeID)
}

func (pq PriorityQueue) Swap(ii, jj int) {
	pq.List[ii], pq.List[jj] = pq.List[jj], pq.List[ii]
}

func (pq *PriorityQueue) Pop() interface{} {
	old := pq.List
	n := len(old)
	x := old[n-1]
	pq.List = old[0 : n-1]
	return x
}

func (pq *PriorityQueue) Push(x interface{}) {
	pq.List = append(pq.List, x.(Contact))
}

func (pq *PriorityQueue) Peek() (bool, Contact) {
	if pq.Len() <= 0 {
		return false, pq.SelfContact
	}
	return true, pq.List[0]
}

func (pq *PriorityQueue) Last() (bool, Contact) {
	old := pq.List
	n := len(old)
	if pq.Len() <= 0 {
		return false, pq.SelfContact
	}
	return true, pq.List[n-1]
}
