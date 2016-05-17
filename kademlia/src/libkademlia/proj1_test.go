package libkademlia

import (
	"bytes"
	//"container/heap"
	//"math/rand"
	"fmt"
	"net"
	"strconv"
	"testing"
	//"time"
)

func StringToIpPort(laddr string) (ip net.IP, port uint16, err error) {
	hostString, portString, err := net.SplitHostPort(laddr)
	if err != nil {
		return
	}
	ipStr, err := net.LookupHost(hostString)
	if err != nil {
		return
	}
	for i := 0; i < len(ipStr); i++ {
		ip = net.ParseIP(ipStr[i])
		if ip.To4() != nil {
			break
		}
	}
	portInt, err := strconv.Atoi(portString)
	port = uint16(portInt)
	return
}

func TestPing(t *testing.T) {
	instance1 := NewKademlia("localhost:7890")
	instance2 := NewKademlia("localhost:7891")
	host2, port2, _ := StringToIpPort("localhost:7891")
	contact2, err := instance2.FindContact(instance2.NodeID)
	if err != nil {
		t.Error("A node cannot find itself's contact info")
	}
	contact2, err = instance2.FindContact(instance1.NodeID)
	if err == nil {
		t.Error("Instance 2 should not be able to find instance " +
			"1 in its buckets before ping instance 1")
	}
	instance1.DoPing(host2, port2)
	contact2, err = instance1.FindContact(instance2.NodeID)
	if err != nil {
		t.Error("Instance 2's contact not found in Instance 1's contact list")
		return
	}
	wrong_ID := NewRandomID()
	_, err = instance2.FindContact(wrong_ID)
	if err == nil {
		t.Error("Instance 2 should not be able to find a node with the wrong ID")
	}

	contact1, err := instance2.FindContact(instance1.NodeID)
	if err != nil {
		t.Error("Instance 1's contact not found in Instance 2's contact list")
		return
	}
	if contact1.NodeID != instance1.NodeID {
		t.Error("Instance 1 ID incorrectly stored in Instance 2's contact list")
	}
	if contact2.NodeID != instance2.NodeID {
		t.Error("Instance 2 ID incorrectly stored in Instance 1's contact list")
	}
	return
}

func TestStore(t *testing.T) {
	// test Dostore() function and LocalFindValue() function
	instance1 := NewKademlia("localhost:7892")
	instance2 := NewKademlia("localhost:7893")
	host2, port2, _ := StringToIpPort("localhost:7893")
	instance1.DoPing(host2, port2)
	contact2, err := instance1.FindContact(instance2.NodeID)
	if err != nil {
		t.Error("Instance 2's contact not found in Instance 1's contact list")
		return
	}
	key := NewRandomID()
	value := []byte("Hello World")
	err = instance1.DoStore(contact2, key, value)
	if err != nil {
		t.Error("Can not store this value")
	}
	storedValue, err := instance2.LocalFindValue(key)
	if err != nil {
		t.Error("Stored value not found!")
	}
	if !bytes.Equal(storedValue, value) {
		t.Error("Stored value did not match found value")
	}
	return
}

func TestFindNode(t *testing.T) {
	// tree structure;
	// A->B->tree
	/*
	         C
	      /
	  A-B -- D
	      \
	         E
	*/
	instance1 := NewKademlia("localhost:7894")
	instance2 := NewKademlia("localhost:7895")
	host2, port2, _ := StringToIpPort("localhost:7895")
	instance1.DoPing(host2, port2)
	contact2, err := instance1.FindContact(instance2.NodeID)
	if err != nil {
		t.Error("Instance 2's contact not found in Instance 1's contact list")
		return
	}
	tree_node := make([]*Kademlia, 10)
	for i := 0; i < 10; i++ {
		address := "localhost:" + strconv.Itoa(7896+i)
		tree_node[i] = NewKademlia(address)
		host_number, port_number, _ := StringToIpPort(address)
		instance2.DoPing(host_number, port_number)
	}
	key := NewRandomID()
	contacts, err := instance1.DoFindNode(contact2, key)
	if err != nil {
		t.Error("Error doing FindNode")
	}

	if contacts == nil || len(contacts) == 0 {
		t.Error("No contacts were found")
	}

	for i := 0; i < 10; i++ {
		returnedContact, err := instance1.FindContact(tree_node[i].NodeID)
		if err != nil {
			t.Error("Instance returned not found in Instance 1's contact list")
			return
		}
		if returnedContact.NodeID != tree_node[i].NodeID {
			t.Error("Returned ID incorrectly stored in Instance 1's contact list")
		}
	}
	return
}

