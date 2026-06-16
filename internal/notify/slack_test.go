package notify

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSlackClient(t *testing.T) {
	t.Run("returns a populated client when all required env vars are set", func(t *testing.T) {
		t.Setenv("SLACK_API_TOKEN", "xoxb-token")
		t.Setenv("SLACK_CHANNEL_ID", "C123")
		t.Setenv("ENVIRONMENT", "staging")
		t.Setenv("SCALE_ACTION", "ScaleDown")

		client := NewSlackClient()
		require.NotNil(t, client)
		assert.Equal(t, "C123", client.ChannelID)
		assert.Equal(t, "staging", client.Environment)
		assert.Equal(t, "ScaleDown", client.ScaleAction)
		assert.NotNil(t, client.Client)
	})

	t.Run("returns nil when a required env var is missing", func(t *testing.T) {
		t.Setenv("SLACK_API_TOKEN", "xoxb-token")
		t.Setenv("SLACK_CHANNEL_ID", "C123")
		t.Setenv("ENVIRONMENT", "") // missing

		assert.Nil(t, NewSlackClient())
	})
}

func TestSlackNilClientIsNoOp(t *testing.T) {
	// Should not panic or attempt to send when no client is configured.
	assert.NotPanics(t, func() {
		Slack(nil, "this should be safely ignored")
	})
}
