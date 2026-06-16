package notify

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNewRelicClient(t *testing.T) {
	t.Run("returns nil when API key is unset", func(t *testing.T) {
		t.Setenv("NEW_RELIC_API_KEY", "")
		t.Setenv("NEW_RELIC_ALERT_POLICIES", "1,2")

		client, err := NewNewRelicClient()
		assert.NoError(t, err)
		assert.Nil(t, client)
	})

	t.Run("returns nil when no alert policies are set", func(t *testing.T) {
		t.Setenv("NEW_RELIC_API_KEY", "dummy-key")
		t.Setenv("NEW_RELIC_ALERT_POLICIES", "")

		client, err := NewNewRelicClient()
		assert.NoError(t, err)
		assert.Nil(t, client)
	})

	t.Run("errors on an unparseable policy ID", func(t *testing.T) {
		t.Setenv("NEW_RELIC_API_KEY", "dummy-key")
		t.Setenv("NEW_RELIC_ALERT_POLICIES", "1,notanumber")

		client, err := NewNewRelicClient()
		assert.Error(t, err)
		assert.Nil(t, client)
	})

	t.Run("parses and trims whitespace around policy IDs", func(t *testing.T) {
		t.Setenv("NEW_RELIC_API_KEY", "dummy-key")
		t.Setenv("NEW_RELIC_ALERT_POLICIES", "1, 2 , 3")

		client, err := NewNewRelicClient()
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.Equal(t, []int{1, 2, 3}, client.PolicyIDs)
	})
}

func TestUpdateNewRelicAlertPolicy(t *testing.T) {
	t.Run("no-op when client is nil", func(t *testing.T) {
		assert.NoError(t, UpdateNewRelicAlertPolicy(nil, ScaleDown))
	})

	t.Run("errors on an invalid action", func(t *testing.T) {
		// Empty PolicyIDs means the API client is never invoked; only the action is validated.
		assert.Error(t, UpdateNewRelicAlertPolicy(&NewRelicClient{}, "Sideways"))
	})
}
