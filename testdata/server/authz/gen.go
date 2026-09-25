//go:generate sh -c "go run github.com/openfga/cli/cmd/fga@v0.8.1 model transform --file tls.model --output-format json > tls.json"
package authz
