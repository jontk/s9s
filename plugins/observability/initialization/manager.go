package initialization

import (
	"fmt"
	"time"

	"github.com/jontk/s9s/plugins/observability/analysis"
	"github.com/jontk/s9s/plugins/observability/api"
	"github.com/jontk/s9s/plugins/observability/config"
	"github.com/jontk/s9s/plugins/observability/historical"
	"github.com/jontk/s9s/plugins/observability/overlays"
	"github.com/jontk/s9s/plugins/observability/prometheus"
	"github.com/jontk/s9s/plugins/observability/subscription"
)

// Components holds all initialized plugin components
type Components struct {
	Client              *prometheus.Client
	CachedClient        *prometheus.CachedClient
	OverlayMgr          *overlays.OverlayManager
	SubscriptionMgr     *subscription.SubscriptionManager
	NotificationMgr     *subscription.NotificationManager
	Persistence         *subscription.SubscriptionPersistence
	HistoricalCollector *historical.HistoricalDataCollector
	HistoricalAnalyzer  *historical.HistoricalAnalyzer
	EfficiencyAnalyzer  *analysis.ResourceEfficiencyAnalyzer
	ExternalAPI         *api.ExternalAPI
}

// Manager handles plugin component initialization
type Manager struct {
	config *config.Config
}

// NewManager creates a new initialization manager
func NewManager(config *config.Config) *Manager {
	return &Manager{
		config: config,
	}
}

// InitializeComponents initializes all plugin components
func (m *Manager) InitializeComponents() (*Components, error) {
	if m.config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	
	components := &Components{}
	
	// Initialize Prometheus client
	if err := m.initPrometheusClient(components); err != nil {
		return nil, fmt.Errorf("failed to initialize Prometheus client: %w", err)
	}
	
	// Initialize caching layer
	if err := m.initCaching(components); err != nil {
		return nil, fmt.Errorf("failed to initialize caching: %w", err)
	}
	
	// Initialize overlay management
	if err := m.initOverlays(components); err != nil {
		return nil, fmt.Errorf("failed to initialize overlays: %w", err)
	}
	
	// Initialize subscription system
	if err := m.initSubscriptions(components); err != nil {
		return nil, fmt.Errorf("failed to initialize subscriptions: %w", err)
	}
	
	// Initialize historical data collection
	if err := m.initHistoricalData(components); err != nil {
		return nil, fmt.Errorf("failed to initialize historical data: %w", err)
	}
	
	// Initialize analysis components
	if err := m.initAnalysis(components); err != nil {
		return nil, fmt.Errorf("failed to initialize analysis: %w", err)
	}
	
	// Initialize external API if enabled
	if err := m.initExternalAPI(components); err != nil {
		return nil, fmt.Errorf("failed to initialize external API: %w", err)
	}
	
	return components, nil
}

// initPrometheusClient initializes the Prometheus client
func (m *Manager) initPrometheusClient(components *Components) error {
	clientConfig := prometheus.ClientConfig{
		Endpoint:      m.config.Prometheus.Endpoint,
		Timeout:       m.config.Prometheus.Timeout,
		TLSSkipVerify: m.config.Prometheus.TLS.InsecureSkipVerify,
		TLSCertFile:   m.config.Prometheus.TLS.CertFile,
		TLSKeyFile:    m.config.Prometheus.TLS.KeyFile,
		TLSCAFile:     m.config.Prometheus.TLS.CAFile,
	}

	// Add authentication configuration
	switch m.config.Prometheus.Auth.Type {
	case "basic":
		clientConfig.Username = m.config.Prometheus.Auth.Username
		clientConfig.Password = m.config.Prometheus.Auth.Password
	case "bearer":
		clientConfig.BearerToken = m.config.Prometheus.Auth.Token
	}

	client, err := prometheus.NewClient(clientConfig)
	if err != nil {
		return err
	}
	
	components.Client = client
	return nil
}

// initCaching initializes the caching layer
func (m *Manager) initCaching(components *Components) error {
	if components.Client == nil {
		return fmt.Errorf("Prometheus client not initialized")
	}
	
	// Optionally wrap with circuit breaker
	var clientForCache prometheus.PrometheusClientInterface = components.Client
	
	// Check if circuit breaker should be enabled (add this to config later)
	// For now, we'll enable it by default with sensible defaults
	circuitConfig := prometheus.DefaultCircuitBreakerConfig()
	circuitConfig.OnStateChange = func(name string, from, to prometheus.CircuitState) {
		// Log state changes (in real implementation, use proper logging)
		// fmt.Printf("Circuit breaker %s changed from %s to %s\n", name, from, to)
	}
	
	circuitClient := prometheus.NewCircuitBreakerClient(components.Client, circuitConfig)
	clientForCache = circuitClient
	
	components.CachedClient = prometheus.NewCachedClientWithInterface(
		clientForCache,
		m.config.Cache.DefaultTTL,
		m.config.Cache.MaxSize,
	)
	
	return nil
}

