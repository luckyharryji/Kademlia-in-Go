run:
	export GOPATH=`pwd`
	go install kademlia
	./bin/kademlia localhost:2334 localhost:2334

test:
	go install kademlia
	go test -race -v libkademlia
