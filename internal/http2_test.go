package internal

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/v2/errors"
)

func TestSingleUseRequestIsNotResentAfterHTTP2Reset(t *testing.T) {
	for _, body := range []map[string]string{{}, {"parent_origin": "https://app.example.com"}} {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = listener.Close() })
		var requests atomic.Int32
		go func() {
			for {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				go resetCompletedHTTP2Streams(conn, &requests)
			}
		}()

		var protocols http.Protocols
		protocols.SetUnencryptedHTTP2(true)
		transport := &http.Transport{Protocols: &protocols}
		t.Cleanup(transport.CloseIdleConnections)
		client := NewHTTPClient("http://"+listener.Addr().String(), "", "", time.Second)
		client.httpc.Transport = transport
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, err = client.NewRequest(http.MethodPost, "/embed/sessions/fixture-session-token/exchange").
			WithoutAuth().WithoutReplay().WithBody(body).Execute(ctx, nil)
		var networkErr *sdkerrors.NetworkError
		if !errors.As(err, &networkErr) || strings.Contains(err.Error(), "fixture-session-token") {
			t.Fatalf("error = %v, want redacted NetworkError", err)
		}
		if got := requests.Load(); got != 1 {
			t.Fatalf("single-use exchange sent %d times, want 1", got)
		}
	}
}

// resetCompletedHTTP2Streams resets each fully received request, after its token
// could have been consumed, using a protocol error that permits transport replay.
func resetCompletedHTTP2Streams(conn net.Conn, requests *atomic.Int32) {
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	write := func(frameType, flags byte, stream uint32, payload []byte) {
		header := []byte{byte(len(payload) >> 16), byte(len(payload) >> 8), byte(len(payload)), frameType, flags, 0, 0, 0, 0}
		binary.BigEndian.PutUint32(header[5:], stream)
		_, _ = conn.Write(append(header, payload...))
	}
	if _, err := io.ReadFull(conn, make([]byte, len("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"))); err != nil {
		return
	}
	write(0x4, 0, 0, nil)
	for {
		header := make([]byte, 9)
		if _, err := io.ReadFull(conn, header); err != nil {
			return
		}
		if _, err := io.ReadFull(conn, make([]byte, int(header[0])<<16|int(header[1])<<8|int(header[2]))); err != nil {
			return
		}
		frameType, flags, stream := header[3], header[4], binary.BigEndian.Uint32(header[5:])&0x7fffffff
		switch {
		case frameType == 0x4 && flags&0x1 == 0:
			write(0x4, 0x1, 0, nil)
		case (frameType == 0x0 || frameType == 0x1) && flags&0x1 != 0:
			requests.Add(1)
			write(0x3, 0, stream, []byte{0, 0, 0, 0x1})
		}
	}
}
