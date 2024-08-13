package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/vertexai/genai"
)

func main() {
	ctx := context.Background()
	var projectId string
	var err error
	projectId = os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectId == "" {
		projectId, err = metadata.ProjectIDWithContext(ctx)
		if err != nil {
			return
		}
	}
	var client *genai.Client
	client, err = genai.NewClient(ctx, projectId, "us-central1")
	if err != nil {
		return
	}
	defer client.Close()
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(group []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				return slog.Attr{Key: "severity", Value: a.Value}
			}
			if a.Key == slog.MessageKey {
				return slog.Attr{Key: "message", Value: a.Value}
			}
			return slog.Attr{Key: a.Key, Value: a.Value}
		},
	}

	jsonHandler := slog.NewJSONHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(jsonHandler))
	model := client.GenerativeModel("gemini-1.5-flash-001")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		animal := r.URL.Query().Get("animal")
		if animal == "" {
			animal = "dog"
		}

		resp, err := model.GenerateContent(
			ctx,
			genai.Text(
				fmt.Sprintf("Give me 10 fun facts about %s. Return the results as HTML without markdown backticks.", animal)),
		)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
			htmlContent := resp.Candidates[0].Content.Parts[0]
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, htmlContent)
		}
		jsonBytes, err := json.Marshal(resp)
		if err != nil {
			slog.Error("Failed to marshal response to JSON", "error", err)
		} else {
			jsonString := string(jsonBytes)
			slog.Debug("Complete response content", "json_response", jsonString)
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	http.ListenAndServe(":"+port, nil)
}
