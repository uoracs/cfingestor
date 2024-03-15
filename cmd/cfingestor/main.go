package main

import (
	"log/slog"
	"net/http"

	"github.com/uoracs/cfingestor/internal/activedirectory"
	"github.com/uoracs/cfingestor/internal/coldfront"
	"github.com/uoracs/cfingestor/internal/ingest"
)

func main() {
	ingest.EnsureIngestDirectory()
	mux := http.NewServeMux()

	// activedirectory
	mux.HandleFunc("POST /manifest", activedirectory.ManifestPOSTHandler)
	mux.HandleFunc("GET /manifest", activedirectory.ManifestGETHandler)

	// coldfront
	mux.HandleFunc("POST /ingest", coldfront.IngestPOSTHandler)
	mux.HandleFunc("GET /ingest", coldfront.IngestGETHandler)

	slog.Info("Starting server on port 8090")
	http.ListenAndServe(":8090", mux)
}
