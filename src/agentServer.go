package main

import (
	pb "agent/protobuf"
	"context"
	"fmt"
)

type AgentServer struct {
	pb.UnimplementedAgentServer
	HostName string
}

func (s *AgentServer) HelloWorld(context.Context, *pb.HelloWorldRequest) (*pb.HelloWorldResponse, error) {
	return &pb.HelloWorldResponse{
		Message: fmt.Sprintf("Hello, World!\n\n\t-from %s", s.HostName),
	}, nil
}
