/* render Library is a simple library to marshall go objects.
At the moment it supports json and handles Accept headers
*/
package render

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/juju/errgo"
)

var DefaultJSON = new(JSON)

func init() {
	Register("application/json", new(JSON))
	Register("*/*", new(JSON))
}

// Renderer
type Renderer interface {
	Render(http.ResponseWriter, *http.Request, int, interface{}) error
}

var register = map[string]Renderer{}

var registeredContentType []string

// register a content type handler
func Register(contentType string, handler Renderer) {
	if prevHandler, ok := register[contentType]; ok {
		log.Printf("warning: content handler already registered %s, %v", contentType, prevHandler)
	}
	register[contentType] = handler
	registeredContentType = append(registeredContentType, contentType)
}

// Render selects a valid Renderer based on the accept header of the request
func Render(w http.ResponseWriter, r *http.Request, httpStatus int, obj interface{}) error {
	accepts := r.Header.Get("Accept")

	for _, accept := range strings.Split(accepts, ",") {
		contentType := strings.Split(accept, ";")[0]
		if handler, ok := register[contentType]; ok {
			return handler.Render(w, r, httpStatus, obj)
		}
	}

	return DefaultJSON.Render(w, r,
		http.StatusNotAcceptable,
		mkerror(1, fmt.Sprint("Accept header must be set to one of ", strings.Join(registeredContentType, ","))))
}

// JSON implements the render.Renderer
type JSON struct{}

// Render implements render.Renderer
func (s *JSON) Render(w http.ResponseWriter, r *http.Request, httpCode int, obj interface{}) error {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	w.Header().Set("Date", time.Now().Format(time.RFC1123Z))

	w.WriteHeader(httpCode)

	if err := json.NewEncoder(w).Encode(obj); err != nil {
		return errgo.Mask(err)
	}

	return nil
}

// Status
type StatusRenderer interface {
	Renderer
	// 2XX
	StatusOK(http.ResponseWriter, *http.Request, interface{}) error
	StatusCreated(http.ResponseWriter, *http.Request, interface{}) error
	StatusAccepted(http.ResponseWriter, *http.Request, interface{}) error

	// 4XX
	StatusBadRequest(http.ResponseWriter, *http.Request, ...interface{}) error
	StatusUnauthorized(http.ResponseWriter, *http.Request, ...interface{}) error
	StatusForbidden(http.ResponseWriter, *http.Request, ...interface{}) error
	StatusNotFound(http.ResponseWriter, *http.Request, ...interface{}) error
	StatusMethodNotAllowed(http.ResponseWriter, *http.Request, ...interface{}) error

	// StatusTooManyRequest(http.ResponseWriter, *http.Request, ...interface{}) error

	// 5XX
	//	Error(http.ResponseWriter, *http.Request, int, ...interface{}) error
}

// this struct uses the module's render
type moduleRender struct{}

func (h *moduleRender) Render(w http.ResponseWriter, r *http.Request, httpStatus int, obj interface{}) error {
	return Render(w, r, httpStatus, obj)
}

func New(r Renderer) StatusRenderer {
	return &handlerHelper{r}
}

type Error struct {
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return e.Message
}

func mkerror(callDepth int, msg string) error {
	return &Error{msg}
}

type handlerHelper struct {
	r Renderer
}

// Renderer
func (h *handlerHelper) Render(w http.ResponseWriter, r *http.Request, httpStatus int, obj interface{}) error {
	return h.Render(w, r, httpStatus, obj)
}

// 200
func (h *handlerHelper) StatusOK(w http.ResponseWriter, r *http.Request, obj interface{}) error {
	return h.r.Render(w, r, http.StatusOK, obj)
}

// 201
func (h *handlerHelper) StatusCreated(w http.ResponseWriter, r *http.Request, obj interface{}) error {
	return h.r.Render(w, r, http.StatusCreated, obj)
}

// 202
func (h *handlerHelper) StatusAccepted(w http.ResponseWriter, r *http.Request, obj interface{}) error {
	return h.r.Render(w, r, http.StatusAccepted, obj)
}

// 400
func (h *handlerHelper) StatusBadRequest(w http.ResponseWriter, r *http.Request, args ...interface{}) error {
	return h.r.Render(w, r, http.StatusBadRequest, mkerror(1, fmt.Sprint(args)))
}

// 401
func (h *handlerHelper) StatusUnauthorized(w http.ResponseWriter, r *http.Request, args ...interface{}) error {
	return h.r.Render(w, r, http.StatusUnauthorized, mkerror(1, fmt.Sprint(args)))
}

// 402
func (h *handlerHelper) StatusForbidden(w http.ResponseWriter, r *http.Request, args ...interface{}) error {
	return h.r.Render(w, r, http.StatusForbidden, mkerror(1, fmt.Sprint(args)))
}

// 403
func (h *handlerHelper) StatusMethodNotAllowed(w http.ResponseWriter, r *http.Request, args ...interface{}) error {
	return h.r.Render(w, r, http.StatusMethodNotAllowed, mkerror(1, fmt.Sprint(args)))
}

// 404
func (h *handlerHelper) StatusNotFound(w http.ResponseWriter, r *http.Request, args ...interface{}) error {
	return h.r.Render(w, r, http.StatusNotFound, mkerror(1, fmt.Sprint(args)))
}

// 500
func (h *handlerHelper) InternalServerError(w http.ResponseWriter, r *http.Request, args ...interface{}) error {
	return h.r.Render(w, r, http.StatusInternalServerError, mkerror(1, fmt.Sprint(args)))
}

// 501
func (h *handlerHelper) ServiceUnavailable(w http.ResponseWriter, r *http.Request, args ...interface{}) error {
	return h.r.Render(w, r, http.StatusServiceUnavailable, mkerror(1, fmt.Sprint(args)))
}

// DefaultStatu
var DefaultStatusRenderer = handlerHelper{&moduleRender{}}

// 200
func StatusOK(w http.ResponseWriter, r *http.Request, obj interface{}) error {
	return DefaultStatusRenderer.StatusOK(w, r, obj)
}

// 201
func StatusCreated(w http.ResponseWriter, r *http.Request, obj interface{}) error {
	return DefaultStatusRenderer.StatusCreated(w, r, obj)
}

// 202
func StatusAccepted(w http.ResponseWriter, r *http.Request, obj interface{}) error {
	return DefaultStatusRenderer.StatusAccepted(w, r, obj)
}

// 400
func StatusBadRequest(w http.ResponseWriter, r *http.Request, args ...interface{}) error {
	return DefaultStatusRenderer.StatusBadRequest(w, r, args...)
}

// 401
func StatusUnauthorized(w http.ResponseWriter, r *http.Request, args ...interface{}) error {
	return DefaultStatusRenderer.StatusUnauthorized(w, r, args...)
}

// 402
func StatusForbidden(w http.ResponseWriter, r *http.Request, args ...interface{}) error {
	return DefaultStatusRenderer.StatusForbidden(w, r, args...)
}

// 403
func StatusMethodNotAllowed(w http.ResponseWriter, r *http.Request, args ...interface{}) error {
	return DefaultStatusRenderer.StatusMethodNotAllowed(w, r, args...)
}

// 404
func StatusNotFound(w http.ResponseWriter, r *http.Request, args ...interface{}) error {
	return DefaultStatusRenderer.StatusNotFound(w, r, args...)
}
