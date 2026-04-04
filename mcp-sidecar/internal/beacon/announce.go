package beacon

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
)

const multicastGroup = "239.255.99.1"

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
		if msg["type"] != "discovery" {
			continue
		}

		log.Printf("Discovery request from %s, announcing...", sender.String())
		if _, err := conn.WriteToUDP(data, sender); err != nil {
			log.Printf("Failed to send announce: %v", err)
		}
	}
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
