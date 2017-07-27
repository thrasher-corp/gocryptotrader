package main

import "net/http"

// Route is a sub type that holds the request routes
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

// Routes is an array of all the registered routes
type Routes []Route

var routes = Routes{}
