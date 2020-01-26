package main

import (
	"bytes"
	"flag"
	"log"

	"github.com/Telefonica/nfqueue"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

func main() {
	queue := flag.Uint("queue", 0, "queue num")
	flag.Parse()

	q := NewQueue(uint16(*queue))
	err := q.Start()
	if err != nil {
		log.Fatalf("failed start queue: %v", err)
	}
}

type Queue struct {
	id    uint16
	queue *nfqueue.Queue
}

func NewQueue(id uint16) *Queue {
	q := &Queue{
		id: id,
	}
	queueCfg := &nfqueue.QueueConfig{
		MaxPackets: 1000,
		BufferSize: 16 * 1024 * 1024,
		QueueFlags: []nfqueue.QueueFlag{nfqueue.FailOpen},
	}

	q.queue = nfqueue.NewQueue(q.id, q, queueCfg)
	return q
}

func (q *Queue) Start() error {
	return q.queue.Start()
}

func (q *Queue) Stop() error {
	return q.queue.Stop()
}

func (q *Queue) Handle(p *nfqueue.Packet) {
	packet := gopacket.NewPacket(p.Buffer, layers.LayerTypeIPv4, gopacket.Default)
	ip := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	if l := packet.Layer(layers.LayerTypeTCP); l != nil {
		tcp := l.(*layers.TCP)
		err := tcp.SetNetworkLayerForChecksum(ip)
		if err != nil {
			panic(err)
		}

		// edit tcp sync
		if tcp.SYN {
			var mss, ts, ws layers.TCPOption
			for _, o := range tcp.Options {
				if o.OptionType == layers.TCPOptionKindTimestamps {
					ts = o
				}
				if o.OptionType == layers.TCPOptionKindMSS {
					mss = o
				}

				if o.OptionType == layers.TCPOptionKindWindowScale {
					ws = o
				}
			}

			// change tcp syn options
			tcp.Options = []layers.TCPOption{
				mss,
				{
					OptionType:   layers.TCPOptionKindSACKPermitted,
					OptionLength: 2,
					OptionData:   nil,
				},
				ts,
				{OptionType: layers.TCPOptionKindNop},
				{OptionType: layers.TCPOptionKindNop},
				{OptionType: layers.TCPOptionKindNop},
				{OptionType: layers.TCPOptionKindNop},
				{OptionType: layers.TCPOptionKindNop},
				ws,
			}
		}

		// new packet
		buf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{
			FixLengths:       true,
			ComputeChecksums: true,
		}

		// replace data in payload (try "curl http://httpbin.org/get?test=nfqueue")
		payload := bytes.ReplaceAll(tcp.Payload, []byte("/get?test=nfqueue"), []byte("/get?test=replace"))

		// recreate packet
		err = gopacket.SerializeLayers(buf, opts,
			ip,
			tcp,
			gopacket.Payload(payload),
		)
		if err != nil {
			panic(err)
		}

		tcpNew := buf.Bytes()

		spew.Dump(packet)
		spew.Dump("old", p.Buffer, "new", tcpNew)

		err = p.Modify(tcpNew)
		if err != nil {
			panic(err)
		}

		return
	}

	// Accept the packet
	err := p.Accept()
	if err != nil {
		panic(err)
	}
}
