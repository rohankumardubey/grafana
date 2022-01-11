package prometheus

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental"
	"github.com/prometheus/client_golang/api"
	apiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/stretchr/testify/require"
)

type MockedRoundTripper struct {
	responseBytes []byte
}

func (mockedRT *MockedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewReader(mockedRT.responseBytes)),
	}, nil
}

func makeMockedClient(responseBytes []byte) (apiv1.API, error) {
	roundTripper := MockedRoundTripper{responseBytes: responseBytes}

	cfg := api.Config{
		Address:      "http://localhost:9999",
		RoundTripper: &roundTripper,
	}

	client, err := api.NewClient(cfg)

	if err != nil {
		return nil, err
	}

	client2 := apiv1.NewAPI(client)

	return client2, nil
}

func getRefs(response *backend.QueryDataResponse) []string {
	var refs []string

	for key, _ := range response.Responses {
		refs = append(refs, key)
	}

	return refs
}

// we run the mocked query, and extract the DataResponse.
// we assume and verify that there is exactly one DataResponse returned.
func getResponse(responseBytes []byte, query PrometheusQuery) (backend.DataResponse, error) {
	client, err := makeMockedClient([]byte(responseBytes))
	if err != nil {
		return backend.DataResponse{}, nil
	}

	ctx := context.Background()

	result, err := runQueries(ctx, client, []*PrometheusQuery{&query})
	if err != nil {
		return backend.DataResponse{}, nil
	}

	if len(result.Responses) != 1 {
		return backend.DataResponse{}, fmt.Errorf("result must contain only one DataResponse, but contained: %v", getRefs(result))
	}

	dr, found := result.Responses["A"]

	if !found {
		return backend.DataResponse{}, fmt.Errorf("DataResponse not found in result, it contained: %v", getRefs(result))
	}

	return dr, nil
}

func TestRunQuery(t *testing.T) {

	t.Run("parse a simple matrix response", func(t *testing.T) {
		// NOTE: the time-related fields in the query structure
		// have to match with the data in the JSON_response,
		// because the code auto-generates missing values.
		start := time.Unix(1641889530, 0)

		query := PrometheusQuery{
			RefId:      "A",
			RangeQuery: true,
			Start:      start,
			End:        start.Add(time.Second * 2),
			Step:       time.Second,
		}

		response := `
		{
			"status": "success",
			"data": {
				"resultType": "matrix",
				"result": [
					{
						"metric": { "__name__": "go_goroutines", "job": "prometheus" },
						"values": [
							[1641889530, "21"],
							[1641889531, "32"],
							[1641889532, "43"]
						]
					}
				]
			}
		}

		`

		dr, err := getResponse([]byte(response), query)
		require.NoError(t, err)
		require.NoError(t, dr.Error)

		err = experimental.CheckGoldenDataResponse(filepath.Join("json_to_dataframes_testdata", "range_simple.golden.txt"), &dr, true)
		require.NoError(t, err)
	})

	t.Run("parse a simple matrix response with value missing steps", func(t *testing.T) {
		// NOTE: the time-related fields in the query structure
		// have to match with the data in the JSON_response,
		// because the code auto-generates missing values.
		start := time.Unix(1641889530, 0)

		query := PrometheusQuery{
			RefId:      "A",
			RangeQuery: true,
			Start:      start,
			End:        start.Add(time.Second * 8),
			Step:       time.Second,
		}

		response := `
		{
			"status": "success",
			"data": {
				"resultType": "matrix",
				"result": [
					{
						"metric": { "__name__": "go_goroutines", "job": "prometheus" },
						"values": [
							[1641889533, "21"],
							[1641889534, "32"],
							[1641889537, "43"]
						]
					}
				]
			}
		}

		`

		dr, err := getResponse([]byte(response), query)
		require.NoError(t, err)
		require.NoError(t, dr.Error)

		err = experimental.CheckGoldenDataResponse(filepath.Join("json_to_dataframes_testdata", "range_missing.golden.txt"), &dr, true)
		require.NoError(t, err)
	})

	t.Run("parse a response with Infinity", func(t *testing.T) {
		// NOTE: the time-related fields in the query structure
		// have to match with the data in the JSON_response,
		// because the code auto-generates missing values.
		start := time.Unix(1641889530, 0)

		query := PrometheusQuery{
			RefId:      "A",
			RangeQuery: true,
			Start:      start,
			End:        start.Add(time.Second * 2),
			Step:       time.Second,
			Expr:       "1 / 0",
		}

		response := `
		{
			"status": "success",
			"data": {
				"resultType": "matrix",
				"result": [
					{
						"metric": {},
						"values": [
							[1641889530, "+Inf"],
							[1641889531, "+Inf"],
							[1641889532, "+Inf"]
						]
					}
				]
			}
		}
		
		`

		dr, err := getResponse([]byte(response), query)
		require.NoError(t, err)
		require.NoError(t, dr.Error)

		err = experimental.CheckGoldenDataResponse(filepath.Join("json_to_dataframes_testdata", "range_infinity.golden.txt"), &dr, true)
		require.NoError(t, err)
	})

	t.Run("parse a response with NaN", func(t *testing.T) {
		// NOTE: the time-related fields in the query structure
		// have to match with the data in the JSON_response,
		// because the code auto-generates missing values.
		start := time.Unix(1641889530, 0)

		query := PrometheusQuery{
			RefId:      "A",
			RangeQuery: true,
			Start:      start,
			End:        start.Add(time.Second * 2),
			Step:       time.Second,
			Expr:       `histogram_quantile(0.99,rate(prometheus_http_response_size_bytes_bucket{handler="/api/v1/query_range"}[1m]))`,
		}

		response := `
		{
			"status": "success",
			"data": {
				"resultType": "matrix",
				"result": [
					{
						"metric": { "handler": "/api/v1/query_range", "job": "prometheus" },
						"values": [
							[1641889530, "NaN"],
							[1641889531, "NaN"],
							[1641889532, "NaN"]
						]
					}
				]
			}
		}
		
		`

		dr, err := getResponse([]byte(response), query)
		require.NoError(t, err)
		require.NoError(t, dr.Error)

		err = experimental.CheckGoldenDataResponse(filepath.Join("json_to_dataframes_testdata", "range_nan.golden.txt"), &dr, true)
		require.NoError(t, err)
	})

}
