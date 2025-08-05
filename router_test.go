package simplerouter

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRouter(t *testing.T) {
	t.Run("creates new router with no middleware", func(t *testing.T) {
		router := NewRouter()

		if router == nil {
			t.Fatal("NewRouter should not return nil")
		}

		if router.mux == nil {
			t.Error("Router should have a mux")
		}

		if len(router.chain) != 0 {
			t.Errorf("Expected empty middleware chain, got %d middlewares", len(router.chain))
		}
	})

	t.Run("creates new router with middleware", func(t *testing.T) {
		middleware1 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}

		middleware2 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}

		router := NewRouter(middleware1, middleware2)

		if len(router.chain) != 2 {
			t.Errorf("Expected 2 middlewares, got %d", len(router.chain))
		}
	})
}

func TestRouterHTTPMethods(t *testing.T) {
	tests := []struct {
		method           string
		setupRoute       func(*Router)
		path             string
		expectedResponse string
	}{
		{
			method: "GET",
			setupRoute: func(r *Router) {
				r.Get("/test", func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(200)
					w.Write([]byte("GET response"))
				})
			},
			path:             "/test",
			expectedResponse: "GET response",
		},
		{
			method: "POST",
			setupRoute: func(r *Router) {
				r.Post("/test", func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(200)
					w.Write([]byte("POST response"))
				})
			},
			path:             "/test",
			expectedResponse: "POST response",
		},
		{
			method: "PUT",
			setupRoute: func(r *Router) {
				r.Put("/test", func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(200)
					w.Write([]byte("PUT response"))
				})
			},
			path:             "/test",
			expectedResponse: "PUT response",
		},
		{
			method: "DELETE",
			setupRoute: func(r *Router) {
				r.Delete("/test", func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(200)
					w.Write([]byte("DELETE response"))
				})
			},
			path:             "/test",
			expectedResponse: "DELETE response",
		},
		{
			method: "HEAD",
			setupRoute: func(r *Router) {
				r.Head("/test", func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(200)
				})
			},
			path:             "/test",
			expectedResponse: "",
		},
		{
			method: "OPTIONS",
			setupRoute: func(r *Router) {
				r.Options("/test", func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(200)
					w.Write([]byte("OPTIONS response"))
				})
			},
			path:             "/test",
			expectedResponse: "OPTIONS response",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s method", tt.method), func(t *testing.T) {
			router := NewRouter()
			tt.setupRoute(router)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			body := w.Body.String()
			if body != tt.expectedResponse {
				t.Errorf("Expected response %q, got %q", tt.expectedResponse, body)
			}
		})
	}
}

func TestRouterAny(t *testing.T) {
	router := NewRouter()
	router.Any("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(fmt.Sprintf("%s response", r.Method)))
	})

	methods := []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"}

	for _, method := range methods {
		t.Run(fmt.Sprintf("Any handles %s", method), func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			if method != "HEAD" {
				expectedBody := fmt.Sprintf("%s response", method)
				body := w.Body.String()
				if body != expectedBody {
					t.Errorf("Expected response %q, got %q", expectedBody, body)
				}
			}
		})
	}
}

func TestRouterMiddleware(t *testing.T) {
	t.Run("middleware executes in order", func(t *testing.T) {
		var order []string

		middleware1 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "middleware1")
				next.ServeHTTP(w, r)
			})
		}

		middleware2 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "middleware2")
				next.ServeHTTP(w, r)
			})
		}

		router := NewRouter(middleware1, middleware2)
		router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "handler")
			w.WriteHeader(200)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		expected := []string{"middleware1", "middleware2", "handler"}
		if len(order) != len(expected) {
			t.Fatalf("Expected %d items in order, got %d", len(expected), len(order))
		}

		for i, item := range expected {
			if order[i] != item {
				t.Errorf("Expected order[%d] = %q, got %q", i, item, order[i])
			}
		}
	})

	t.Run("Use adds middleware", func(t *testing.T) {
		router := NewRouter()

		middleware1 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Test-1", "true")
				next.ServeHTTP(w, r)
			})
		}

		middleware2 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Test-2", "true")
				next.ServeHTTP(w, r)
			})
		}

		router.Use(middleware1, middleware2)
		router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Header().Get("X-Test-1") != "true" {
			t.Error("Expected X-Test-1 header to be set")
		}

		if w.Header().Get("X-Test-2") != "true" {
			t.Error("Expected X-Test-2 header to be set")
		}
	})
}

