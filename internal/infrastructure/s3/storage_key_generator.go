package s3

type StorageKeyGeneratorImpl struct{}

func NewStorageKeyGenerator() *StorageKeyGeneratorImpl {
	return &StorageKeyGeneratorImpl{}
}

func (g *StorageKeyGeneratorImpl) GenerateStorageKey(oid, hashAlgo string) (string, error) {
	return GenerateStorageKey(oid, hashAlgo)
}
