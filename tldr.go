package tldr

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"

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
	tmpl       *template.Template
}

func NewTLDRServer(o *oracle.Oracle, addr string) *TLDRServer {
	s := &TLDRServer{
		oracle: o,
		httpServer: &http.Server{
			Addr: addr,
		},
		tmpl: template.Must(template.ParseFiles("templates/card.html")),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/chat/", s.chatHandler)
	mux.HandleFunc("/", s.indexHandler)

	s.httpServer.Handler = mux
	return s
}

func (s *TLDRServer) indexHandler(w http.ResponseWriter, r *http.Request) {
	err := template.Must(template.ParseFiles("templates/index.html")).Execute(w, nil)
	if err != nil {
		// Log something here?
		fmt.Fprintln(os.Stderr, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}

func (s *TLDRServer) chatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	url := r.FormValue("summaryUrl")
	err = s.htmlFragment(w, url)
	if err != nil {
		// Log something here?
		fmt.Fprintln(os.Stderr, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}

func (s *TLDRServer) htmlFragment(w http.ResponseWriter, url string) error {
	summary, err := TLDR(s.oracle, url)
	if err != nil {
		// Log something here?
		return err
	}
	title, err := s.oracle.Ask(fmt.Sprintf("Given the summary, generate a title for this article: %s?", summary))
	if err != nil {
		// Log something here?
		return err
	}
	data := struct {
		URL     string
		Title   string
		Summary string
	}{
		URL:     url,
		Title:   title,
		Summary: summary,
	}
	return s.tmpl.Execute(w, data)
}

func (s *TLDRServer) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

func (s *TLDRServer) Shutdown() error {
	return s.httpServer.Shutdown(context.TODO())
}
