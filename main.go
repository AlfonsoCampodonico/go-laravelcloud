package main

import (
	"log"
	"net/http"
)

const addr = ":8080"

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /hello", helloHandler)

	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

// helloHandler responds to GET /hello with a "hello world" payload.
//
// TODO: implement the response. Decisions to make:
//   - Plain text ("hello world") vs JSON ({"message": "hello world"})?
//   - Set Content-Type header explicitly?
//   - Any status code other than 200?
//
// JSON is more conventional for an API and easier for clients to parse;
// plain text is simpler and avoids importing encoding/json.
func helloHandler(w http.ResponseWriter, r *http.Request) {
	// TODO(you): write 3-5 lines here
}
