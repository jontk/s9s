// Package initialization provides centralized component initialization and lifecycle management
// for the observability plugin. It handles dependency injection, proper startup order,
// and graceful shutdown of all plugin components including Prometheus clients, security
// layers, caching systems, and external APIs.
package initialization

import (
	"context"
	"fmt"
	"time"

	"github.com/jontk/s9s/plugins/observability/analysis"
	"github.com/jontk/s9s/plugins/observability/endpoints"
	"github.com/jontk/s9s/plugins/observability/config"
	"github.com/jontk/s9s/plugins/observability/historical"
	"github.com/jontk/s9s/plugins/observability/metrics"
	"github.com/jontk/s9s/plugins/observability/overlays"
	"github.com/jontk/s9s/plugins/observability/prometheus"
	"github.com/jontk/s9s/plugins/observability/security"
	"github.com/jontk/s9s/plugins/observability/subscription"
)

// Components holds all initialized plugin components
type Components struct {
	Client              *prometheus.Client
	CachedClient        *prometheus.CachedClient
	MetricsCollector    *metrics.Collector
	SecretsManager      *security.SecretsManager
	OverlayMgr          *overlays.OverlayManager
	SubscriptionMgr     *subscription.SubscriptionManager
	NotificationMgr     *subscription.NotificationManager
	Persistence         *subscription.SubscriptionPersistence
	HistoricalCollector *historical.HistoricalDataCollector
	HistoricalAnalyzer  *historical.HistoricalAnalyzer
	EfficiencyAnalyzer  *analysis.ResourceEfficiencyAnalyzer
	ExternalAPI         *endpoints.ExternalAPI
}

// Manager handles plugin component initialization
type Manager struct {
	config *config.Config
	ctx    context.Context
}

// NewManager creates a new initialization manager
func NewManager(config *config.Config) *Manager {
	return &Manager{
		config: config,
		ctx:    context.Background(),
	}
}

// NewManagerWithContext creates a new initialization manager with context
func NewManagerWithContext(ctx context.Context, config *config.Config) *Manager {
	return &Manager{
		config: config,
		ctx:    ctx,
	}
}

// InitializeComponents initializes all plugin components
func (m *Manager) InitializeComponents() (*Components, error) {
	if m.config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	components := &Components{}

	initializers := []struct {
		name  string
		fn    func(*Components) error
	}{
		{"secrets manager", m.initSecretsManager},
		{"Prometheus client", m.initPrometheusClient},
		{"caching", m.initCaching},
		{"metrics", m.initMetrics},
		{"overlays", m.initOverlays},
		{"subscriptions", m.initSubscriptions},
		{"historical data", m.initHistoricalData},
		{"analysis", m.initAnalysis},
		{"external API", m.initExternalAPI},
	}

	for _, init := range initializers {
		if err := init.fn(components); err != nil {
			return nil, fmt.Errorf("failed to initialize %s: %w", init.name, err)
		}
	}

	return components, nil
}

// initSecretsManager initializes the secrets manager
func (m *Manager) initSecretsManager(components *Components) error {
	secretsManager, err := security.NewSecretsManager(m.ctx, &m.config.Security.Secrets)
	if err != nil {
		return fmt.Errorf("failed to create secrets manager: %w", err)
	}

	components.SecretsManager = secretsManager
	return nil
}

// resolveSecret resolves a secret value from either direct config or secret reference
func (m *Manager) resolveSecret(directValue, secretRef string, components *Components) (string, error) {
	if secretRef != "" && components.SecretsManager != nil {
		// Use secret reference
		return components.SecretsManager.GetSecretValue(secretRef)
	}
	// Use direct value
	return directValue, nil
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

	// Add authentication configuration with secret resolution
	switch m.config.Prometheus.Auth.Type {
	case "basic":
		clientConfig.Username = m.config.Prometheus.Auth.Username

		// Resolve password from secret or direct config
		password, err := m.resolveSecret(m.config.Prometheus.Auth.Password, m.config.Prometheus.Auth.PasswordSecretRef, components)
		if err != nil {
			return fmt.Errorf("failed to resolve password secret: %w", err)
		}
		clientConfig.Password = password

	case "bearer":
		// Resolve token from secret or direct config
		token, err := m.resolveSecret(m.config.Prometheus.Auth.Token, m.config.Prometheus.Auth.TokenSecretRef, components)
		if err != nil {
			return fmt.Errorf("failed to resolve bearer token secret: %w", err)
		}
		clientConfig.BearerToken = token
	}

	client, err := prometheus.NewClient(&clientConfig)
	if err != nil {
		return err
	}

	components.Client = client
	return nil
}