// initOverlays initializes the overlay management system
func (m *Manager) initOverlays(components *Components) error {
	if components.CachedClient == nil {
		return fmt.Errorf("cached client not initialized")
	}
	
	components.OverlayMgr = overlays.NewOverlayManager(
		components.CachedClient,
		m.config.Display.RefreshInterval,
	)
	
	return nil
}

// initSubscriptions initializes the subscription system
func (m *Manager) initSubscriptions(components *Components) error {
	if components.CachedClient == nil {
		return fmt.Errorf("cached client not initialized")
	}
	
	// Create subscription manager
	components.SubscriptionMgr = subscription.NewSubscriptionManager(components.CachedClient)
	
	// Create notification manager
	components.NotificationMgr = subscription.NewNotificationManager(1000)
	
	// Create persistence manager
	persistenceConfig := subscription.PersistenceConfig{
		DataDir:      "./data/observability",
		AutoSave:     true,
		SaveInterval: 5 * time.Minute,
	}

	persistence, err := subscription.NewSubscriptionPersistence(persistenceConfig, components.SubscriptionMgr)
	if err != nil {
		return err
	}
	components.Persistence = persistence

	// Load persisted subscriptions (ignore errors for initialization)
	_ = components.Persistence.LoadSubscriptions()
	
	return nil
}

// initHistoricalData initializes historical data collection
func (m *Manager) initHistoricalData(components *Components) error {
	if components.CachedClient == nil {
		return fmt.Errorf("cached client not initialized")
	}
	
	// Create historical data collector
	historicalConfig := historical.CollectorConfig{
		DataDir:         "./data/historical",
		Retention:       30 * 24 * time.Hour, // 30 days
		CollectInterval: 5 * time.Minute,
		MaxDataPoints:   10000,
		Queries:         historical.DefaultCollectorConfig().Queries,
	}

	collector, err := historical.NewHistoricalDataCollector(components.CachedClient, historicalConfig)
	if err != nil {
		return err
	}
	components.HistoricalCollector = collector

	// Create historical data analyzer
	analyzer := historical.NewHistoricalAnalyzer(collector)
	components.HistoricalAnalyzer = analyzer
	
	return nil
}

// initAnalysis initializes analysis components
func (m *Manager) initAnalysis(components *Components) error {
	if components.CachedClient == nil {
		return fmt.Errorf("cached client not initialized")
	}
	
	// Create resource efficiency analyzer
	analyzer := analysis.NewResourceEfficiencyAnalyzer(
		components.CachedClient,
		components.HistoricalCollector,
		components.HistoricalAnalyzer,
	)
	components.EfficiencyAnalyzer = analyzer
	
	return nil
}

// initExternalAPI initializes the external API if enabled
func (m *Manager) initExternalAPI(components *Components) error {
	// Check if API is enabled (we'll need to add this config option)
	// For now, we'll initialize it but not start it
	
	if components.CachedClient == nil {
		return fmt.Errorf("cached client not initialized")
	}
	
	apiConfig := api.Config{
		Port:      8080, // Default port
		AuthToken: "",   // No auth by default
	}

	externalAPI := api.NewExternalAPI(
		components.CachedClient,
		components.SubscriptionMgr,
		components.HistoricalCollector,
		components.HistoricalAnalyzer,
		components.EfficiencyAnalyzer,
		apiConfig,
	)
	components.ExternalAPI = externalAPI
	
	return nil
}

// Stop gracefully stops all components
func (c *Components) Stop() error {
	var errors []error
	
	// Stop external API
	if c.ExternalAPI != nil {
		// Note: ExternalAPI.Stop needs a context, we'll handle this in the main plugin
	}
	
	// Stop historical collector
	if c.HistoricalCollector != nil {
		if err := c.HistoricalCollector.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop historical collector: %w", err))
		}
	}
	
	// Stop subscription persistence
	if c.Persistence != nil {
		c.Persistence.Stop()
	}
	
	// Stop subscription manager
	if c.SubscriptionMgr != nil {
		if err := c.SubscriptionMgr.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop subscription manager: %w", err))
		}
	}
	
	// Stop overlay manager
	if c.OverlayMgr != nil {
		if err := c.OverlayMgr.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop overlay manager: %w", err))
		}
	}
	
	// Stop cached client
	if c.CachedClient != nil {
		c.CachedClient.Stop()
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errors)
	}
	
	return nil
}