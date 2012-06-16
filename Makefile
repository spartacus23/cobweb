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

backend: ./Backend/bin/cobweb_backend

clean: clean_test
	rm -v ./Backend/bin/cobweb_backend; true

./Backend/bin/cobweb_backend: $(GOFILES)
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


