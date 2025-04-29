package core

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// MockService implements the Interface for testing
type MockService struct {
	startCalled bool
	stopCalled  bool
	startError  error
	mu          sync.Mutex
	started     chan struct{}
	stopped     chan struct{}
}

func NewMockService() *MockService {
	return &MockService{
		started: make(chan struct{}, 1),
		stopped: make(chan struct{}, 1),
	}
}

func (ms *MockService) Start(ctx context.Context) error {
	ms.mu.Lock()
	ms.startCalled = true
	ms.mu.Unlock()

	select {
	case ms.started <- struct{}{}:
	default:
	}

	return ms.startError
}

func (ms *MockService) Stop() {
	ms.mu.Lock()
	ms.stopCalled = true
	ms.mu.Unlock()

	select {
	case ms.stopped <- struct{}{}:
	default:
	}
}

func (ms *MockService) WasStarted() bool {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.startCalled
}

func (ms *MockService) WasStopped() bool {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.stopCalled
}

func (ms *MockService) SetStartError(err error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.startError = err
}

// MockServiceWithID is a mock core that includes an ID for identifying it
type MockServiceWithID struct {
	MockService
	ID string
}

// NewMockServiceWithID creates a new mock core with an ID
func NewMockServiceWithID(id string) *MockServiceWithID {
	return &MockServiceWithID{
		MockService: MockService{
			started: make(chan struct{}, 1),
			stopped: make(chan struct{}, 1),
		},
		ID: id,
	}
}

// StopRecorder records the order of core stops
type StopRecorder struct {
	mu        sync.Mutex
	stopOrder []string
}

// RecordStop records a core stop
func (r *StopRecorder) RecordStop(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stopOrder = append(r.stopOrder, id)
}

// GetStopOrder returns the recorded stop order
func (r *StopRecorder) GetStopOrder() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stopOrder
}

// TestNewRegistry tests the creation of a new registry
func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("Expected registry to be created, got nil")
	}

	if len(registry.services) != 0 {
		t.Errorf("Expected empty services slice, got %d services", len(registry.services))
	}
}

// TestRegister tests the registration of services
func TestRegister(t *testing.T) {
	registry := NewRegistry()

	service1 := NewMockService()
	service2 := NewMockService()

	registry.Register(service1)
	if len(registry.services) != 1 {
		t.Errorf("Expected 1 core, got %d", len(registry.services))
	}

	registry.Register(service2)
	if len(registry.services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(registry.services))
	}
}

// TestStartAll tests starting all services
func TestStartAll(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	service1 := NewMockService()
	service2 := NewMockService()

	registry.Register(service1)
	registry.Register(service2)

	err := registry.StartAll(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !service1.WasStarted() {
		t.Error("Expected service1 to be started")
	}

	if !service2.WasStarted() {
		t.Error("Expected service2 to be started")
	}
}

// TestStartAllError tests handling of errors when starting services
func TestStartAllError(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	service1 := NewMockService()
	service2 := NewMockService()

	expectedErr := errors.New("start error")
	service2.SetStartError(expectedErr)

	registry.Register(service1)
	registry.Register(service2)

	err := registry.StartAll(ctx)
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}

	if !service1.WasStarted() {
		t.Error("Expected service1 to be started")
	}

	if !service2.WasStarted() {
		t.Error("Expected service2 to be started")
	}
}

// TestStopAll tests stopping all services
func TestStopAll(t *testing.T) {
	registry := NewRegistry()

	service1 := NewMockService()
	service2 := NewMockService()

	registry.Register(service1)
	registry.Register(service2)

	registry.StopAll()

	if !service1.WasStopped() {
		t.Error("Expected service1 to be stopped")
	}

	if !service2.WasStopped() {
		t.Error("Expected service2 to be stopped")
	}
}

// TestStopAllInReverseOrder tests that services are stopped in reverse order
func TestStopAllInReverseOrder(t *testing.T) {
	registry := NewRegistry()
	recorder := &StopRecorder{}

	// Create services with specific IDs
	service1 := createRecordingService("service1", recorder)
	service2 := createRecordingService("service2", recorder)
	service3 := createRecordingService("service3", recorder)

	// Register services in order
	registry.Register(service1)
	registry.Register(service2)
	registry.Register(service3)

	// Stop all services
	registry.StopAll()

	// Verify stop order (should be reverse of registration: service3, service2, service1)
	stopOrder := recorder.GetStopOrder()

	if len(stopOrder) != 3 {
		t.Fatalf("Expected 3 stops, got %d", len(stopOrder))
	}

	if stopOrder[0] != "service3" {
		t.Errorf("Expected service3 to be stopped first, got %s", stopOrder[0])
	}

	if stopOrder[1] != "service2" {
		t.Errorf("Expected service2 to be stopped second, got %s", stopOrder[1])
	}

	if stopOrder[2] != "service1" {
		t.Errorf("Expected service1 to be stopped third, got %s", stopOrder[2])
	}
}

// createRecordingService creates a core that records when it is stopped
func createRecordingService(id string, recorder *StopRecorder) Interface {
	return &recordingService{
		id:       id,
		recorder: recorder,
	}
}

// recordingService is a core that records when it is stopped
type recordingService struct {
	id       string
	recorder *StopRecorder
}

func (s *recordingService) Start(ctx context.Context) error {
	return nil
}

func (s *recordingService) Stop() {
	s.recorder.RecordStop(s.id)
}
