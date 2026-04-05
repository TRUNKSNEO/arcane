package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getarcaneapp/arcane/backend/internal/models"
	"github.com/stretchr/testify/require"
)

func TestDecodeSwarmSpecInternal_AllowsEmptyObject(t *testing.T) {
	spec, err := decodeSwarmSpecInternal(json.RawMessage(`{}`))
	require.NoError(t, err)
	require.NotNil(t, spec.Labels)
	require.Empty(t, spec.Labels)
}

func TestDecodeSwarmSpecInternal_RejectsNull(t *testing.T) {
	_, err := decodeSwarmSpecInternal(json.RawMessage(`null`))
	require.EqualError(t, err, "swarm spec is required")
}

func TestDefaultSwarmListenAddrInternal(t *testing.T) {
	require.Equal(t, defaultSwarmListenAddr, defaultSwarmListenAddrInternal(""))
	require.Equal(t, defaultSwarmListenAddr, defaultSwarmListenAddrInternal("   "))
	require.Equal(t, "eth0:2377", defaultSwarmListenAddrInternal(" eth0:2377 "))
}

func TestSwarmService_FetchSwarmNodeIdentityViaEdgeInternal_UsesEnvironmentAccessToken(t *testing.T) {
	ctx := context.Background()
	db := setupEnvironmentServiceTestDB(t)
	settingsSvc, err := NewSettingsService(ctx, db)
	require.NoError(t, err)
	envSvc := NewEnvironmentService(db, nil, nil, nil, settingsSvc, nil)

	accessToken := "token-123"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/swarm/node-identity", r.URL.Path)
		require.Equal(t, accessToken, r.Header.Get("X-API-Key"))
		require.Equal(t, accessToken, r.Header.Get("X-Arcane-Agent-Token"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"swarmNodeId":"node-1","hostname":"worker-1","role":"worker","engineVersion":"29.3.1","swarmActive":true}}`))
	}))
	defer server.Close()

	createTestEnvironmentWithState(
		t,
		db,
		"env-1",
		server.URL,
		string(models.EnvironmentStatusOnline),
		false,
		&accessToken,
	)

	svc := NewSwarmService(nil, nil, nil, nil, envSvc)

	identity, err := svc.fetchSwarmNodeIdentityViaEdgeInternal(ctx, "env-1")
	require.NoError(t, err)
	require.NotNil(t, identity)
	require.Equal(t, "node-1", identity.SwarmNodeID)
	require.Equal(t, "worker-1", identity.Hostname)
	require.Equal(t, "worker", identity.Role)
	require.Equal(t, "29.3.1", identity.EngineVersion)
	require.True(t, identity.SwarmActive)
}
