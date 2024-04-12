package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	//srv := NewServer(":8080")
	//
	//pingRoute := NewRoute("GET", "/ping", ping)
	//
	//srv.HandleRoute(pingRoute)
	//
	//chain := srv.Chain(Logger)
	//
	//srv.Attach(chain)
	//srv.Run()

	router := http.NewServeMux()

	router.HandleFunc("GET /ping", ping)

	chain := ChainMiddleMan(Logger)

	v1 := http.NewServeMux()
	http.Handle("/v1/", http.StripPrefix("/v1", router))

	v2 := http.NewServeMux()
	http.Handle("/v2/", http.StripPrefix("/v2", router))

	server := http.Server{
		Addr:    ":8080",
		Handler: chain(v1, v2),
	}

	server.Handler = chain(v2)

	fmt.Println("Server listening on port :8080")
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf(err.Error())
	}
}

func Chain(middleware ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middleware) - 1; i >= 0; i-- {
			next = middleware[i](next)
		}
		return next
	}
}

func ChainMiddleMan(middlewares ...Middleware) MiddleMan {
	return func(handlers ...http.Handler) http.Handler {
		var idx int
		for i := len(middlewares) - 1; i >= 0; i-- {
			idx = i
			handlers[i] = middlewares[i](handlers[i])
		}
		return handlers[idx]
	}
}

type Response struct {
	Message   string         `json:"message"`
	Data      map[string]any `json:"data"`
	Timestamp int64          `json:"timestamp"`
}

func NewResponse(message string) Response {
	return Response{Message: message, Timestamp: time.Now().Unix()}
}

func NewResponseWithData(message string, data map[string]any) Response {
	response := NewResponse(message)
	response.Data = data
	return response
}

type Data map[string]any

func (m Data) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{
		Space: "",
		Local: "map",
	}
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	for key, value := range m {
		elem := xml.StartElement{
			Name: xml.Name{Space: "", Local: key},
			Attr: []xml.Attr{},
		}
		if err := e.EncodeElement(value, elem); err != nil {
			return err
		}
	}

	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

func ping(w http.ResponseWriter, r *http.Request) {

	response := NewResponseWithData("Success", Data{
		"value": "pong",
	})

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Fatalf(err.Error())
	}
}

type Server struct {
	Addr     string
	Handlers []http.Handler
	router   *http.ServeMux
	muxers   *[]http.ServeMux
	server   http.Server
	chain    Middleware
}

type Route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
}

func NewRoute(method, path string, handler http.HandlerFunc) *Route {
	return &Route{Method: method, Path: path, Handler: handler}
}

func (s *Server) HandleRoute(routes ...*Route) {
	for _, route := range routes {
		s.Handlers = append(s.Handlers, route.Handler)
	}
}

func NewServer(addr string) *Server {
	return &Server{Addr: addr, router: http.NewServeMux()}
}

func (s *Server) Run() {
	s.server = http.Server{
		Addr:    s.Addr,
		Handler: s.router,
	}
	err := s.server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func (s *Server) HandleFunc(pattern string, handler http.HandlerFunc) {
	s.router.HandleFunc(pattern, handler)
}

func (s *Server) Attach(middleware Middleware) {
	s.chain = middleware
}

type Middleware func(http.Handler) http.Handler

type MiddleMan func(...http.Handler) http.Handler

func (s *Server) Chain(middleware ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middleware) - 1; i >= 0; i-- {
			next = middleware[i](next)
		}
		return next
	}
}

type LogWriter struct {
	request http.Request
	writer  http.ResponseWriter
	status  int
}

func NewLogWriter(r http.Request, w http.ResponseWriter, status int) *LogWriter {
	return &LogWriter{request: r, writer: w, status: status}
}

func (w *LogWriter) Write() {
	start := time.Now()
	log.Printf("Received request for %s", w.request.URL)
	log.Println(w.status, w.request.Method, w.request.URL.Path, time.Since(start))
	log.Printf("Request for %s took %s", w.request.URL, time.Since(start))
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wr := NewLogWriter(*r, w, http.StatusOK)
		next.ServeHTTP(wr.writer, r)
		wr.Write()
	})
}
