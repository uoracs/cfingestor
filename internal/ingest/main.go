package ingest

import (
	"log/slog"
	"os"
)

var IngestDirectory = "/var/run/cfingestor"

func EnsureIngestDirectory() {
    slog.Info("Ensuring ingest directory")
    _ = os.Mkdir(IngestDirectory, 0755)
    // if err != nil {
    //     slog.Error("Error creating ingest directory: %v", err)
    // }
}