func TestRouterGroup(t *testing.T) {
	router := NewRouter()

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Root", "true")
			next.ServeHTTP(w, r)
		})
	}

	router.Use(middleware1)

	router.Group(func(r *Router) {
		middleware2 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Group", "true")
				next.ServeHTTP(w, r)
			})
		}

		r.Use(middleware2)
		r.Get("/group", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("group response"))
		})
	})

	router.Get("/root", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("root response"))
	})

	// Test group route
	req := httptest.NewRequest("GET", "/group", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("X-Root") != "true" {
		t.Error("Expected X-Root header to be set in group route")
	}

	if w.Header().Get("X-Group") != "true" {
		t.Error("Expected X-Group header to be set in group route")
	}

	// Test root route (should not have group middleware)
	req = httptest.NewRequest("GET", "/root", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("X-Root") != "true" {
		t.Error("Expected X-Root header to be set in root route")
	}

	if w.Header().Get("X-Group") != "" {
		t.Error("Expected X-Group header to NOT be set in root route")
	}
}

func TestRouterRoute(t *testing.T) {
	router := NewRouter()

	subRouter := router.Route("/api", func(r *Router) {
		r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("users"))
		})

		r.Post("/users", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(201)
			w.Write([]byte("user created"))
		})
	})

	if subRouter == nil {
		t.Fatal("Route should return a sub-router")
	}

	// Test GET /api/users
	req := httptest.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "users" {
		t.Errorf("Expected 'users', got %q", w.Body.String())
	}

	// Test POST /api/users
	req = httptest.NewRequest("POST", "/api/users", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	if w.Body.String() != "user created" {
		t.Errorf("Expected 'user created', got %q", w.Body.String())
	}
}

func TestRouterMount(t *testing.T) {
	router := NewRouter()

	// Create a simple handler to mount
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("mounted handler"))
	})

	router.Mount("/mounted", handler)

	req := httptest.NewRequest("GET", "/mounted/anything", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "mounted handler" {
		t.Errorf("Expected 'mounted handler', got %q", w.Body.String())
	}
}

func TestRouterNotFound(t *testing.T) {
	t.Run("default not found handler", func(t *testing.T) {
		router := NewRouter()
		router.Get("/exists", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		})

		req := httptest.NewRequest("GET", "/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 404 {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("custom not found handler", func(t *testing.T) {
		router := NewRouter()
		router.SetNotFoundHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
			w.Write([]byte("custom not found"))
		}))

		req := httptest.NewRequest("GET", "/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 404 {
			t.Errorf("Expected status 404, got %d", w.Code)
		}

		if w.Body.String() != "custom not found" {
			t.Errorf("Expected 'custom not found', got %q", w.Body.String())
		}
	})
}

func TestMuxWrapper(t *testing.T) {
	t.Run("Handle registers handler", func(t *testing.T) {
		mux := newMuxWrapper()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("test"))
		})

		mux.Handle("/test", handler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Body.String() != "test" {
			t.Errorf("Expected 'test', got %q", w.Body.String())
		}
	})

	t.Run("HandleFunc registers handler function", func(t *testing.T) {
		mux := newMuxWrapper()

		mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("test func"))
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Body.String() != "test func" {
			t.Errorf("Expected 'test func', got %q", w.Body.String())
		}
	})
}

func TestRouterWithPathParameters(t *testing.T) {
	t.Run("handles path parameters", func(t *testing.T) {
		router := NewRouter()

		router.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := r.PathValue("id")
			w.WriteHeader(200)
			w.Write([]byte(fmt.Sprintf("user %s", id)))
		})

		req := httptest.NewRequest("GET", "/users/123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Body.String() != "user 123" {
			t.Errorf("Expected 'user 123', got %q", w.Body.String())
		}
	})
}

