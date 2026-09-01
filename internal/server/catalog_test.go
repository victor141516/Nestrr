package server

import (
	"net/url"
	"testing"
)

func TestCacheableCatalogDetailSeparatesEnrichedResponses(t *testing.T) {
	if !cacheableCatalogDetail(true, url.Values{"language": {"es-ES"}}) {
		t.Fatal("ordinary detail responses should use the user metadata cache")
	}
	if cacheableCatalogDetail(true, url.Values{"append_to_response": {"release_dates"}}) {
		t.Fatal("enriched movie details must not be served from or overwrite the ordinary cache")
	}
	if cacheableCatalogDetail(true, url.Values{"append_to_response": {"content_ratings"}}) {
		t.Fatal("enriched TV details must not be served from or overwrite the ordinary cache")
	}
	if cacheableCatalogDetail(false, nil) {
		t.Fatal("list responses are never user detail metadata")
	}
}
