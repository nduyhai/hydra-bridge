package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
)

type PageData struct {
	LoginURL string
}

var loginTmpl *template.Template
var successTmpl *template.Template

func main() {
	// Parse the template
	var err error
	loginTmpl, err = template.New("login").Parse(loginTemplate)
	if err != nil {
		log.Fatal("Error parsing template:", err)
	}
	successTmpl, err = template.New("success").Parse(successTemplate)
	if err != nil {
		log.Fatal("Error parsing template:", err)
		return
	}
	// Set up a route
	http.HandleFunc("/", handleLogin)
	http.HandleFunc("/success", handleSuccess)
	http.HandleFunc("/exchange-token", handleTokenExchange)

	fmt.Println("Server starting on http://localhost:8091")
	log.Fatal(http.ListenAndServe(":8091", nil))
}