func TestRouterWrap(t *testing.T) {
	t.Run("wrap applies middleware correctly", func(t *testing.T) {
		router := NewRouter()

		middleware1 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Middleware-1", "applied")
				next.ServeHTTP(w, r)
			})
		}

		middleware2 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Middleware-2", "applied")
				next.ServeHTTP(w, r)
			})
		}

		router.Use(middleware1)

		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("handler"))
		}

		wrapped := router.wrap(handler, []middleware{middleware2})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, req)

		if w.Header().Get("X-Middleware-1") != "applied" {
			t.Error("Expected middleware1 to be applied")
		}

		if w.Header().Get("X-Middleware-2") != "applied" {
			t.Error("Expected middleware2 to be applied")
		}

		if w.Body.String() != "handler" {
			t.Errorf("Expected 'handler', got %q", w.Body.String())
		}
	})
}

func TestBuildPath(t *testing.T) {
	tests := []struct {
		name     string
		paths    []string
		expected string
	}{
		{
			name:     "path without leading slash",
			paths:    []string{"api"},
			expected: "/api",
		},
		{
			name:     "single path with leading slash",
			paths:    []string{"/api"},
			expected: "/api",
		},
		{
			name:     "multiple paths",
			paths:    []string{"api", "v1", "users"},
			expected: "/api/v1/users",
		},
		{
			name:     "paths with leading slashes",
			paths:    []string{"/api", "/v1", "/users"},
			expected: "/api/v1/users",
		},
		{
			name:     "paths with trailing slashes",
			paths:    []string{"api/", "v1/", "users/"},
			expected: "/api/v1/users",
		},
		{
			name:     "paths with both leading and trailing slashes",
			paths:    []string{"/api/", "/v1/", "/users/"},
			expected: "/api/v1/users",
		},
		{
			name:     "empty paths",
			paths:    []string{"", "api", "", "users"},
			expected: "/api/users",
		},
		{
			name:     "no paths",
			paths:    []string{},
			expected: "",
		},
		{
			name:     "single empty path",
			paths:    []string{""},
			expected: "",
		},
		{
			name:     "mixed slashes and empty",
			paths:    []string{"/", "api", "/", "users", "/"},
			expected: "/api/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRootPath(tt.paths...)
			if result != tt.expected {
				t.Errorf("buildRootPath(%v) = %q, expected %q", tt.paths, result, tt.expected)
			}
		})
	}
}

func TestRouterBasePath(t *testing.T) {
	t.Run("default base path is empty", func(t *testing.T) {
		router := NewRouter()
		if router.BasePath() != "" {
			t.Errorf("Expected empty base path, got %q", router.BasePath())
		}
	})

	t.Run("SetBasePath sets the base path", func(t *testing.T) {
		router := NewRouter()
		router.SetBasePath("/api/v1")

		if router.BasePath() != "/api/v1" {
			t.Errorf("Expected base path '/api/v1', got %q", router.BasePath())
		}
	})

	t.Run("AppendPath appends to existing path", func(t *testing.T) {
		router := NewRouter()
		router.SetBasePath("/api")
		router.AppendPath("v1")

		if router.BasePath() != "/api/v1" {
			t.Errorf("Expected base path '/api/v1', got %q", router.BasePath())
		}
	})

	t.Run("AppendPath appends to empty path", func(t *testing.T) {
		router := NewRouter()
		router.AppendPath("api")

		if router.BasePath() != "/api" {
			t.Errorf("Expected base path '/api', got %q", router.BasePath())
		}
	})

	t.Run("multiple AppendPath calls", func(t *testing.T) {
		router := NewRouter()
		router.AppendPath("api")
		router.AppendPath("v1")
		router.AppendPath("users")

		if router.BasePath() != "/api/v1/users" {
			t.Errorf("Expected base path '/api/v1/users', got %q", router.BasePath())
		}
	})
}

