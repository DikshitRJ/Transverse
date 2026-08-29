package handlers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"transverse/internal/models"
	"transverse/internal/services/ingest"
)

type IngestHandler struct {
	svc *ingest.Service
}

func NewIngestHandler(svc *ingest.Service) *IngestHandler {
	return &IngestHandler{svc: svc}
}

func parseRecords[T any](body io.Reader, out *[]T) ([]error, error) {
	var parseErrs []error
	b, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return nil, nil
	}

	// If it starts with '[', it's a JSON array
	if b[0] == '[' {
		if err := json.Unmarshal(b, out); err != nil {
			// If JSON array is malformed, we can't easily isolate the bad record without custom streaming parsing.
			// The instruction specifically emphasizes NDJSON per-record errors: "malformed NDJSON records are rejected individually"
			return nil, err
		}
		return nil, nil
	}

	// Otherwise, it's NDJSON
	scanner := bufio.NewScanner(bytes.NewReader(b))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var record T
		if err := json.Unmarshal(line, &record); err != nil {
			parseErrs = append(parseErrs, fmt.Errorf("line %d malformed JSON: %v", lineNum, err))
			continue
		}
		*out = append(*out, record)
	}
	return parseErrs, scanner.Err()
}

func (h *IngestHandler) IngestTutorials(w http.ResponseWriter, r *http.Request) {
	var records []models.TutorialIngestRecord
	parseErrs, err := parseRecords(r.Body, &records)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to parse JSON body: " + err.Error()})
		return
	}

	errs := h.svc.IngestTutorials(r.Context(), records)
	
	// Combine parse errors and ingest errors
	errs = append(parseErrs, errs...)
	
	if len(errs) > 0 {
		var errStrings []string
		for _, err := range errs {
			errStrings = append(errStrings, err.Error())
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMultiStatus) // 207
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Some records failed to ingest",
			"errors":  errStrings,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "success"})
}

func (h *IngestHandler) IngestRoadmaps(w http.ResponseWriter, r *http.Request) {
	var records []models.RoadmapTemplateIngestRecord
	parseErrs, err := parseRecords(r.Body, &records)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to parse JSON body: " + err.Error()})
		return
	}

	errs := h.svc.IngestRoadmapTemplates(r.Context(), records)
	
	// Combine parse errors and ingest errors
	errs = append(parseErrs, errs...)

	if len(errs) > 0 {
		var errStrings []string
		for _, err := range errs {
			errStrings = append(errStrings, err.Error())
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMultiStatus)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Some records failed to ingest",
			"errors":  errStrings,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "success"})
}
