package interfaces

// IHealthReporter is implemented by services that report whether they are able
// to serve data. The /health handler aggregates these.
type IHealthReporter interface {
	// Healthy reports whether the service has usable data
	Healthy() bool
}
