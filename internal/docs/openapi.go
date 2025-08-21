package docs

import (
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// paramRe is a regular expression to match Gin path parameters.
var paramRe = regexp.MustCompile(`(:\w+|\*\w+)`)

// Mount adds routes to serve an auto-generated OpenAPI spec and Swagger UI.
// - GET /openapi.json
// - GET /docs/*any (Swagger UI loading /openapi.json)
// Local dev: Swagger UI is available at http://localhost:8080/docs/index.html
func Mount(r *gin.Engine) {
	// Serve OpenAPI JSON
	r.GET("/openapi.json", func(c *gin.Context) {
		doc := buildOpenAPIDoc(r)
		c.JSON(http.StatusOK, doc)
	})

	// Serve Swagger UI pointing to generated spec
	r.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/openapi.json")))
}

// buildOpenAPIDoc builds an OpenAPI document from a Gin router.
func buildOpenAPIDoc(r *gin.Engine) *openapi3.T {
	routes := r.Routes()

	doc := &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:       "MCPJungle API",
			Description: "MCP Jungle API",
			Version:     "0.0.1",
		},
		Paths: openapi3.NewPaths(),
	}

	// Create a stable order of routes
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path == routes[j].Path {
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Path < routes[j].Path
	})

	for _, rt := range routes {
		// skip internal swagger/docs and openapi routes to avoid recursion
		if strings.HasPrefix(rt.Path, "/docs/") || rt.Path == "/openapi.json" {
			continue
		}

		// Convert Gin path to OpenAPI path
		path := ginPathToOpenAPIPath(rt.Path)

		// Derive summary and tags from the route
		op := &openapi3.Operation{
			Summary:     deriveSummary(rt),
			Description: "Auto-generated operation",
			Tags:        deriveTags(path),
			Responses:   openapi3.NewResponses(),
		}

		// Generic 200 response
		op.AddResponse(200, openapi3.NewResponse().WithDescription("OK"))

		// For methods that usually carry a body, add a generic requestBody schema
		switch strings.ToUpper(rt.Method) {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			op.RequestBody = &openapi3.RequestBodyRef{Value: &openapi3.RequestBody{
				Required: false,
				Content: openapi3.Content{
					"application/json": &openapi3.MediaType{
						Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{openapi3.TypeObject}}},
					},
				},
			}}
		}

		// Add operation to document
		doc.AddOperation(path, strings.ToUpper(rt.Method), op)
	}

	return doc
}

// ginPathToOpenAPIPath converts a Gin path to an OpenAPI path.
func ginPathToOpenAPIPath(p string) string {
	// Convert ":param" or "*param" to "{param}"
	out := paramRe.ReplaceAllStringFunc(p, func(m string) string {
		name := strings.TrimPrefix(m, ":")
		name = strings.TrimPrefix(name, "*")
		return "{" + name + "}"
	})
	return out
}

// deriveSummary derives a summary from a Gin route.
func deriveSummary(rt gin.RouteInfo) string {
	// Use the last part of handler as a human-friendly summary
	h := rt.Handler
	if idx := strings.LastIndex(h, "."); idx >= 0 && idx+1 < len(h) {
		h = h[idx+1:]
	}
	return strings.Title(strings.ToLower(rt.Method)) + " " + h
}

// deriveTags derives tags from a path.
func deriveTags(path string) []string {
	// Tag by first path segment after "/api/v*" if present, otherwise first segment
	segs := strings.Split(strings.Trim(path, "/"), "/")
	if len(segs) == 0 {
		return nil
	}
	if len(segs) >= 3 && segs[0] == "api" && strings.HasPrefix(segs[1], "v") {
		return []string{segs[2]}
	}
	return []string{segs[0]}
}
