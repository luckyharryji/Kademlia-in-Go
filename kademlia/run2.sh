export GOPATH=`pwd`
go install kademlia
./bin/kademlia localhost:2335 localhost:2334
