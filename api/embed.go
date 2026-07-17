// api/embed.go
package api

import "embed"

//go:embed openapi.json
var OpenAPISpec []byte

//go:embed all:swagger-ui
var SwaggerUI embed.FS
