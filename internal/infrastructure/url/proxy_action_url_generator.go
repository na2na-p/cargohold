package url

import (
	"strings"

	"github.com/na2na-p/cargohold/internal/usecase"
)

var _ usecase.ActionURLGenerator = (*ProxyActionURLGenerator)(nil)

type ProxyActionURLGenerator struct{}

func NewProxyActionURLGenerator() *ProxyActionURLGenerator {
	return &ProxyActionURLGenerator{}
}

func (g *ProxyActionURLGenerator) GenerateUploadURL(baseURL, owner, repo, oid string) string {
	return g.generateURL(baseURL, owner, repo, oid)
}

func (g *ProxyActionURLGenerator) GenerateDownloadURL(baseURL, owner, repo, oid string) string {
	return g.generateURL(baseURL, owner, repo, oid)
}

func (g *ProxyActionURLGenerator) generateURL(baseURL, owner, repo, oid string) string {
	base := strings.TrimSuffix(baseURL, "/")
	return base + "/" + owner + "/" + repo + "/info/lfs/objects/" + oid
}
