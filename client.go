package websocket

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gbrlsnchs/uuid"
	"github.com/gbrlsnchs/websocket/internal"
)

// Open creates a WebSocket client.
func Open(address string) (*WebSocket, error) {
	uri, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	switch uri.Scheme {
	case "ws":
		uri.Scheme = "http"
	case "wss":
		uri.Scheme = "https"
	default:
		return nil, fmt.Errorf("websocket: unsupported protocol %s", uri.Scheme)
	}

	r, err := http.NewRequest(http.MethodGet, uri.String(), nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Upgrade", internal.UpgradeHeader)
	r.Header.Set("Connection", internal.ConnectionHeader)
	r.Header.Set("Sec-WebSocket-Version", internal.SecWebSocketVersionHeader)
	guid, err := uuid.GenerateV4(nil)
	if err != nil {
		return nil, err
	}
	encKey := base64.StdEncoding.EncodeToString(guid[:])
	r.Header.Set("Sec-WebSocket-Key", encKey)

	d := &net.Dialer{Timeout: 15 * time.Second}
	conn, err := d.Dial("tcp", uri.Host)
	if err != nil {
		return nil, err
	}
	b, err := httputil.DumpRequestOut(r, true)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if _, err = conn.Write(b); err != nil {
		conn.Close()
		return nil, err
	}
	rd := bufio.NewReader(conn)
	rr, err := http.ReadResponse(rd, r)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if rr.StatusCode != http.StatusSwitchingProtocols {
		conn.Close()
		return nil, errors.New("websocket: client did not receive 101 status response")
	}
	if err = validateServerHeaders(rr.Header, encKey); err != nil {
		conn.Close()
		return nil, err
	}
	return newWebSocket(conn, true), nil
}

func validateServerHeaders(hdr http.Header, encKey string) error {
	switch {
	case strings.ToLower(hdr.Get("Upgrade")) != internal.UpgradeHeader:
		return internal.ErrUpgradeMismatch
	case strings.ToLower(hdr.Get("Connection")) != internal.ConnectionHeader:
		return internal.ErrConnectionMismatch
	}
	key, err := internal.ConcatKey(encKey)
	if err != nil {
		return err
	}
	srvKey := hdr.Get("Sec-WebSocket-Accept")
	dec, err := base64.StdEncoding.DecodeString(srvKey)
	if err != nil {
		return err
	}
	if string(key) != string(dec) {
		return internal.ErrSecWebSocketKeyMismatch
	}
	return nil
}
