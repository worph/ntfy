package beacon

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net"
	"time"
)

const multicastGroup = "239.255.99.1"

// ackMagic opens the 8-byte ack sent instead of the manifest when a discovery
// request carries "ack": the magic, then the FNV-1a 32-bit hash of the
// manifest, big-endian. Beacon then asks for the manifest 1:1 with
// {"type":"manifest"} only when the hash is new. Full manifests answered to a
// broadcast all land on Beacon at once and overflow its socket buffer; an ack
// never can. A request without "ack" (an older Beacon) still gets the full
// manifest. See beacon/sdk/node/mcp-announce.js.
var ackMagic = []byte("BCNA")

type announcePayload struct {
	Type        string           `json:"type"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Port        int              `json:"port"`
	Path        string           `json:"path"`
	Tools       []map[string]any `json:"tools"`
}

// StartAnnouncer listens for Beacon discovery packets on the multicast group
// and responds with the server's announce payload.
func StartAnnouncer(ctx context.Context, tools []map[string]any, mcpPort int) error {
	payload := announcePayload{
		Type:        "announce",
		Name:        "ntfy",
		Description: "Push notifications via ntfy",
		Port:        mcpPort,
		Path:        "/mcp",
		Tools:       tools,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	h := fnv.New32a()
	h.Write(data)
	ack := binary.BigEndian.AppendUint32(append([]byte{}, ackMagic...), h.Sum32())

	addr, err := net.ResolveUDPAddr("udp4", multicastGroup+":"+itoa(mcpPort))
	if err != nil {
		return err
	}

	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Printf("Beacon announcer listening on %s:%d (multicast)", multicastGroup, mcpPort)

	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, sender, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			log.Printf("UDP read error: %v", err)
			continue
		}

		var msg map[string]string
		if json.Unmarshal(buf[:n], &msg) != nil {
			continue
		}
		reply := data
		switch {
		case msg["type"] == "manifest":
		case msg["type"] == "discovery" && msg["ack"] != "":
			log.Printf("Discovery request from %s, acking", sender.String())
			reply = ack
		case msg["type"] == "discovery":
			log.Printf("Discovery request from %s, announcing...", sender.String())
		default:
			continue
		}
		if _, err := conn.WriteToUDP(reply, sender); err != nil {
			log.Printf("Failed to send announce: %v", err)
		}
	}
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
