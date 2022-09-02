package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.Handle("/client/", http.FileServer(http.Dir(".")))
	fmt.Println("Open site to access this demo")
	panic(http.ListenAndServe(":9876", nil))
}
