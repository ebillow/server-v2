package msgq

import (
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"log"
	"server/internal/pb"
	"server/internal/util"
	"strings"
	"sync"
	"testing"
	"time"
)

func connect() *nats.Conn {
	servers := []string{"nats://127.0.0.1:4222"}
	nc, err := nats.Connect(strings.Join(servers, ","), nats.UserInfo("123456", "123456"))
	if err != nil {
		log.Fatal(err)
	}
	return nc
}

func TestConnect(t *testing.T) {
	nc := connect()
	defer nc.Close()
}

func TestSyncSubscribe(t *testing.T) {
	nc := connect()
	defer nc.Close()

	// Subscribe
	sub, err := nc.SubscribeSync("updates")
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for i := 0; i < 10; i++ {
			nc.Publish("updates", []byte("hello world"))
			time.Sleep(time.Second)
		}
	}()

	for i := 0; i < 11; i++ {
		// Wait for a message
		msg, err := sub.NextMsg(10 * time.Second)
		if err != nil {
			log.Fatal(err)
		}

		// Use the response
		log.Printf("Reply: %s", msg.Data)
	}
}

func TestSubscribe(t *testing.T) {
	nc := connect()

	// Use a WaitGroup to wait for a message to arrive
	wg := sync.WaitGroup{}
	wg.Add(1)

	// Subscribe
	if _, err := nc.Subscribe("updates", func(m *nats.Msg) {
		wg.Done()
	}); err != nil {
		log.Fatal(err)
	}

	// Wait for a message to come in
	wg.Wait()
}

// BenchmarkEncode-10    	 3087978	       383.8 ns/op
func BenchmarkEncode(b *testing.B) {
	serName := "game"
	serID := int32(1)
	msgID := 2

	for i := 0; i < b.N; i++ {
		msg := nats.NewMsg(getIndexSubject(serName, serID))

		data, err := proto.Marshal(&pb.RankItemShow{
			Id:      1,
			Name:    "name",
			Score:   999,
			Club:    "club",
			Country: "cn",
			Flag:    1,
			Vip:     99,
		})
		if err != nil {
			log.Fatal(err)
		}
		msg.Header.Set("ser_name", "self name")
		msg.Header.Set("ser_id", "self_id")
		msg.Header.Set("msg_id", util.ToString(msgID))
		msg.Header.Set("role_id", "3")
		msg.Data = data
		// ------------------
		msg.Header.Get("ser_name")
		msg.Header.Get("ser_id")
		msg.Header.Get("msg_id")
		msg.Header.Get("role_id")

		dataMsg := &pb.RankItemShow{}
		err = proto.Unmarshal(msg.Data, dataMsg)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// BenchmarkHead-10    	 9132805	       127.9 ns/op
func BenchmarkHead(b *testing.B) {
	serName := "game"
	serID := int32(1)
	msgID := 2

	for i := 0; i < b.N; i++ {
		msg := nats.NewMsg(getIndexSubject(serName, serID))

		msg.Header.Set("ser_name", "self name")
		msg.Header.Set("ser_id", "self_id")
		msg.Header.Set("msg_id", util.ToString(msgID))
		msg.Header.Set("role_id", "3")

		// ------------------
		msg.Header.Get("ser_name")
		msg.Header.Get("ser_id")
		msg.Header.Get("msg_id")
		msg.Header.Get("role_id")
	}
}
