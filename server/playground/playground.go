package playground

import (
	"html/template"
	"net/http"

	"github.com/interline-io/transitland-lib/internal/util"
)

const doc = `
<!DOCTYPE html>
<html>

<head>
    <title>{{.title}}</title>
    <style>
        html,
        body {
            height: 100%;
            margin: 0;
            overflow: hidden;
            width: 100%;
        }
        #graphiql {
            height: 100vh;
        }
    </style>

    <link rel="stylesheet" href="https://unpkg.com/graphiql@3.0.9/graphiql.min.css" crossorigin="anonymous" />
</head>

<body>
    <div id="graphiql"></div>
    <script src="https://unpkg.com/react@18/umd/react.production.min.js" crossorigin="anonymous"></script>
    <script src="https://unpkg.com/react-dom@18/umd/react-dom.production.min.js" crossorigin="anonymous"></script>
    <script src="https://unpkg.com/graphiql@3.0.9/graphiql.min.js" crossorigin="anonymous"></script>
    <script>
        const url = location.protocol + '//' + location.host + '{{.endpoint}}';
        const wsProto = location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = wsProto + '//' + location.host + '{{.endpoint}}';

        const fetcher = GraphiQL.createGraphiQLFetcher({
            url: url,
            wsUrl: wsUrl,
        });

        ReactDOM.createRoot(document.getElementById('graphiql')).render(
            React.createElement(GraphiQL, {
                fetcher: fetcher,
                defaultQuery: 'query { feeds { onestop_id } }',
            }),
        );
    </script>
</body>
</html>
`

var page = template.Must(template.New("graphiql").Parse(doc))

// Handler responsible for setting up the playground
func Handler(title string, endpoint string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")
		err := page.Execute(w, map[string]string{
			"title":    title,
			"endpoint": endpoint,
		})
		if err != nil {
			util.WriteJsonError(w, http.StatusText(http.StatusUnauthorized), http.StatusInternalServerError)
		}
	}
}
