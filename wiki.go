package main

import (
	"errors"
	"html"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
)

var templates = template.Must(template.ParseFiles("tpl/edit.html", "tpl/view.html"))

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

var linkPattern = regexp.MustCompile("\\[[a-zA-Z0-9]+\\]")

type Page struct {
	Title string
	Body  []byte
}

type PageView struct {
	Title    string
	Body     string
	HTMLBody template.HTML
}

func getTitle(w http.ResponseWriter, r *http.Request) (string, error) {
	m := validPath.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return "", errors.New("invalid page title")
	}
	return m[2], nil
}

func (p *Page) save() error {
	filename := "data/" + p.Title + ".txt"
	return os.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := "data/" + title + ".txt"
	body, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func makePageView(p *Page) PageView {
	body := string(p.Body)
	return PageView{
		Title:    p.Title,
		Body:     body,
		HTMLBody: renderBody(body),
	}
}

func renderBody(body string) template.HTML {
	escaped := html.EscapeString(body)
	linked := linkPattern.ReplaceAllFunc([]byte(escaped), func(match []byte) []byte {
		title := string(match[1 : len(match)-1])
		return []byte("<a href=\"/view/" + title + "\">" + title + "</a>")
	})
	return template.HTML(linked)
}

func renderTemplate(w http.ResponseWriter, tpl string, p *Page) {
	data := makePageView(p)
	err := templates.ExecuteTemplate(w, tpl+".html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/view/FrontPage", http.StatusFound)
}

func main() {
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
