package tldr

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/cixtor/readability"
	"github.com/mr-joshcrane/oracle"
)

func TLDR(o *oracle.Oracle, url string) (string, error) {
	o.SetPurpose("Please summarise the provided text as best you can. The shorter the better.")
	content, err := GetContent(url)
	if err != nil {
		return "", err
	}
	return o.Ask(content)
}

// GetContent takes a url, checks if it is a file or URL, and returns the
// contents of the file or the text of the URL.
func GetContent(path string) (msg string, err error) {
	_, err = url.ParseRequestURI(path)
	if err != nil {
		path = "https://" + path
	}
	resp, err := http.Get(path)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	r := readability.New()
	article, err := r.Parse(resp.Body, path)
	if err != nil {
		return "", err
	}
	return article.TextContent, nil
}

type TLDRServer struct {
	oracle     *oracle.Oracle
	httpServer *http.Server
}

func NewTLDRServer(o *oracle.Oracle, addr string) *TLDRServer {
	s := &TLDRServer{
		oracle: o,
		httpServer: &http.Server{
			Addr: addr,
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/chat/", s.chatHandler)
	mux.HandleFunc("/", s.indexHandler)

	s.httpServer.Handler = mux
	return s
}

func (s *TLDRServer) indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "TLDR")
}

func (s *TLDRServer) chatHandler(w http.ResponseWriter, r *http.Request) {
	req, err := httputil.DumpRequest(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println(string(req))
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	err = r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	url := r.FormValue("summaryUrl")

	fmt.Println("url: ", url)
	summary, err := TLDR(s.oracle, url)
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusBadGateway)
	}
	htmlFragment := fmt.Sprintf("<div class='results'><h3><a href='%s' target='_blank'>%s</a></h3>%s</div>", url, url, summary)

	fmt.Fprintf(w, "%s", htmlFragment)

}

func (s *TLDRServer) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

func (s *TLDRServer) Shutdown() error {
	return s.httpServer.Shutdown(nil)
}

// mux := http.NewServeMux()
