package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"unicode"

	"github.com/cockroachdb/pebble"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

func jsonResponse(w http.ResponseWriter, body map[string]any, err error) {
	data := map[string]any{
		"body":   body,
		"status": "ok",
	}

	if err == nil {
		w.WriteHeader(http.StatusOK)
	} else {
		data["status"] = "error"
		data["error"] = err.Error()
		w.WriteHeader(http.StatusBadRequest)
	}
	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	err = enc.Encode(data)
	if err != nil {
		// TODO: set up panic handler?
		panic(err)
	}
}

type server struct {
	db   *pebble.DB // File location where data should be stored
	port string
}

func newServer(database string, port string) (*server, error) {
	s := server{db: nil, port: port}
	var err error
	s.db, err = pebble.Open(database, &pebble.Options{})
	return &s, err
}

func (s server) addDocument(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	dec := json.NewDecoder(r.Body)
	var document map[string]any
	err := dec.Decode(&document)
	if err != nil {
		jsonResponse(w, nil, err)
		return
	}

	// New unique id for the document
	id := uuid.New().String()

	bs, err := json.Marshal(document)
	if err != nil {
		jsonResponse(w, nil, err)
		return
	}
	err = s.db.Set([]byte(id), bs, pebble.Sync)
	if err != nil {
		jsonResponse(w, nil, err)
		return
	}

	jsonResponse(w, map[string]any{
		"id": id,
	}, nil)
}

type queryEquals struct {
	key   []string
	value string
}

type query struct {
	ands []queryEquals
}

func getPath(doc map[string]any, parts []string) (any, bool) {
	var docSegment any = doc
	for _, part := range parts {
		m, ok := docSegment.(map[string]any)
		if !ok {
			return nil, false
		}

		if docSegment, ok = m[part]; !ok {
			return nil, false
		}
	}

	return docSegment, true
}

func (q query) match(doc map[string]any) bool {
	for _, argument := range q.ands {
		value, ok := getPath(doc, argument.key)
		if !ok {
			return false
		}

		match := fmt.Sprintf("%v", value) == argument.value
		if !match {
			return false
		}

	}

	return true
}

// Handles either quoted strings or unquoted strings of only contiguous digits and letters
func lexString(input []rune, index int) (string, int, error) {
	if index >= len(input) {
		return "", index, nil
	}
	if input[index] == '"' {
		index++
		foundEnd := false

		var s []rune
		// TODO: handle nested quotes
		for index < len(input) {
			if input[index] == '"' {
				foundEnd = true
				break
			}

			s = append(s, input[index])
			index++
		}

		if !foundEnd {
			return "", index, fmt.Errorf("Expected end of quoted string")
		}

		return string(s), index + 1, nil
	}

	// If unquoted, read as much contiguous digits/letters as there are
	var s []rune
	var c rune
	// TODO: someone needs to validate there's not ...
	for index < len(input) {
		c = input[index]
		if !(unicode.IsLetter(c) || unicode.IsDigit(c) || c == '.') {
			break
		}
		s = append(s, c)
		index++
	}

	if len(s) == 0 {
		return "", index, fmt.Errorf("No string found")
	}

	return string(s), index, nil
}

// E.g. q=a.b:12
func parseQuery(q string) (*query, error) {
	if q == "" {
		return &query{}, nil
	}

	i := 0
	var parsed query
	var qRune = []rune(q)
	for i < len(qRune) {
		// Eat whitespace
		for unicode.IsSpace(qRune[i]) {
			i++
		}

		key, nextIndex, err := lexString(qRune, i)
		if err != nil {
			return nil, fmt.Errorf("Expected valid key, got [%s]: `%s`", err, q[nextIndex:])
		}

		// Expect =, eventually some other operator
		if q[nextIndex] != ':' {
			// TODO: return an error
			return nil, fmt.Errorf("Expected colon at %d, got: `%s`", nextIndex, q[nextIndex:])
		}
		i = nextIndex
		i++

		value, nextIndex, err := lexString(qRune, i)
		if err != nil {
			return nil, fmt.Errorf("Expected valid value, got [%s]: `%s`", err, q[nextIndex:])
		}
		i = nextIndex

		argument := queryEquals{key: strings.Split(key, "."), value: value}
		parsed.ands = append(parsed.ands, argument)
	}

	return &parsed, nil
}

func (s server) searchDocuments(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	q, err := parseQuery(r.URL.Query().Get("q"))
	if err != nil {
		jsonResponse(w, nil, err)
		return
	}

	iter := s.db.NewIter(nil)
	defer iter.Close()
	var documents []any
	for iter.First(); iter.Valid(); iter.Next() {
		valBytes, closer, err := s.db.Get(iter.Key())
		if err != nil {
			jsonResponse(w, nil, err)
			return
		}

		var document map[string]any
		err = json.Unmarshal(valBytes, &document)
		if err != nil {
			jsonResponse(w, nil, err)
			return
		}

		err = closer.Close()
		if err != nil {
			jsonResponse(w, nil, err)
			return
		}

		if q.match(document) {
			documents = append(documents, map[string]any{
				"id":   string(iter.Key()),
				"body": document,
			})
		}
	}

	jsonResponse(w, map[string]any{"documents": documents}, nil)
}

func (s server) getDocument(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")

	val, closer, err := s.db.Get([]byte(id))
	if err != nil {
		jsonResponse(w, nil, err)
		return
	}

	var document map[string]any
	err = json.Unmarshal(val, &document)
	if err != nil {
		jsonResponse(w, nil, err)
		return
	}

	// Don't forget this apparently!
	err = closer.Close()
	if err != nil {
		jsonResponse(w, nil, err)
		return
	}

	jsonResponse(w, map[string]any{
		"document": document,
	}, nil)
}

func (s server) deleteDocument(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	panic("Delete document not implemented")
}

func main() {
	s, err := newServer("docdb.data", "8080")
	if err != nil {
		log.Fatal(err)
	}
	defer s.db.Close()

	router := httprouter.New()
	router.POST("/docs", s.addDocument)
	router.GET("/docs", s.searchDocuments)
	router.GET("/docs/:id", s.getDocument)
	router.DELETE("/docs/:id", s.deleteDocument)

	log.Println("Listening on " + s.port)
	log.Fatal(http.ListenAndServe(":"+s.port, router))
}
