package web

import (
	"encoding/json"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// StatusResponseWriter holds the status code that the server wrote to the client.
// This allows upstream logging of the response
type StatusResponseWriter struct {
	status int
	writer http.ResponseWriter
}

func (s *StatusResponseWriter) Header() http.Header {
	return s.writer.Header()
}

func (s *StatusResponseWriter) Write(b []byte) (int, error) {
	return s.writer.Write(b)
}

func (s *StatusResponseWriter) WriteHeader(status int) {
	s.status = status
	s.writer.WriteHeader(status)
}

// Error type for all HTTP related functions
type Error struct {
	Code    int    `json:"code"`
	Status  int    `json:"-"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return e.Message
}

// Handler function types can process a request, returning the response status and an optional error.
// Functions of this type are designed to be wrapped to allow upstream processing of the error (e.g. to
// return a JSON response body with associated error message).
type Handler func(w http.ResponseWriter, r *http.Request) *Error

// NewErrorHandler will wrap a given Handler type to encode any downstream Error that is created
func NewErrorHandler(h Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)

		if err == nil {
			return
		}

		w.WriteHeader(err.Status)

		if e := json.NewEncoder(w).Encode(err); e != nil {
			log.Errorf("failed to write error to client: error = %s", e)
		}
	}
}

func NewCorsHandler(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Credentials", "true")
		w.Header().Add("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, Origin, User-Agent, Cache-Control, Keep-Alive, If-Modified-Since, If-None-Match")
		w.Header().Add("Access-Control-Allow-Methods", "GET, HEAD")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Expose-Headers", "Content-Type, Cache-Control, ETag, Expires, Last-Modified, Content-Length")
		w.Header().Add("Access-Control-Max-Age", "3600")

		// preflight check
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		h.ServeHTTP(w, r)
	}
}

// NewStatusHandler creates a service specific handler that can be used for server updates required by HAProxy.
func NewStatusHandler(serviceID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		if r.Method == http.MethodOptions {
			return
		}

		err := json.NewEncoder(w).Encode(map[string]interface{}{
			"serviceID": serviceID,
		})

		if err != nil {
			log.Errorf("failed to send status: error = %s", err)
		}
	}

}

// NewRequestHandler wraps a handler func to provide standard request logging and setup
func NewRequestHandler(sid string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now().UTC().UnixNano()

		sw := &StatusResponseWriter{status: http.StatusOK, writer: w}
		h.ServeHTTP(sw, r)
		delta := time.Now().UTC().UnixNano() - start
		log.WithFields(log.Fields{
			"serverid":  sid,
			"addr":      r.RemoteAddr,
			"method":    r.Method,
			"uri":       r.RequestURI,
			"userAgent": r.UserAgent(),
			"startTime": start / 1000,
			"deltaTime": delta / 1000,
			"status":    sw.status,
		}).Info("Request handled")
	})
}