func TestMuxWrapperFullPattern(t *testing.T) {
	tests := []struct {
		name     string
		rootPath string
		pattern  string
		expected string
	}{
		{
			name:     "empty root path",
			rootPath: "",
			pattern:  "/users",
			expected: "/users",
		},
		{
			name:     "root path with simple pattern",
			rootPath: "/api",
			pattern:  "/users",
			expected: "/api/users",
		},
		{
			name:     "root path with method pattern",
			rootPath: "/api",
			pattern:  "GET /users",
			expected: "GET /api/users",
		},
		{
			name:     "root path with method and path parameters",
			rootPath: "/api/v1",
			pattern:  "POST /users/{id}",
			expected: "POST /api/v1/users/{id}",
		},
		{
			name:     "root path with method and wildcard",
			rootPath: "/api",
			pattern:  "GET /users/{id...}",
			expected: "GET /api/users/{id...}",
		},
		{
			name:     "pattern with tab separator",
			rootPath: "/api",
			pattern:  "GET\t/users",
			expected: "GET /api/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := &muxWrapper{
				ServeMux: http.NewServeMux(),
				rootPath: tt.rootPath,
			}

			result := mux.fullPattern(tt.pattern)
			if result != tt.expected {
				t.Errorf("fullPattern(%q) with rootPath %q = %q, expected %q",
					tt.pattern, tt.rootPath, result, tt.expected)
			}
		})
	}
}

func TestRouterPathBuildingWithRoutes(t *testing.T) {
	t.Run("routes respect base path", func(t *testing.T) {
		router := NewRouter()
		router.SetBasePath("/api")

		router.Get("/users", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("users"))
		})

		// Test that the route is accessible at the full path
		req := httptest.NewRequest("GET", "/api/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Body.String() != "users" {
			t.Errorf("Expected 'users', got %q", w.Body.String())
		}

		// Test that the route is NOT accessible at the original path
		req = httptest.NewRequest("GET", "/users", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 404 {
			t.Errorf("Expected status 404 for /users, got %d", w.Code)
		}
	})

	t.Run("nested routes build paths correctly", func(t *testing.T) {
		router := NewRouter()

		router.Route("/api", func(r *Router) {
			r.Route("/v1", func(r *Router) {
				r.Get("/users", func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(200)
					w.Write([]byte("nested users"))
				})
			})
		})

		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Body.String() != "nested users" {
			t.Errorf("Expected 'nested users', got %q", w.Body.String())
		}
	})

	t.Run("mounted handlers respect base path", func(t *testing.T) {
		router := NewRouter()
		router.SetBasePath("/api")

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("mounted"))
		})

		router.Mount("/service", handler)

		req := httptest.NewRequest("GET", "/api/service/anything", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Body.String() != "mounted" {
			t.Errorf("Expected 'mounted', got %q", w.Body.String())
		}
	})

	t.Run("sub-router inherits parent base path", func(t *testing.T) {
		router := NewRouter()
		router.SetBasePath("/api")

		subRouter := router.Route("/v1", func(r *Router) {
			r.Get("/users", func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(200)
				w.Write([]byte("sub-router"))
			})
		})

		// Check that sub-router has the correct base path
		expectedBasePath := "/api/v1"
		if subRouter.BasePath() != expectedBasePath {
			t.Errorf("Expected sub-router base path %q, got %q", expectedBasePath, subRouter.BasePath())
		}

		// Test the route works
		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Body.String() != "sub-router" {
			t.Errorf("Expected 'sub-router', got %q", w.Body.String())
		}
	})
}

func TestNewMuxWrapper(t *testing.T) {
	tests := []struct {
		name     string
		paths    []string
		expected string
	}{
		{
			name:     "no paths",
			paths:    []string{},
			expected: "",
		},
		{
			name:     "single path",
			paths:    []string{"api"},
			expected: "/api",
		},
		{
			name:     "multiple paths",
			paths:    []string{"api", "v1", "users"},
			expected: "/api/v1/users",
		},
		{
			name:     "paths with slashes",
			paths:    []string{"/api/", "/v1/"},
			expected: "/api/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := newMuxWrapper(tt.paths...)
			if mux.rootPath != tt.expected {
				t.Errorf("newMuxWrapper(%v) rootPath = %q, expected %q", tt.paths, mux.rootPath, tt.expected)
			}
		})
	}
}

