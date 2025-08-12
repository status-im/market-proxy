package interfaces

type CacheStatus string

const (
	CacheStatusFull    CacheStatus = "full"
	CacheStatusPartial CacheStatus = "partial"
	CacheStatusMiss    CacheStatus = "miss"
)

func (cs CacheStatus) String() string {
	return string(cs)
}
