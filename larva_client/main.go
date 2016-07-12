package main

import (
	"io"
	"log"

	pb "github.com/b00stfr3ak/larva"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	address = "localhost:50051"
)

func main() {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v\n", err)
	}
	defer conn.Close()
	c := pb.NewExecuteClient(conn)
	r, err := c.CMD(context.Background(), &pb.Request{Command: "ping", Argument: "-c 2 google.com"})
	if err != nil {
		log.Fatalf("could not send command: %v\n", err)
	}
	log.Printf("Results: %s\n", r.Message)
	s, err := c.StreamCMD(context.Background(), &pb.Request{Command: "nmap", Argument: "-sV -T4 -vv 127.0.0.1"})
	if err != nil {
		log.Fatalf("could not send command: %v\n", err)
	}
	for {
		b, err := s.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(c, err)
		}
		log.Println(b.Message)
	}
	l, err := c.List(context.Background(), &pb.Empty{})
	if err != nil {
		log.Fatalf("could not send command %v\n", err)
	}
	for {
		b, err := l.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(l, err)
		}
		log.Println(b)
	}
	//fmt.Println(c.Status(context.Background(), &pb.ID{Id: 1}))
	//fmt.Println(c.Status(context.Background(), &pb.ID{Id: 7}))
}
