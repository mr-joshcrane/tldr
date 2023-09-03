package tldr_test

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"testing"

	"github.com/mr-joshcrane/assistant"
	"github.com/mr-joshcrane/oracle"
)

func TestChatHandlerAcceptsPost(t *testing.T) {
	t.Parallel()
	addr := newTestTLDRServer(t)
	fmt.Println(addr)
	req, err := http.NewRequest(http.MethodPost, "http://"+addr+"/api/chat", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	dump, err := httputil.DumpRequest(req, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(dump))
	resp, err := http.Post("http://"+addr+"/api/chat/", "text/plain", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %v, got %v", http.StatusOK, resp.StatusCode)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !bytes.Contains(data, []byte("https://example.com")) {
		t.Fatalf("Expected %s to contain %s", data, "https://example.com")
	}
}

func newTestTLDRServer(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Expected no error opening listener, got %v", err)
	}
	addr := l.Addr().String()
	l.Close()
	o := oracle.NewOracle("dummy-key")
	srv := assistant.NewTLDRServer(o, addr)
	go func() {
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed && err != nil {
			panic(err)
		}
	}()
	t.Cleanup(func() { srv.Shutdown() })
	return addr
}
