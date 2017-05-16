gen:
	protoc -I raft raft/raft.proto --go_out=plugins=grpc:raft
