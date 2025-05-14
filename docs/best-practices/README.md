# Best Practices for MCP Go SDK

This guide covers best practices for building robust and scalable MCP services using the Go SDK.

## Circuit Breaker Pattern

The SDK supports circuit breaker patterns to prevent cascading failures. Here's how to implement it:

```go
package main

import (
    "context"
    "time"

    "github.com/harriteja/mcp-go-sdk/pkg/client"
    "github.com/sony/gobreaker"
)

func NewClientWithCircuitBreaker() *client.Client {
    // Configure circuit breaker
    cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
        Name:        "mcp-client",
        MaxRequests: 3,
        Interval:    10 * time.Second,
        Timeout:     60 * time.Second,
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
            return counts.Requests >= 3 && failureRatio >= 0.6
        },
    })

    // Create client with circuit breaker
    cli := client.New(client.Options{
        ServerURL: "http://localhost:8080",
        Middleware: []client.Middleware{
            func(next client.Handler) client.Handler {
                return func(ctx context.Context, req interface{}) (interface{}, error) {
                    return cb.Execute(func() (interface{}, error) {
                        return next(ctx, req)
                    })
                }
            },
        },
    })

    return cli
}
```

## Caching Strategies

Implement caching to improve performance and reduce server load:

```go
package main

import (
    "context"
    "time"

    "github.com/harriteja/mcp-go-sdk/pkg/server"
    "github.com/harriteja/mcp-go-sdk/pkg/types"
    "github.com/patrickmn/go-cache"
)

func NewServerWithCache() *server.Server {
    // Create cache with 5 minute TTL and 10 minute cleanup interval
    c := cache.New(5*time.Minute, 10*time.Minute)

    srv := server.New(server.Options{
        Name:    "cached-server",
        Version: "1.0.0",
    })

    // Wrap tool handler with caching
    srv.OnCallTool(func(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
        // Create cache key from tool name and args
        key := fmt.Sprintf("%s:%v", name, args)

        // Check cache first
        if cached, found := c.Get(key); found {
            return cached, nil
        }

        // Call actual tool handler
        result, err := callTool(ctx, name, args)
        if err != nil {
            return nil, err
        }

        // Cache successful result
        c.Set(key, result, cache.DefaultExpiration)
        return result, nil
    })

    return srv
}
```

## Service Discovery Integration

Integrate with service discovery systems:

```go
package main

import (
    "context"

    "github.com/harriteja/mcp-go-sdk/pkg/client"
    "github.com/hashicorp/consul/api"
)

func NewClientWithServiceDiscovery() (*client.Client, error) {
    // Configure Consul client
    consulConfig := api.DefaultConfig()
    consulClient, err := api.NewClient(consulConfig)
    if err != nil {
        return nil, err
    }

    // Create service resolver
    resolver := func() (string, error) {
        services, _, err := consulClient.Health().Service("mcp-server", "", true, nil)
        if err != nil {
            return "", err
        }
        if len(services) == 0 {
            return "", fmt.Errorf("no healthy services found")
        }
        service := services[0].Service
        return fmt.Sprintf("http://%s:%d", service.Address, service.Port), nil
    }

    // Create client with dynamic service discovery
    cli := client.New(client.Options{
        ServerResolver: resolver,
    })

    return cli, nil
}
```

## Performance Optimization

### Connection Pooling

Use connection pooling for HTTP clients:

```go
func NewClientWithPool() *client.Client {
    transport := &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 100,
        IdleConnTimeout:     90 * time.Second,
    }

    return client.New(client.Options{
        HTTPClient: &http.Client{
            Transport: transport,
            Timeout:   30 * time.Second,
        },
    })
}
```

### Request Rate Limiting

Implement rate limiting to protect your services:

```go
func NewServerWithRateLimit() *server.Server {
    limiter := rate.NewLimiter(rate.Limit(100), 200) // 100 requests/sec, burst of 200

    srv := server.New(server.Options{
        Name:    "rate-limited-server",
        Version: "1.0.0",
        Middleware: []server.Middleware{
            func(next server.Handler) server.Handler {
                return func(ctx context.Context, req interface{}) (interface{}, error) {
                    if err := limiter.Wait(ctx); err != nil {
                        return nil, &types.Error{
                            Code:    429,
                            Message: "Too many requests",
                        }
                    }
                    return next(ctx, req)
                }
            },
        },
    })

    return srv
}
```

## Monitoring and Metrics

Integrate with Prometheus for metrics:

```go
func NewServerWithMetrics() *server.Server {
    // Define metrics
    requestCounter := prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "mcp_requests_total",
            Help: "Total number of MCP requests",
        },
        []string{"method", "status"},
    )

    requestDuration := prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "mcp_request_duration_seconds",
            Help:    "Request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method"},
    )

    // Register metrics
    prometheus.MustRegister(requestCounter, requestDuration)

    // Create server with metrics middleware
    srv := server.New(server.Options{
        Name:    "monitored-server",
        Version: "1.0.0",
        Middleware: []server.Middleware{
            func(next server.Handler) server.Handler {
                return func(ctx context.Context, req interface{}) (interface{}, error) {
                    start := time.Now()
                    result, err := next(ctx, req)
                    duration := time.Since(start).Seconds()

                    method := ctx.Value("method").(string)
                    status := "success"
                    if err != nil {
                        status = "error"
                    }

                    requestCounter.WithLabelValues(method, status).Inc()
                    requestDuration.WithLabelValues(method).Observe(duration)

                    return result, err
                }
            },
        },
    })

    return srv
}
```

## Security Best Practices

### TLS Configuration

Always use secure TLS configuration:

```go
func NewSecureServer() *server.Server {
    tlsConfig := &tls.Config{
        MinVersion:               tls.VersionTLS12,
        PreferServerCipherSuites: true,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
        },
    }

    srv := server.New(server.Options{
        Name:      "secure-server",
        Version:   "1.0.0",
        TLSConfig: tlsConfig,
    })

    return srv
}
```

### Authentication Middleware

Implement authentication middleware:

```go
func NewAuthenticatedServer() *server.Server {
    srv := server.New(server.Options{
        Name:    "auth-server",
        Version: "1.0.0",
        Middleware: []server.Middleware{
            func(next server.Handler) server.Handler {
                return func(ctx context.Context, req interface{}) (interface{}, error) {
                    token := ctx.Value("authorization").(string)
                    if !validateToken(token) {
                        return nil, &types.Error{
                            Code:    401,
                            Message: "Unauthorized",
                        }
                    }
                    return next(ctx, req)
                }
            },
        },
    })

    return srv
}
```

## Error Handling

Implement proper error handling and logging:

```go
func NewServerWithErrorHandling() *server.Server {
    logger, _ := zap.NewProduction()

    srv := server.New(server.Options{
        Name:    "error-handled-server",
        Version: "1.0.0",
        Middleware: []server.Middleware{
            func(next server.Handler) server.Handler {
                return func(ctx context.Context, req interface{}) (interface{}, error) {
                    result, err := next(ctx, req)
                    if err != nil {
                        logger.Error("Request failed",
                            zap.Error(err),
                            zap.Any("request", req),
                            zap.String("method", ctx.Value("method").(string)),
                        )
                        
                        // Convert internal errors to user-friendly errors
                        if _, ok := err.(*types.Error); !ok {
                            return nil, &types.Error{
                                Code:    500,
                                Message: "Internal server error",
                            }
                        }
                    }
                    return result, err
                }
            },
        },
    })

    return srv
}
``` 