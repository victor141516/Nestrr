// mockjackett implements the Jackett endpoints exercised by the E2E test.
package main

import (
	"fmt"
	"html"
	"net/http"
	"strings"
	"sync/atomic"
)

const releaseName = "Temporal.Horizon.2026.1080p.WEB-DL.VP9-E2E"
const infoHash = "4242424242424242424242424242424242424242"

var searches atomic.Int64

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("/api/v2.0/server/config", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"appVersion":"mock-jackett-e2e"}`))
	})
	mux.HandleFunc("/api/v2.0/indexers/all/results/torznab/api", func(w http.ResponseWriter, r *http.Request) {
		searches.Add(1)
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		magnet := "magnet:?xt=urn:btih:" + infoHash + "&dn=" + releaseName
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?><rss xmlns:torznab="http://torznab.com/schemas/2015/feed"><channel><title>E2E Jackett</title><item><title>%s</title><guid>mock:%s</guid><jackettindexer>E2E Indexer</jackettindexer><link>%s</link><enclosure url="%s" length="1048576" type="application/x-bittorrent"/><torznab:attr name="seeders" value="42"/><torznab:attr name="size" value="1048576"/><torznab:attr name="magneturl" value="%s"/><torznab:attr name="tracker" value="E2E Indexer"/><torznab:attr name="publishdate" value="2026-08-27T00:00:00Z"/></item></channel></rss>`, releaseName, html.EscapeString(query), html.EscapeString(magnet), html.EscapeString(magnet), html.EscapeString(magnet))
	})
	mux.HandleFunc("/mock/state", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"searches":%d,"releaseName":%q}`, searches.Load(), releaseName)
	})
	_ = http.ListenAndServe(":9117", mux)
}
