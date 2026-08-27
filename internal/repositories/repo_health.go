package repositories

import "context"

// RepositoryHealthMethods exposes liveness probes for the backing stores.
type RepositoryHealthMethods interface {
	PingDatabase(ctx context.Context) error
	PingCache(ctx context.Context) error
}

type RepositoryHealth struct {
	access *RepositoryAccess
}

func NewRepositoryHealth(access *RepositoryAccess) RepositoryHealthMethods {
	return &RepositoryHealth{access: access}
}

// PingDatabase verifies both Postgres handles, or reports
// ErrDatabaseNotConfigured when the service runs without one.
func (r *RepositoryHealth) PingDatabase(ctx context.Context) error {
	if r.access.Db == nil {
		return ErrDatabaseNotConfigured
	}
	return r.access.Db.Ping(ctx)
}

// PingCache verifies the Redis connection.
func (r *RepositoryHealth) PingCache(ctx context.Context) error {
	return r.access.Cache.Ping(ctx)
}
