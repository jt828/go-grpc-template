package unit

import (
	"testing"

	"github.com/jt828/go-grpc-template/internal/bootstrap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPodNodeID(t *testing.T) {
	t.Run("different hostnames produce different node IDs", func(t *testing.T) {
		t.Setenv("HOSTNAME", "go-grpc-template-7f8b9c6d4-x2k9p")
		id1, err := bootstrap.PodNodeID()
		require.NoError(t, err)

		t.Setenv("HOSTNAME", "go-grpc-template-7f8b9c6d4-a3m7n")
		id2, err := bootstrap.PodNodeID()
		require.NoError(t, err)

		assert.NotEqual(t, id1, id2)
	})

	t.Run("same hostname produces same node ID", func(t *testing.T) {
		t.Setenv("HOSTNAME", "go-grpc-template-7f8b9c6d4-x2k9p")
		id1, err := bootstrap.PodNodeID()
		require.NoError(t, err)

		id2, err := bootstrap.PodNodeID()
		require.NoError(t, err)

		assert.Equal(t, id1, id2)
	})

	t.Run("node ID is within valid range 0-1023", func(t *testing.T) {
		hostnames := []string{
			"go-grpc-template-7f8b9c6d4-x2k9p",
			"go-grpc-template-7f8b9c6d4-a3m7n",
			"go-grpc-template-abc123-def456",
			"my-app-pod-zzzzz",
		}

		for _, hostname := range hostnames {
			t.Setenv("HOSTNAME", hostname)
			id, err := bootstrap.PodNodeID()
			require.NoError(t, err)
			assert.GreaterOrEqual(t, id, int64(0))
			assert.LessOrEqual(t, id, int64(1023))
		}
	})

	t.Run("empty hostname returns error", func(t *testing.T) {
		t.Setenv("HOSTNAME", "")
		_, err := bootstrap.PodNodeID()
		assert.Error(t, err)
	})
}
