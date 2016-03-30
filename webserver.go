package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

type Page struct {
	Title            string
	StaticStylesheet template.HTML
	Body             []byte
	Error            string
}

const (
	coverCSS     = `<link rel="stylesheet" href="web/static/css/cover.css">`
	dashboardCSS = `<link rel="stylesheet" href="web/static/css/dashboard.css">`
)

func GetWebserverHost() string {
	host := SplitStrings(bot.config.Webserver.ListenAddress, ":")[0]
	if host == "" {
		return "localhost"
	}
	return host
}

func GetWebserverPort() int {
	portStr := SplitStrings(bot.config.Webserver.ListenAddress, ":")[1]
	port, _ := strconv.Atoi(portStr)
	return port
}

func StartWebserver() error {
	fs := http.FileServer(http.Dir("web"))
	http.Handle("/web/", http.StripPrefix("/web/", fs))
	http.HandleFunc("/", index)
	var err error
	go func() {
		err = http.ListenAndServe(bot.config.Webserver.ListenAddress, nil)
	}()
	return err
}

func ServerHTTPError(w http.ResponseWriter, err error) {
	log.Println(err)
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
}

func index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	switch r.URL.Path {
	case "/":
		renderTemplate(w, "index.html", readPage("/index"))
	case "/login":
		renderTemplate(w, "login.html", readPage(r.URL.Path))
	case "/logout":
		renderTemplate(w, "index.html", readPage("/index"))
	case "/dashboard-marketdepth":
		renderTemplate(w, "dashboard-marketdepth.html", readPage(r.URL.Path))
	case "/dashboard-ordermanagement":
		renderTemplate(w, "dashboard-ordermanagement.html", readPage(r.URL.Path))
	case "/dashboard-contact":
		renderTemplate(w, "dashboard-contact.html", readPage(r.URL.Path))
	case "/dashboard-settings":
		renderTemplate(w, "dashboard-settings.html", readPage(r.URL.Path))
	case "/dashboard-reports":
		renderTemplate(w, "dashboard-reports.html", readPage(r.URL.Path))
	default:
		w.WriteHeader(http.StatusNotFound)
		renderTemplate(w, "error.html", readPage("/error"))
	}
}

func readPage(client string) *Page {
	filename := "web/" + client[1:] + ".html"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println("Webserver: Failed to open file -- ", err, "client string is: ", client)
		return nil
	}
	stylesheet := setStylesheet(client)
	return &Page{Title: client[1:], StaticStylesheet: stylesheet, Body: body}
}

func setStylesheet(client string) template.HTML {
	if len(client) >= 10 {
		if client[:10] == "/dashboard" {
			return template.HTML(dashboardCSS)
		}
	}
	return template.HTML(coverCSS)
}

func renderTemplate(w http.ResponseWriter, pageName string, p *Page) {
	tmpl, err := template.ParseFiles("web/index.html", "web/header.html",
		"web/footer.html", "web/dashboard-marketdepth.html", "web/login.html",
		"web/dashboard-ordermanagement.html", "web/dashboard-reports.html",
		"web/dashboard-settings.html", "web/dashboard-contact.html", "web/error.html")

	if err != nil {
		log.Println("Webserver: Could not parsefile -- ", err)
		ServerHTTPError(w, err)
		return
	}
	err = tmpl.ExecuteTemplate(w, pageName, p)
	if err != nil {
		log.Println("Webserver: Could not execute template -- ", err)
	}
}
