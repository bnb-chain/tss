all: protob
	go build

protob:
	protoc -I=./proto --go_out=./common ./proto/messages.proto
	protoc -I=./proto --go_out=./common ./proto/ClientMessage.proto
	protoc -I=./proto --go_out=./common ./proto/Heartbeat.proto