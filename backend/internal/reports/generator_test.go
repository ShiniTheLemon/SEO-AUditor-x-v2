package reports

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestFetchPageSpeedBuildsExpectedRequestAndSummary(t *testing.T) {
	generator := NewGenerator(GeneratorConfig{
		HTTPClient: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Host != "pagespeedonline.googleapis.com" {
				t.Fatalf("unexpected request host: %s", req.URL.Host)
			}

			if req.URL.Query().Get("url") != "https://example.com/path" {
				t.Fatalf("unexpected PageSpeed url query: %q", req.URL.Query().Get("url"))
			}

			if req.URL.Query().Get("strategy") != "MOBILE" {
				t.Fatalf("unexpected PageSpeed strategy: %q", req.URL.Query().Get("strategy"))
			}

			if got := req.URL.Query()["category"]; len(got) != 4 {
				t.Fatalf("expected 4 categories, got %v", got)
			}

			return jsonResponse(http.StatusOK, `{
			  "id": "https://example.com/path",
			  "analysisUTCTimestamp": "2026-07-05T17:00:00Z",
			  "lighthouseResult": {
			    "requestedUrl": "https://example.com/path",
			    "finalUrl": "https://example.com/path",
			    "runWarnings": ["warning"],
			    "categories": {
			      "performance": {"id": "performance", "title": "Performance", "score": 0.72},
			      "seo": {"id": "seo", "title": "SEO", "score": 0.91}
			    },
			    "audits": {
			      "document-title": {
			        "id": "document-title",
			        "title": "Document has a title element",
			        "description": "Titles help search engines.",
			        "score": 1,
			        "scoreDisplayMode": "binary"
			      },
			      "unused-javascript": {
			        "id": "unused-javascript",
			        "title": "Reduce unused JavaScript",
			        "description": "Reduce unused JavaScript and defer loading scripts.",
			        "score": 0.4,
			        "scoreDisplayMode": "numeric",
			        "displayValue": "Potential savings of 220 KiB"
			      }
			    }
			  }
			}`), nil
		}),
		OpenAIAPIKey:      "test-key",
		OpenAIModel:       "gpt-test",
		PageSpeedStrategy: "mobile",
	})

	payload, err := generator.fetchPageSpeed(context.Background(), "https://example.com/path")
	if err != nil {
		t.Fatal(err)
	}

	snapshot := summarizePageSpeed(payload, generator.pageSpeedStrategy)

	if snapshot.FinalURL != "https://example.com/path" {
		t.Fatalf("unexpected final URL: %q", snapshot.FinalURL)
	}

	if len(snapshot.Categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(snapshot.Categories))
	}

	if len(snapshot.Opportunities) != 1 {
		t.Fatalf("expected 1 opportunity, got %d", len(snapshot.Opportunities))
	}

	if snapshot.Opportunities[0].Title != "Reduce unused JavaScript" {
		t.Fatalf("unexpected opportunity title: %q", snapshot.Opportunities[0].Title)
	}

	if len(snapshot.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(snapshot.Warnings))
	}
}

func TestGenerateRejectsInvalidURL(t *testing.T) {
	generator := NewGenerator(GeneratorConfig{
		OpenAIAPIKey: "test-key",
	})

	_, _, err := generator.Generate(context.Background(), "ftp://example.com")
	if err == nil {
		t.Fatal("expected invalid URL error")
	}
}

func TestGenerateRequiresOpenAIKey(t *testing.T) {
	generator := NewGenerator(GeneratorConfig{})

	_, _, err := generator.Generate(context.Background(), "https://example.com")
	if err == nil {
		t.Fatal("expected OpenAI key error")
	}
}

func TestNewGeneratorDefaultsOpenAIBaseURL(t *testing.T) {
	generator := NewGenerator(GeneratorConfig{})

	if generator.openAIBaseURL != defaultOpenAIBaseURL {
		t.Fatalf("expected default base URL %q, got %q", defaultOpenAIBaseURL, generator.openAIBaseURL)
	}
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
