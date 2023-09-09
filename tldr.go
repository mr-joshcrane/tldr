package tldr

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/cixtor/readability"
	"github.com/mr-joshcrane/oracle"
)

//go:embed templates/*.html
var templates embed.FS

//go:embed static/*
var staticFiles embed.FS

func TLDR(o *oracle.Oracle, url string) (string, error) {
	o.SetPurpose("Please summarise the provided text as best you can. The shorter the better. If there is a general thesis statement, please provide it.")
	content, err := GetContent(url)
	if err != nil {
		return "", err
	}
	if len(content) >= 4096*3 {
		content, err = RecursiveSummary(o, content, 4096*3)
		if err != nil {
			return "", err
		}
	}
	return o.Ask(content)
}

func RecursiveSummary(o *oracle.Oracle, content string, maxLen int) (string, error) {
	var wg sync.WaitGroup
	contentSplit := Split(content, maxLen)
	if len(content) <= maxLen {
		return content, nil
	}
	chunkSummaries := make([]string, len(contentSplit))
	for i, s := range contentSplit {
		wg.Add(1)
		go func(i int, s string) {
			defer wg.Done()
			chunkSummary, _ := o.Ask(s)

			chunkSummaries[i] = chunkSummary

		}(i, s)
	}
	wg.Wait()
	return o.Ask(strings.Join(chunkSummaries, " "))
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
	s := &TLDRServer{
		oracle: o,
		httpServer: &http.Server{
			Addr: addr,
		},
		templates: make(map[string]*template.Template),
	}
	t, err := template.ParseFS(templates, "templates/*.html")
	if err != nil {
		panic(err)
	}
	s.templates["index"] = t.Lookup("index.html")
	s.templates["card"] = t.Lookup("card.html")
	s.templates["error"] = t.Lookup("error.html")

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

func Split(s string, n int) []string {
	var chunks []string
	for len(s) > n {
		chunks = append(chunks, s[:n])
		s = s[n:]
	}
	chunks = append(chunks, s)
	return chunks
}
