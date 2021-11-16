package serve

import (
	"net/http"
	"testing"
	"time"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stretchr/testify/require"
)

func TestHorizonClient(t *testing.T) {
	opts := Options{HorizonURL: "my-horizon.domain.com"}
	horizonClientInterface := opts.horizonClient()

	horizonClient, ok := horizonClientInterface.(*horizonclient.Client)
	require.True(t, ok)
	require.Equal(t, "my-horizon.domain.com", horizonClient.HorizonURL)

	httpClient, ok := horizonClient.HTTP.(*http.Client)
	require.True(t, ok)
	require.Equal(t, http.Client{Timeout: 30 * time.Second}, *httpClient)
}