// initCaching initializes the caching layer
func (m *Manager) initCaching(components *Components) error {
	if components.Client == nil {
		return fmt.Errorf("prometheus client not initialized")
	}

	// Optionally wrap with circuit breaker
	// Check if circuit breaker should be enabled (add this to config later)
	// For now, we'll enable it by default with sensible defaults
	circuitConfig := prometheus.DefaultCircuitBreakerConfig()
	circuitConfig.OnStateChange = func(_ string, _, _ prometheus.CircuitState) {
		// Log state changes (in real implementation, use proper logging)
		// fmt.Printf("Circuit breaker %s changed from %s to %s\n", name, from, to)
	}

	circuitClient := prometheus.NewCircuitBreakerClient(components.Client, circuitConfig)
	clientForCache := circuitClient

	components.CachedClient = prometheus.NewCachedClientWithInterface(
		clientForCache,
		m.config.Cache.DefaultTTL,
		m.config.Cache.MaxSize,
	)

	return nil
}

// initMetrics initializes the metrics collection system
func (m *Manager) initMetrics(components *Components) error {
	if components.CachedClient == nil {
		return fmt.Errorf("cached client not initialized")
	}

	// Create metrics collector
	collector := metrics.NewCollector(m.ctx, components.CachedClient)

	// Wrap the cached client with instrumentation
	instrumentedClient := collector.WrapClient(components.CachedClient)

	// Update the cached client to use the instrumented version
	if cachedClient, ok := instrumentedClient.(*prometheus.CachedClient); ok {
		components.CachedClient = cachedClient
	} else {
		// If wrapping doesn't return a CachedClient, we need to handle this
		// For now, we'll create a new cached client with the instrumented client
		components.CachedClient = prometheus.NewCachedClientWithInterface(
			instrumentedClient,
			m.config.Cache.DefaultTTL,
			m.config.Cache.MaxSize,
		)
	}

	components.MetricsCollector = collector

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

// initExternalAPI initializes the external API
func (m *Manager) initExternalAPI(components *Components) error {
	if components.CachedClient == nil {
		return fmt.Errorf("cached client not initialized")
	}

	// Resolve auth token from secret if needed
	authToken, err := m.resolveSecret(m.config.Security.API.AuthToken, m.config.Security.API.AuthTokenSecretRef, components)
	if err != nil {
		return fmt.Errorf("failed to resolve API auth token: %w", err)
	}

	apiConfig := m.config.ExternalAPI
	// Override with security settings from main config
	apiConfig.AuthToken = authToken
	apiConfig.RateLimit = m.config.Security.API.RateLimit
	apiConfig.Validation = m.config.Security.API.Validation
	apiConfig.Audit = m.config.Security.API.Audit

	// Always initialize ExternalAPI, whether enabled or not
	// If disabled, it just won't be started
	externalAPI := endpoints.NewExternalAPI(
		components.CachedClient,
		components.SubscriptionMgr,
		components.HistoricalCollector,
		components.HistoricalAnalyzer,
		components.EfficiencyAnalyzer,
		&apiConfig,
	)
	components.ExternalAPI = externalAPI

	return nil
}

// Stop gracefully stops all components
func (c *Components) Stop() error {
	var errors []error

	// Note: ExternalAPI.Stop needs a context, we'll handle this in the main plugin
	c.stopErrorComponent(c.HistoricalCollector, "historical collector", &errors)
	c.stopComponent(c.Persistence)
	c.stopErrorComponent(c.SubscriptionMgr, "subscription manager", &errors)
	c.stopErrorComponent(c.OverlayMgr, "overlay manager", &errors)
	c.stopErrorComponent(c.MetricsCollector, "metrics collector", &errors)
	c.stopComponent(c.CachedClient)
	c.stopErrorComponent(c.SecretsManager, "secrets manager", &errors)

	if len(errors) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errors)
	}

	return nil
}

func (c *Components) stopComponent(component interface{ Stop() }) {
	if component != nil {
		component.Stop()
	}
}

func (c *Components) stopErrorComponent(component interface{ Stop() error }, name string, errors *[]error) {
	if component != nil {
		if err := component.Stop(); err != nil {
			*errors = append(*errors, fmt.Errorf("failed to stop %s: %w", name, err))
		}
	}
}