func TestRouterSetHandler(t *testing.T) {
	t.Run("SetHandler sets top-level handler", func(t *testing.T) {
		router := NewRouter()

		topLevelHandler := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Top-Level", "applied")
				next.ServeHTTP(w, r)
			})
		}

		router.SetHandler(topLevelHandler)
		router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("test response"))
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Header().Get("X-Top-Level") != "applied" {
			t.Error("Expected top-level handler to be applied")
		}

		if w.Body.String() != "test response" {
			t.Errorf("Expected 'test response', got %q", w.Body.String())
		}
	})

	t.Run("top-level handler wraps all route methods", func(t *testing.T) {
		router := NewRouter()

		topLevelHandler := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Wrapped", "true")
				next.ServeHTTP(w, r)
			})
		}

		router.SetHandler(topLevelHandler)
		router.Get("/get", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("GET"))
		})
		router.Post("/post", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("POST"))
		})
		router.Put("/put", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("PUT"))
		})

		methods := []struct {
			method string
			path   string
			body   string
		}{
			{"GET", "/get", "GET"},
			{"POST", "/post", "POST"},
			{"PUT", "/put", "PUT"},
		}

		for _, tt := range methods {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Header().Get("X-Wrapped") != "true" {
				t.Errorf("Expected top-level handler to wrap %s %s", tt.method, tt.path)
			}

			if w.Body.String() != tt.body {
				t.Errorf("Expected %q for %s %s, got %q", tt.body, tt.method, tt.path, w.Body.String())
			}
		}
	})

	t.Run("top-level handler does not affect not found handler", func(t *testing.T) {
		router := NewRouter()

		topLevelHandler := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Top-Level", "applied")
				next.ServeHTTP(w, r)
			})
		}

		notFoundHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Not-Found", "custom")
			w.WriteHeader(404)
			w.Write([]byte("custom not found"))
		})

		router.SetHandler(topLevelHandler)
		router.SetNotFoundHandler(notFoundHandler)
		router.Get("/exists", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("exists"))
		})

		// Test existing route - should have top-level handler
		req := httptest.NewRequest("GET", "/exists", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Header().Get("X-Top-Level") != "applied" {
			t.Error("Expected top-level handler to be applied to existing route")
		}

		if w.Body.String() != "exists" {
			t.Errorf("Expected 'exists', got %q", w.Body.String())
		}

		// Test non-existing route - should not have top-level handler
		req = httptest.NewRequest("GET", "/nonexistent", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Header().Get("X-Top-Level") != "" {
			t.Error("Expected top-level handler NOT to be applied to not found route")
		}

		if w.Header().Get("X-Not-Found") != "custom" {
			t.Error("Expected custom not found handler to be applied")
		}

		if w.Body.String() != "custom not found" {
			t.Errorf("Expected 'custom not found', got %q", w.Body.String())
		}
	})

	t.Run("top-level handler runs before all middleware layers", func(t *testing.T) {
		router := NewRouter()
		var order []string

		routerMiddleware1 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "router-middleware-1")
				next.ServeHTTP(w, r)
			})
		}

		routerMiddleware2 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "router-middleware-2")
				next.ServeHTTP(w, r)
			})
		}

		topLevelHandler := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "top-level")
				next.ServeHTTP(w, r)
			})
		}

		routeMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "route-middleware")
				next.ServeHTTP(w, r)
			})
		}

		router.Use(routerMiddleware1, routerMiddleware2)
		router.SetHandler(topLevelHandler)
		router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "handler")
			w.WriteHeader(200)
		}, routeMiddleware)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		expected := []string{"top-level", "router-middleware-1", "router-middleware-2", "route-middleware", "handler"}
		if len(order) != len(expected) {
			t.Fatalf("Expected %d items in order, got %d: %v", len(expected), len(order), order)
		}

		for i, item := range expected {
			if order[i] != item {
				t.Errorf("Expected order[%d] = %q, got %q", i, item, order[i])
			}
		}
	})

	t.Run("top-level handler can modify request", func(t *testing.T) {
		router := NewRouter()

		topLevelHandler := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r.Header.Set("X-Modified", "by-top-level")
				next.ServeHTTP(w, r)
			})
		}

		router.SetHandler(topLevelHandler)
		router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			modifiedValue := r.Header.Get("X-Modified")
			w.WriteHeader(200)
			w.Write([]byte(fmt.Sprintf("modified: %s", modifiedValue)))
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		expectedBody := "modified: by-top-level"
		if w.Body.String() != expectedBody {
			t.Errorf("Expected %q, got %q", expectedBody, w.Body.String())
		}
	})

	t.Run("top-level handler can short-circuit request", func(t *testing.T) {
		router := NewRouter()

		topLevelHandler := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Block") == "true" {
					w.WriteHeader(403)
					w.Write([]byte("blocked"))
					return
				}
				next.ServeHTTP(w, r)
			})
		}

		router.SetHandler(topLevelHandler)
		router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("allowed"))
		})

		// Test normal request
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Body.String() != "allowed" {
			t.Errorf("Expected 'allowed', got %q", w.Body.String())
		}

		// Test blocked request
		req = httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Block", "true")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 403 {
			t.Errorf("Expected status 403, got %d", w.Code)
		}

		if w.Body.String() != "blocked" {
			t.Errorf("Expected 'blocked', got %q", w.Body.String())
		}
	})

	t.Run("top-level handler can be nil", func(t *testing.T) {
		router := NewRouter()

		// Don't set a top-level handler, should work normally
		router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("no top-level"))
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Body.String() != "no top-level" {
			t.Errorf("Expected 'no top-level', got %q", w.Body.String())
		}
	})
}

