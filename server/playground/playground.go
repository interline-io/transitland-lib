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

    <link rel="stylesheet"
        href="https://cdn.jsdelivr.net/npm/graphiql-with-extensions@0.14.3/graphiqlWithExtensions.css"
        integrity="{{.cssSRI}}"
		crossorigin="anonymous" />
    <script src="https://cdn.jsdelivr.net/npm/whatwg-fetch@2.0.3/fetch.min.js"
		integrity="{{.fetchSRI}}"
        crossorigin="anonymous"></script>
    <script src="https://cdn.jsdelivr.net/npm/react@16.8.6/umd/react.production.min.js"
        integrity="{{.reactSRI}}"
        crossorigin="anonymous"></script>
    <script src="https://cdn.jsdelivr.net/npm/react-dom@16.8.6/umd/react-dom.production.min.js"
        integrity="{{.reactDOMSRI}}"
        crossorigin="anonymous"></script>
    <script src="https://cdn.jsdelivr.net/npm/graphiql-with-extensions@0.14.3/graphiqlWithExtensions.min.js"
        integrity="{{.jsSRI}}"
        crossorigin="anonymous"></script>
</head>

<body>
    <div id="graphiql"></div>
    <script>
        var query = "query{feeds{onestop_id}}";
        class GraphiQLOpenExplorer extends GraphiQLWithExtensions.GraphiQLWithExtensions {
            state = {
                query: this.props.defaultQuery,
                explorerIsOpen: true,
            };
        }
		const url = location.protocol + '//' + location.host + '{{.endpoint}}';
		const wsProto = location.protocol == 'https:' ? 'wss:' : 'ws:';
		const subscriptionUrl = wsProto + '//' + location.host + '{{.endpoint}}';		
        function graphQLFetcher(graphQLParams) {
            var headers = {
                Accept: 'application/json',
                'Content-Type': 'application/json',
            };
            return fetch(url, {
                method: 'post',
                headers: headers,
                body: JSON.stringify(graphQLParams),
            })
                .then(function (response) {
                    return response.text();
                })
                .then(function (responseBody) {
                    try {
                        return JSON.parse(responseBody);
                    } catch (error) {
                        return responseBody;
                    }
                });
        }
        ReactDOM.render(React.createElement(GraphiQLOpenExplorer, {
            defaultQuery: query,
            fetcher: graphQLFetcher,
        }), document.getElementById('graphiql'));
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
			"title":       title,
			"endpoint":    endpoint,
			"cssSRI":      "sha384-GBqwox+q8UtVEyBLBKloN5QDlBDsQnuoSUfMeJH1ZtDiCrrk103D7Bg/WjIvl4ya",
			"reactSRI":    "sha384-qn+ML/QkkJxqn4LLs1zjaKxlTg2Bl/6yU/xBTJAgxkmNGc6kMZyeskAG0a7eJBR1",
			"reactDOMSRI": "sha384-85IMG5rvmoDsmMeWK/qUU4kwnYXVpC+o9hoHMLi4bpNR+gMEiPLrvkZCgsr7WWgV",
			"fetchSRI":    "sha384-dcF7KoWRaRpjcNbVPUFgatYgAijf8DqW6NWuqLdfB5Sb4Cdbb8iHX7bHsl9YhpKa",
			"jsSRI":       "sha384-TqI6gT2PjmSrnEOTvGHLad1U4Vm5VoyzMmcKK0C/PLCWTnwPyXhCJY6NYhC/tp19",
		})
		if err != nil {
			util.WriteJsonError(w, http.StatusText(http.StatusUnauthorized), http.StatusInternalServerError)
		}
	}
}