func Connect(t *testing.T, list []*Kademlia, kNum int) {
	for i := 0; i < kNum; i++ {
		for j := 0; j < kNum; j += 2 {
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
	kNum := 50
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
		t.Log("ID:" + tree_node[i].SelfContact.NodeID.AsString())
	}
	SearchKey := tree_node[targetIdx].SelfContact.NodeID
	Connect(t, tree_node, kNum)
	fmt.Println("SearchKey:" + SearchKey.AsString())
	res, err := tree_node[0].DoIterativeFindNode(SearchKey)
	if err != nil {
		t.Error(err.Error())
	}
	t.Log(len(res))
	t.Log("SearchKey:" + SearchKey.AsString())
	if res == nil || len(res) == 0 {
		t.Error("No contacts were found")
	}
	find := false
	for _, value := range res {
		t.Log(value.NodeID.AsString())
		if value.NodeID.Equals(SearchKey) {
			find = true
		}
	}
	if !find {
		t.Log("Instance2:" + instance2.NodeID.AsString())
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
	kNum := 30
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
		t.Log("ID:" + tree_node[i].SelfContact.NodeID.AsString())
	}
	SearchKey := tree_node[targetIdx].SelfContact.NodeID
	Connect(t, tree_node, kNum)
	value := []byte("hello")
	res, err := tree_node[0].DoIterativeStore(SearchKey, value)
	fmt.Println("After interativestore")
	if err != nil {
		t.Error(err.Error())
		return
	}
	t.Log("SearchKey:" + SearchKey.AsString())
	if res == nil || len(res) == 0 {
		t.Error("No contacts were found")
	}
	find := true
	for _, node := range res {
		t.Log(node.NodeID.AsString())
		res, _, err := tree_node[0].DoFindValue(&node, SearchKey)
		if err != nil {
			find = false
		}
		if !bytes.Equal(res, value) {
			find = false
		}
	}
	if !find {
		t.Log("Instance2:" + instance2.NodeID.AsString())
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
	kNum := 50
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
		t.Log("ID:" + tree_node[i].SelfContact.NodeID.AsString())
	}
	SearchKey := tree_node[targetIdx].SelfContact.NodeID
	fmt.Println("Before Connect")
	Connect(t, tree_node, kNum)
	fmt.Println("After Connect")
	value := []byte("hello")
	tree_node[0].DoStore(&(tree_node[targetIdx].SelfContact), SearchKey, value)
	fmt.Println("Stored")
	res, err := tree_node[5].DoIterativeFindValue(SearchKey)
	if err != nil {
		t.Error(err.Error())
	}
	t.Log("SearchKey:" + SearchKey.AsString())
	find := true
	if !bytes.Equal(res, value) {
		find = false
	}
	if !find {
		t.Log("Instance2:" + instance2.NodeID.AsString())
		t.Error("Find wrong value")
	}
	return
}

func TestFindValue(t *testing.T) {
	// tree structure;
	// A->B->tree
	/*
	         C
	      /
	  A-B -- D
	      \
	         E
	*/
	instance1 := NewKademlia("localhost:7926")
	instance2 := NewKademlia("localhost:7927")
	host2, port2, _ := StringToIpPort("localhost:7927")
	instance1.DoPing(host2, port2)
	contact2, err := instance1.FindContact(instance2.NodeID)
	if err != nil {
		t.Error("Instance 2's contact not found in Instance 1's contact list")
		return
	}

	tree_node := make([]*Kademlia, 10)
	for i := 0; i < 10; i++ {
		address := "localhost:" + strconv.Itoa(7928+i)
		tree_node[i] = NewKademlia(address)
		host_number, port_number, _ := StringToIpPort(address)
		instance2.DoPing(host_number, port_number)
	}

	key := NewRandomID()
	value := []byte("Hello world")
	err = instance2.DoStore(contact2, key, value)
	if err != nil {
		t.Error("Could not store value")
	}

	// Given the right keyID, it should return the value
	foundValue, contacts, err := instance1.DoFindValue(contact2, key)
	if !bytes.Equal(foundValue, value) {
		t.Error("Stored value did not match found value")
	}

	//Given the wrong keyID, it should return k nodes.
	wrongKey := NewRandomID()
	foundValue, contacts, err = instance1.DoFindValue(contact2, wrongKey)
	if contacts == nil || len(contacts) < 10 {
		t.Error("Searching for a wrong ID did not return contacts")
	}

	for i := 0; i < 10; i++ {
		returnedContact, err := instance1.FindContact(tree_node[i].NodeID)
		if err != nil {
			t.Error("Instance returned not found in Instance 1's contact list")
			return
		}
		if returnedContact.NodeID != tree_node[i].NodeID {
			t.Error("Returned ID incorrectly stored in Instance 1's contact list")
		}
	}
	return
}

func TestReturnKContact(t *testing.T) {
	/*
		Test to see if findValue return exactly k contact even if it sotres more
		than K nodes information
	*/

	// tree structure;
	// A->B->tree
	/*
	         C
	      /
	  A-B -- D
	      \
	         E
	*/
	instance1 := NewKademlia("localhost:8926")
	instance2 := NewKademlia("localhost:8927")
	host2, port2, _ := StringToIpPort("localhost:8927")
	instance1.DoPing(host2, port2)
	contact2, err := instance1.FindContact(instance2.NodeID)
	if err != nil {
		t.Error("Instance 2's contact not found in Instance 1's contact list")
		return
	}

	tree_node := make([]*Kademlia, 30)
	for i := 0; i < 30; i++ {
		address := "localhost:" + strconv.Itoa(8928+i)
		tree_node[i] = NewKademlia(address)
		host_number, port_number, _ := StringToIpPort(address)
		instance2.DoPing(host_number, port_number)
	}

	key := NewRandomID()
	value := []byte("Hello world")
	err = instance2.DoStore(contact2, key, value)
	if err != nil {
		t.Error("Could not store value")
	}

	// Given the right keyID, it should return the value
	foundValue, contacts, err := instance1.DoFindValue(contact2, key)
	if !bytes.Equal(foundValue, value) {
		t.Error("Stored value did not match found value")
	}

	//Given the wrong keyID, it should return k nodes.
	wrongKey := NewRandomID()
	foundValue, contacts, err = instance1.DoFindValue(contact2, wrongKey)
	if contacts == nil || len(contacts) != 20 {
		t.Error("Searching for a wrong ID did not return contacts")
	}
	return
}
