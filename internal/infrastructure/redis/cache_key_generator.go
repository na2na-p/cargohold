package redis

import "time"

type CacheKeyGeneratorImpl struct{}

func NewCacheKeyGenerator() *CacheKeyGeneratorImpl {
	return &CacheKeyGeneratorImpl{}
}

func (g *CacheKeyGeneratorImpl) MetadataKey(oid string) string {
	return MetadataKeyPrefix + oid
}

func (g *CacheKeyGeneratorImpl) SessionKey(sessionID string) string {
	return SessionKeyPrefix + sessionID
}

func (g *CacheKeyGeneratorImpl) BatchUploadKey(oid string) string {
	return BatchUploadKeyPrefix + oid
}

type CacheConfigImpl struct{}

func NewCacheConfig() *CacheConfigImpl {
	return &CacheConfigImpl{}
}

func (c *CacheConfigImpl) MetadataTTL() time.Duration {
	return MetadataTTL
}
