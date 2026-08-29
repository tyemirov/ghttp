package integration

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

const (
	loopAwarePixelOrigin = "https://loopaware.mprlab.com/pixel.js"
	gHTTPPixelURL        = loopAwarePixelOrigin + "?site_id=8a50915e-f5db-461a-b3da-3e6105ad9935"
)

func TestPagesArtifactUsesProductionLoopAwareIdentity(t *testing.T) {
	repositoryRoot := getRepositoryRoot(t)
	pagesServer := httptest.NewServer(http.FileServer(http.Dir(filepath.Join(repositoryRoot, "docs"))))
	t.Cleanup(pagesServer.Close)

	for _, pagePath := range []string{"/", "/404.html"} {
		response, requestErr := http.Get(pagesServer.URL + pagePath)
		if requestErr != nil {
			t.Fatalf("request published page %s: %v", pagePath, requestErr)
		}
		pageDocument, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil {
			t.Fatalf("read published page %s: %v", pagePath, readErr)
		}
		if response.StatusCode != http.StatusOK {
			t.Errorf("published page %s returned %d", pagePath, response.StatusCode)
		}
		pageText := string(pageDocument)
		if strings.Count(pageText, loopAwarePixelOrigin) != 1 {
			t.Errorf("published page %s must contain one LoopAware pixel", pagePath)
		}
		if !strings.Contains(pageText, gHTTPPixelURL) {
			t.Errorf("published page %s does not use the production gHTTP site identity", pagePath)
		}
	}
}
