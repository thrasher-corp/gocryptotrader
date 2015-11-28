package main

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
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

	tmpl, err := template.ParseFiles("web/index.html", "web/header.html", "web/footer.html")
	if err != nil {
		ServerHTTPError(w, err)
		return
	}

	tmplValues := map[string]interface{}{"title": "Home"}
	tmpl.Execute(w, tmplValues)
	if err != nil {
		ServerHTTPError(w, err)
		return
	}
}
