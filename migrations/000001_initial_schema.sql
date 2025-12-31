-- +goose Up
-- cargohold 初期スキーマ
-- LFSオブジェクトメタデータと許可リポジトリリストのテーブルを作成

-- lfs_objects テーブル作成
CREATE TABLE lfs_objects (
	oid VARCHAR(64) PRIMARY KEY,
	size BIGINT NOT NULL,
	hash_algo VARCHAR(16) NOT NULL DEFAULT 'sha256',
	storage_key TEXT NOT NULL,
	uploaded BOOLEAN NOT NULL DEFAULT false,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_lfs_objects_created_at ON lfs_objects(created_at);
CREATE INDEX idx_lfs_objects_uploaded ON lfs_objects(uploaded);

-- repository_allowlist テーブル作成
CREATE TABLE repository_allowlist (
	id SERIAL PRIMARY KEY,
	repository VARCHAR(255) UNIQUE NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- lfs_object_access_policies テーブル作成
-- LFSオブジェクトとリポジトリの紐付けを管理するテーブル
CREATE TABLE lfs_object_access_policies (
	id SERIAL PRIMARY KEY,
	lfs_object_oid VARCHAR(64) NOT NULL,
	repository VARCHAR(255) NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(lfs_object_oid),
	FOREIGN KEY (lfs_object_oid) REFERENCES lfs_objects(oid) ON DELETE CASCADE
);

-- リポジトリ検索用インデックス
CREATE INDEX idx_access_policies_repository ON lfs_object_access_policies(repository);

-- OID検索用インデックス
CREATE INDEX idx_access_policies_oid ON lfs_object_access_policies(lfs_object_oid);

-- +goose Down
-- cargohold 初期スキーマのロールバック
DROP TABLE IF EXISTS lfs_object_access_policies;
DROP TABLE IF EXISTS repository_allowlist;
DROP TABLE IF EXISTS lfs_objects;
