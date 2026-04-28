package handlers

import (
	crand "crypto/rand"
	"encoding/hex"
	"math/rand/v2"
	"net/http"
	"strconv"

	"github.com/alfonso/go-laravelcloud/internal/respond"
)

var quotes = []string{
	"Plans are worthless, but planning is everything.",
	"Premature optimization is the root of all evil.",
	"Simple is better than complex.",
	"Make it work, make it right, make it fast.",
	"Talk is cheap. Show me the code.",
	"There are only two hard things in computer science: cache invalidation and naming things.",
	"The best code is no code at all.",
	"Programs must be written for people to read, and only incidentally for machines to execute.",
}

func Random(w http.ResponseWriter, r *http.Request) {
	n := 16
	if v := r.URL.Query().Get("bytes"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 1024 {
			n = parsed
		}
	}
	buf := make([]byte, n)
	if _, err := crand.Read(buf); err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to read random bytes")
		return
	}
	respond.JSON(w, http.StatusOK, map[string]any{
		"bytes":  n,
		"hex":    hex.EncodeToString(buf),
		"int":    rand.Int64(),
		"float":  rand.Float64(),
		"dice":   rand.IntN(6) + 1,
		"coin":   []string{"heads", "tails"}[rand.IntN(2)],
	})
}

func Quote(w http.ResponseWriter, r *http.Request) {
	respond.JSON(w, http.StatusOK, map[string]string{
		"quote": quotes[rand.IntN(len(quotes))],
	})
}
