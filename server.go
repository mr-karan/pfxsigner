package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/knadh/pfxsigner/internal/processor"
	"github.com/urfave/cli"
)

type httpResp struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// initServer initializes CLI mode.
func initServer(c *cli.Context) error {
	r := chi.NewRouter()
	r.Post("/document", handleDocument)

	// HTTP Server.
	srv := &http.Server{
		Addr:         c.String("address"),
		ReadTimeout:  c.Duration("timeout"),
		WriteTimeout: c.Duration("timeout"),
		Handler:      r,
	}

	logger.Printf("starting server on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		logger.Fatalf("couldn't start server: %v", err)
	}
	return nil
}

// handleDocument handles an HTTP document signing request.
func handleDocument(w http.ResponseWriter, r *http.Request) {
	// Read the JSON request payload from the 'request' field.
	// If it's empty, use the default props.
	var (
		props processor.SignProps
		reqB  = []byte(r.FormValue("props"))
	)

	if len(reqB) > 0 {
		pr, err := parseProps(reqB)
		if err != nil {
			logger.Printf("error reading JSON `request`: %v", err)
			sendErrorResponse(w, fmt.Sprintf("Error reading JSON `request`: %v", err),
				http.StatusBadRequest, nil)
		}
		props = pr
	} else {
		props = proc.GetProps()
	}

	// Get the file.
	file, _, err := r.FormFile("file")
	if err != nil {
		logger.Printf("invalid file in the `file` field: %v", err)
		sendErrorResponse(w, "Invalid file in the `file` field.",
			http.StatusBadRequest, nil)
		return
	}

	// Sign the document.
	out, err := proc.ProcessDoc(props, "", file)
	if err != nil {
		logger.Printf("error processing PDF: %v", err)
		sendErrorResponse(w, "Error processing document.",
			http.StatusInternalServerError, nil)
		return
	}

	w.Header().Set("content-type", "application/pdf")
	w.Write(out)
}

// sendErrorResponse sends a JSON envelope to the HTTP response.
func sendResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	out, err := json.Marshal(httpResp{Status: "success", Data: data})
	if err != nil {
		sendErrorResponse(w, "Internal Server Error", http.StatusInternalServerError, nil)
		return
	}
	w.Write(out)
}

// sendErrorResponse sends a JSON error envelope to the HTTP response.
func sendErrorResponse(w http.ResponseWriter, message string, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)

	resp := httpResp{Status: "error",
		Message: message,
		Data:    data}
	out, _ := json.Marshal(resp)
	w.Write(out)
}