func TestRedirectFromRouteToRouteSlash(t *testing.T) {
	t.Run("intercepts the automatic 301 from net/http", func(t *testing.T) {
		methods := map[string]func(*Router, string, http.HandlerFunc, ...middleware){
			"GET":     (*Router).Get,
			"POST":    (*Router).Post,
			"PUT":     (*Router).Put,
			"DELETE":  (*Router).Delete,
			"HEAD":    (*Router).Head,
			"OPTIONS": (*Router).Options,
		}

		for methodName, methodFunc := range methods {
			t.Run(methodName, func(t *testing.T) {
				router := NewRouter()

				methodFunc(router, "/route/{$}", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
					w.Write([]byte("success"))
				})

				req := httptest.NewRequest(methodName, "/route", nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				// Check that we get a 307 status code
				if w.Code != http.StatusTemporaryRedirect {
					t.Errorf("Expected status %d, got %d", http.StatusTemporaryRedirect, w.Code)
				}

				// Check that Location header is set correctly
				location := w.Header().Get("Location")
				expectedLocation := "/route/"
				if location != expectedLocation {
					t.Errorf("Expected Location header %q, got %q", expectedLocation, location)
				}

				t.Run("with nested routes", func(t *testing.T) {
					router.Route("/nested/", func(r *Router) {
						methodFunc(r, "/route/", func(w http.ResponseWriter, r *http.Request) {
							w.WriteHeader(200)
							w.Write([]byte("nested success"))
						})

						methodFunc(r, "/{$}", func(w http.ResponseWriter, r *http.Request) {
							w.WriteHeader(200)
							w.Write([]byte("nested root success"))
						})
					})

					for _, path := range []string{"/nested/route", "/nested"} {
						req = httptest.NewRequest(methodName, path, nil)
						w = httptest.NewRecorder()
						router.ServeHTTP(w, req)

						// Check that we get a 307 status code
						if w.Code != http.StatusTemporaryRedirect {
							t.Errorf("Expected status %d, got %d", http.StatusTemporaryRedirect, w.Code)
						}

						// Check that Location header is set correctly
						location := w.Header().Get("Location")
						expectedLocation := path + "/"
						if location != expectedLocation {
							t.Errorf("Expected Location header %q, got %q", expectedLocation, location)
						}
					}
				})
			})
		}
  })
}
