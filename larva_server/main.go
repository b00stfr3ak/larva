package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os/exec"
	"strings"

	pb "github.com/b00stfr3ak/larva"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	port = ":50051"
)

type server struct {
	CommandInfo      []commandInfo
	CommandsExecuted int32
}

type commandInfo struct {
	ID     int32
	Name   string
	Status string
}

func (s *server) updateStatus(id int32, status string) {
	s.CommandInfo[id].Status = status
}

func (s *server) incCommandsExecuted() {
	s.CommandsExecuted++
}

func argCleanup(args string) []string {
	if args == "" {
		return []string{}
	}
	return strings.Split(args, " ")
}

func (s *server) CMD(ctx context.Context, in *pb.Request) (*pb.Reply, error) {
	ci := commandInfo{}
	ci.Status = "running"
	ci.Name = fmt.Sprintf("%s %s", in.Command, in.Argument)
	ci.ID = s.CommandsExecuted
	s.incCommandsExecuted()
	s.CommandInfo = append(s.CommandInfo, ci)
	args := argCleanup(in.Argument)
	var out []byte
	var err error
	if len(args) != 0 {
		out, err = exec.Command(in.Command, args...).Output()
		if err != nil {
			s.updateStatus(ci.ID, fmt.Sprintf("%v", err))
			return nil, err
		}
	} else {
		out, err = exec.Command(in.Command).Output()
		if err != nil {
			s.updateStatus(ci.ID, fmt.Sprintf("%v", err))
			return nil, err
		}
	}
	s.updateStatus(ci.ID, "complete")
	return &pb.Reply{Message: string(out)}, nil
}

func (s *server) StreamCMD(in *pb.Request, stream pb.Execute_StreamCMDServer) error {
	ci := commandInfo{}
	ci.Status = "running"
	ci.Name = fmt.Sprintf("%s %s", in.Command, in.Argument)
	ci.ID = s.CommandsExecuted
	s.incCommandsExecuted()
	s.CommandInfo = append(s.CommandInfo, ci)
	cmd := exec.Command(in.Command, argCleanup(in.Argument)...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.updateStatus(ci.ID, fmt.Sprintf("stdoutpipe error %v", err))
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.updateStatus(ci.ID, fmt.Sprintf("stderrpipe error %v", err))
		return err
	}
	if err := cmd.Start(); err != nil {
		s.updateStatus(ci.ID, fmt.Sprintf("cmd start error %v", err))
		return err
	}
	o := bufio.NewReader(stdout)
	for {
		out, _, err := o.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			s.updateStatus(ci.ID, fmt.Sprintf("stdout error %v", err))
			return err
		}
		if err := stream.Send(&pb.Reply{Message: string(out)}); err != nil {
			s.updateStatus(ci.ID, fmt.Sprintf("%v", err))
			return err
		}
	}
	r := bufio.NewReader(stderr)
	line, _, _ := r.ReadLine()
	if len(line) != 0 && err != io.EOF {
		s.updateStatus(ci.ID, fmt.Sprintf("stderr %v", string(line)))
		//return errors.New(string(line))
	}
	if err := cmd.Wait(); err != nil {
		s.updateStatus(ci.ID, fmt.Sprintf("cmd wait error %v", err))
		return err
	}
	s.updateStatus(ci.ID, "complete")
	return nil
}

func (s *server) List(in *pb.Empty, stream pb.Execute_ListServer) error {
	for _, c := range s.CommandInfo {
		if err := stream.Send(&pb.Info{Id: c.ID, Name: c.Name, Status: c.Status}); err != nil {
			return err
		}
	}
	return nil
}

func (s *server) Status(ctx context.Context, in *pb.ID) (*pb.Info, error) {
	for _, value := range s.CommandInfo {
		if value.ID == in.Id {
			return &pb.Info{Id: value.ID, Name: value.Name, Status: value.Status}, nil
		}
	}
	return nil, errors.New("ID not found")
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v\n", err)
	}
	s := grpc.NewServer()
	server := &server{CommandsExecuted: 0}
	pb.RegisterExecuteServer(s, server)
	s.Serve(lis)
}
