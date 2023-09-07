package tldr_test

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mr-joshcrane/oracle"
	"github.com/mr-joshcrane/tldr"
)

func TestChatHandlerAcceptsPost(t *testing.T) {
	t.Parallel()
	addr := newTestTLDRServer(t)
	fmt.Println(addr)
	v := url.Values{
		"summaryUrl": []string{"https://example.com"},
	}
	resp, err := http.PostForm("http://"+addr+"/api/chat/", v)
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
	if !bytes.Contains(data, []byte("A summary of the article")) {
		t.Fatalf("Expected %s to contain %s", data, "https://example.com")
	}
}

func TestSplit(t *testing.T) {
	t.Parallel()
	longInput := "some long text"
	got := tldr.Split(longInput, 4)
	want := []string{"some", "long", "text"}
	if !cmp.Equal(got, want) {
		t.Fatalf("Expected %v, got %v", want, got)
	}

}

func newTestTLDRServer(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Expected no error opening listener, got %v", err)
	}

	addr := l.Addr().String()
	l.Close()
	o := oracle.NewOracle("dummy-key", oracle.WithDummyClient("A summary of the article"))
	srv := tldr.NewTLDRServer(o, addr)
	go func() {
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed && err != nil {
			panic(err)
		}
	}()
	t.Cleanup(func() { _ = srv.Shutdown() })
	return addr
}
