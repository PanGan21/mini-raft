server-1:
	go run cmd/main.go -port="8001" -server-name="test-server-1"

server-2:
	go run cmd/main.go -port="8002" -server-name="test-server-2"

server-3:
	go run cmd/main.go -port="8003" -server-name="test-server-3"

server-4:
	go run cmd/main.go -port="8004" -server-name="test-server-4"

client-1:
	go run client/client.go localhost:8001

clean:
	rm -rf registry.txt state.txt test-server-1.txt test-server-2.txt test-server-3.txt test-server-4.txt