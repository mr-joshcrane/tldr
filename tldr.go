package tldr

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"

	"github.com/cixtor/readability"
	"github.com/mr-joshcrane/oracle"
)

//go:embed templates/card.html
var cardsHTML string

//go:embed templates/error.html
var errorHTML string

//go:embed templates/index.html
var indexHTML string

//go:embed static/*
var staticFiles embed.FS

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
	templates  map[string]*template.Template
}

func NewTLDRServer(o *oracle.Oracle, addr string) *TLDRServer {
	templates := map[string]*template.Template{
		"card":  template.Must(template.New("card").Parse(cardsHTML)),
		"index": template.Must(template.New("index").Parse(indexHTML)),
		"error": template.Must(template.New("error").Parse(errorHTML)),
	}
	s := &TLDRServer{
		oracle: o,
		httpServer: &http.Server{
			Addr: addr,
		},
		templates: templates,
	}
	mux := http.NewServeMux()
	assets := http.FS(staticFiles)
	mux.Handle("/static/", http.FileServer(assets))
	mux.HandleFunc("/api/chat/", s.chatHandler)
	mux.HandleFunc("/", s.indexHandler)

	s.httpServer.Handler = mux
	return s
}

func (s *TLDRServer) indexHandler(w http.ResponseWriter, r *http.Request) {
	err := s.templates["index"].Execute(w, nil)
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
		fmt.Fprintln(os.Stderr, err)
		s.templates["error"].Execute(w, err)
	}
}

func (s *TLDRServer) htmlFragment(w http.ResponseWriter, url string) error {
	summary, err := TLDR(s.oracle, url)
	if err != nil {
		return err
	}
	title, err := s.oracle.Ask(fmt.Sprintf("Given the summary, generate a title for this article: %s?", summary))
	if err != nil {
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
	return s.templates["card"].Execute(w, data)
}

func (s *TLDRServer) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

func (s *TLDRServer) Shutdown() error {
	return s.httpServer.Shutdown(context.TODO())
}
