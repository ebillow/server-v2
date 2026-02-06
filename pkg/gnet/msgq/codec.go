package msgq

import (
	"google.golang.org/protobuf/proto"
	"server/pkg/pb"
)

func encode(msg *pb.NatsMsg) ([]byte, error) {
	b, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return b, err
}

func decode(in []byte) (*pb.NatsMsg, error) {
	msg := &pb.NatsMsg{}
	err := proto.Unmarshal(in, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}
