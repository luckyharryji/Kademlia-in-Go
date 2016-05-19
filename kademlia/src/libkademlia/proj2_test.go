package libkademlia

import (
	"bytes"
	//"container/heap"
	//"math/rand"
	//	"fmt"
	//"net"
	"strconv"
	"testing"
	//"time"
)

func Connect(t *testing.T, list []*Kademlia, kNum int) {
	for i := 0; i < kNum; i++ {
		for j := 0; j < kNum; j += 5 {
			if j != i {
				list[i].DoPing(list[j].SelfContact.Host, list[j].SelfContact.Port)
			}
		}
	}
}

func TestIterativeFindNode(t *testing.T) {
	// tree structure;
	// A->B->tree
	/*
	         C
	      /
	  A-B -- D
	      \
	         E
	*/
	kNum := 40
	targetIdx := kNum - 10
	instance1 := NewKademlia("localhost:7304")
	instance2 := NewKademlia("localhost:7305")
	host2, port2, _ := StringToIpPort("localhost:7305")
	instance1.DoPing(host2, port2)
	_, err := instance1.FindContact(instance2.NodeID)
	if err != nil {
		t.Error("Instance 2's contact not found in Instance 1's contact list")
		return
	}
	tree_node := make([]*Kademlia, kNum)
	for i := 0; i < kNum; i++ {
		address := "localhost:" + strconv.Itoa(7306+i)
		tree_node[i] = NewKademlia(address)
		tree_node[i].DoPing(host2, port2)
	}
	/*
		for i := 0; i < kNum; i++ {
			if i != targetIdx {
				tree_node[targetIdx].DoPing(tree_node[i].SelfContact.Host, tree_node[i].SelfContact.Port)
			}
		}
	*/
	SearchKey := tree_node[targetIdx].SelfContact.NodeID
	Connect(t, tree_node, kNum)
	res, err := tree_node[0].DoIterativeFindNode(SearchKey)
	if err != nil {
		t.Error(err.Error())
	}
	if res == nil || len(res) == 0 {
		t.Error("No contacts were found")
	}
	find := false
	for _, value := range res {
		if value.NodeID.Equals(SearchKey) {
			find = true
		}
	}
	if !find {
		t.Error("Find wrong id")
	}
	return
}

func TestIterativeStore(t *testing.T) {
	// tree structure;
	// A->B->tree
	/*
	         C
	      /
	  A-B -- D
	      \
	         E
	*/
	kNum := 40
	targetIdx := kNum - 10
	instance1 := NewKademlia("localhost:10004")
	instance2 := NewKademlia("localhost:10005")
	host2, port2, _ := StringToIpPort("localhost:10005")
	instance1.DoPing(host2, port2)
	_, err := instance1.FindContact(instance2.NodeID)
	if err != nil {
		t.Error("Instance 2's contact not found in Instance 1's contact list")
		return
	}
	tree_node := make([]*Kademlia, kNum)
	for i := 0; i < kNum; i++ {
		address := "localhost:" + strconv.Itoa(10006+i)
		tree_node[i] = NewKademlia(address)
		tree_node[i].DoPing(host2, port2)
	}
	/*
		for i := 0; i < kNum; i++ {
			if i != targetIdx {
				tree_node[targetIdx].DoPing(tree_node[i].SelfContact.Host, tree_node[i].SelfContact.Port)
			}
		}
	*/
	SearchKey := tree_node[targetIdx].SelfContact.NodeID
	Connect(t, tree_node, kNum)
	value := []byte("hello")
	res, err := tree_node[0].DoIterativeStore(SearchKey, value)
	if err != nil {
		t.Error(err.Error())
		return
	}
	if res == nil || len(res) == 0 {
		t.Error("No contacts were found")
	}
	find := true
	for _, node := range res {
		res, _, err := tree_node[0].DoFindValue(&node, SearchKey)
		if err != nil {
			find = false
		}
		if !bytes.Equal(res, value) {
			find = false
		}
	}
	if !find {
		t.Error("Find wrong value")
	}
	return
}

func TestIterativeFindValue(t *testing.T) {
	// tree structure;
	// A->B->tree
	/*
	         C
	      /
	  A-B -- D
	      \
	         E
	*/
	kNum := 40
	targetIdx := kNum - 10
	instance1 := NewKademlia("localhost:20004")
	instance2 := NewKademlia("localhost:20005")
	host2, port2, _ := StringToIpPort("localhost:20005")
	instance1.DoPing(host2, port2)
	_, err := instance1.FindContact(instance2.NodeID)
	if err != nil {
		t.Error("Instance 2's contact not found in Instance 1's contact list")
		return
	}
	tree_node := make([]*Kademlia, kNum)
	for i := 0; i < kNum; i++ {
		address := "localhost:" + strconv.Itoa(20006+i)
		tree_node[i] = NewKademlia(address)
		tree_node[i].DoPing(host2, port2)
	}

	for i := 0; i < kNum; i++ {
		if i != targetIdx {
			tree_node[i].DoPing(tree_node[targetIdx].SelfContact.Host, tree_node[targetIdx].SelfContact.Port)
		}
	}

	SearchKey := tree_node[targetIdx].SelfContact.NodeID
	Connect(t, tree_node, kNum)
	value := []byte("hello")
	tree_node[0].DoStore(&tree_node[targetIdx].SelfContact, SearchKey, value)
	res, err := tree_node[0].DoIterativeFindValue(SearchKey)
	//fmt.Println("SelfContact NodeID :" + tree_node[0].NodeID.AsString())
	if err != nil {
		t.Error(err.Error())
	}
	find := true
	if !bytes.Equal(res, value) {
		find = false
	}
	if !find {
		t.Error("Find wrong value")
	}
	return
}
