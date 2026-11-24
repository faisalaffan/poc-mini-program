package http

import (
	"net/http"
	"os"
)

const swaggerHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Livin' Village POC · Swagger UI</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
      window.onload = function () {
        window.ui = SwaggerUIBundle({
          url: '/openapi.yaml',
          dom_id: '#swagger-ui',
          persistAuthorization: true
        });
      };
    </script>
  </body>
</html>
`

func NewSwaggerUIHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(swaggerHTML))
	}
}

func NewOpenAPIHandler(specPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile(specPath)
		if err != nil {
			http.Error(w, "OpenAPI spec not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}
}
