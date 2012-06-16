GOFILES = ./Backend/src/AccessControlList.go \
	./Backend/src/AdminHandler.go \
	./Backend/src/AliasSystem.go \
	./Backend/src/ConnectHandler.go \
	./Backend/src/ConnectionAcceptor.go \
	./Backend/src/ConnectionAuthenticator.go \
	./Backend/src/DataStore.go \
	./Backend/src/EchoServer.go \
	./Backend/src/EventDrivenArchitecture.go \
	./Backend/src/FileServer.go \
	./Backend/src/Forwarder.go \
	./Backend/src/LocalConnectAuthenticator.go \
	./Backend/src/LocalConnectionDispatcher.go \
	./Backend/src/MailDBsqlite.go \
	./Backend/src/MailServer.go \
	./Backend/src/main.go \
	./Backend/src/PortMapper.go \
	./Backend/src/RequestDispatcher.go \
	./Backend/src/UpdateManager.go \
	./Backend/src/WebFrontend.go

all: backend ./Tests/testgenerator graph_tools

backend: ./Backend/bin/cobweb_backend

clean: clean_test clean_graph
	rm -rv ./Backend/bin; true

clean_graph:
	rm ./SocialGraph/*.gob ./SocialGraph/*.png ; true
	rm ./SocialGraph/computeAvailability ./SocialGraph/computeRedundancy ./SocialGraph/generateGraph; true

./Backend/bin/cobweb_backend: $(GOFILES)
	mkdir -p ./Backend/bin
	go build -x -o ./Backend/bin/cobweb_backend $(GOFILES)

./Tests/testgenerator: ./Tests/generatetest.go
	go build -x -o ./Tests/testgenerator ./Tests/generatetest.go

testgenerator: ./Tests/testgenerator

clean_test:
	rm -rvf ./Tests/clients ./Tests/testgenerator; true

prepare_test: backend testgenerator
	cd ./Tests/ && ./testgenerator -r=1 -n=10

connect_test:
	cd ./Tests/ && ./testgenerator -r=2 -n=10

run_test:
	cd ./Tests/ && ./testgenerator -r=3 -n=10

test_shell:
	cd ./Tests/clients/client0000 && ../../client.sh 10000


graph_tools: ./SocialGraph/generateGraph ./SocialGraph/computeAvailability ./SocialGraph/computeRedundancy

./SocialGraph/generateGraph: ./SocialGraph/generateGraph.go
	go build -x -o ./SocialGraph/generateGraph ./SocialGraph/generateGraph.go
./SocialGraph/computeAvailability: ./SocialGraph/computeAvailability.go
	go build -x -o ./SocialGraph/computeAvailability ./SocialGraph/computeAvailability.go
./SocialGraph/computeRedundancy: ./SocialGraph/computeRedundancy.go
	go build -x -o ./SocialGraph/computeRedundancy ./SocialGraph/computeRedundancy.go

