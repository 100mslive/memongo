package memongo_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/100mslive/memongo/v2"
	"github.com/100mslive/memongo/v2/memongolog"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

func TestDefaultOptions(t *testing.T) {
	versions := []string{"6.0.0", "7.0.0", "8.0.0"}

	for _, version := range versions {
		t.Run(version, func(t *testing.T) {
			server, err := memongo.StartWithOptions(&memongo.Options{
				MongoVersion: version,
				LogLevel:     memongolog.LogLevelDebug,
			})
			require.NoError(t, err)
			defer server.Stop()

			client, err := mongo.Connect(options.Client().ApplyURI(server.URI()))
			require.NoError(t, err)

			require.NoError(t, client.Ping(context.Background(), nil))
		})
	}
}

func TestWithReplica(t *testing.T) {
	versions := []string{"6.0.0", "7.0.0", "8.0.0"}

	for _, version := range versions {
		t.Run(version, func(t *testing.T) {
			server, err := memongo.StartWithOptions(&memongo.Options{
				MongoVersion:     version,
				LogLevel:         memongolog.LogLevelDebug,
				ShouldUseReplica: true,
			})
			require.NoError(t, err)
			defer server.Stop()

			uri := fmt.Sprintf("%s%s", server.URI(), "/retryWrites=false")
			client, err := mongo.Connect(options.Client().ApplyURI(uri))
			if err != nil {
				t.Logf("err Connect: %v", err)
			}

			require.NoError(t, err)
			require.NoError(t, client.Ping(context.Background(), readpref.Primary()))
		})
	}
}

func TestWithAuth(t *testing.T) {
	versions := []string{"6.0.0", "7.0.0", "8.0.0"}

	for _, version := range versions {
		t.Run(version, func(t *testing.T) {
			server, err := memongo.StartWithOptions(&memongo.Options{
				MongoVersion: version,
				LogLevel:     memongolog.LogLevelDebug,
				Auth:         true,
			})
			require.NoError(t, err)
			defer server.Stop()

			client, err := mongo.Connect(options.Client().ApplyURI(server.URI()))
			require.NoError(t, err)

			require.NoError(t, client.Ping(context.Background(), nil))

			// Create a default user admin / 12345 to test auth.
			admin := client.Database("admin")
			res := admin.RunCommand(context.Background(), bson.D{
				{Key: "createUser", Value: "admin"},
				{Key: "pwd", Value: "12345"},
				{Key: "roles", Value: []bson.M{
					{"role": "userAdminAnyDatabase", "db": "admin"},
				}},
			})
			require.NoError(t, res.Err())

			// Verify we cannot connect without auth
			client2, err := mongo.Connect(options.Client().ApplyURI(server.URI()))
			require.NoError(t, err)

			require.NoError(t, client2.Ping(context.Background(), nil))
			_, err = client2.ListDatabaseNames(context.Background(), bson.D{})
			if strings.HasPrefix(version, "7.") || strings.HasPrefix(version, "8.") {
				require.EqualError(t, err, "(Unauthorized) Command listDatabases requires authentication")
			} else {
				require.EqualError(t, err, "(Unauthorized) command listDatabases requires authentication")
			}

			// Now connect again with auth
			opts := options.Client().ApplyURI(server.URI())
			opts.Auth = &options.Credential{
				Username:   "admin",
				Password:   "12345",
				AuthSource: "admin",
			}
			client3, err := mongo.Connect(opts)
			require.NoError(t, err)

			require.NoError(t, client3.Ping(context.Background(), nil))
			_, err = client3.ListDatabaseNames(context.Background(), bson.D{})
			require.NoError(t, err)
		})
	}
}

func TestWithReplicaAndAuth(t *testing.T) {
	versions := []string{"6.0.0", "7.0.0", "8.0.0"}

	for _, version := range versions {
		t.Run(version, func(t *testing.T) {
			server, err := memongo.StartWithOptions(&memongo.Options{
				MongoVersion:     version,
				LogLevel:         memongolog.LogLevelDebug,
				ShouldUseReplica: true,
				Auth:             true,
			})
			require.NoError(t, err)
			defer server.Stop()

			uri := fmt.Sprintf("%s%s", server.URI(), "/retryWrites=false")
			client, err := mongo.Connect(options.Client().ApplyURI(uri))
			if err != nil {
				t.Logf("err Connect: %v", err)
			}

			require.NoError(t, err)
			require.NoError(t, client.Ping(context.Background(), readpref.Primary()))

			// Create a default user admin / 12345 to test auth.
			admin := client.Database("admin")
			res := admin.RunCommand(context.Background(), bson.D{
				{Key: "createUser", Value: "admin"},
				{Key: "pwd", Value: "12345"},
				{Key: "roles", Value: []bson.M{
					{"role": "userAdminAnyDatabase", "db": "admin"},
				}},
			})
			require.NoError(t, res.Err())

			// Verify we cannot connect without auth
			client2, err := mongo.Connect(options.Client().ApplyURI(server.URI()))
			require.NoError(t, err)

			require.NoError(t, client2.Ping(context.Background(), nil))
			_, err = client2.ListDatabaseNames(context.Background(), bson.D{})
			if strings.HasPrefix(version, "7.") || strings.HasPrefix(version, "8.") {
				require.EqualError(t, err, "(Unauthorized) Command listDatabases requires authentication")
			} else {
				require.EqualError(t, err, "(Unauthorized) command listDatabases requires authentication")
			}

			// Now connect again with auth
			opts := options.Client().ApplyURI(server.URI())
			opts.Auth = &options.Credential{
				Username:   "admin",
				Password:   "12345",
				AuthSource: "admin",
			}
			client3, err := mongo.Connect(opts)
			require.NoError(t, err)

			require.NoError(t, client3.Ping(context.Background(), nil))
			_, err = client3.ListDatabaseNames(context.Background(), bson.D{})
			require.NoError(t, err)
		})
	}
}

func TestServerPing(t *testing.T) {
	server, err := memongo.StartWithOptions(&memongo.Options{
		MongoVersion: "8.0.0",
		LogLevel:     memongolog.LogLevelDebug,
	})
	require.NoError(t, err)
	defer server.Stop()

	// Test Ping method
	err = server.Ping(context.Background())
	require.NoError(t, err)
}

func TestServerHelperMethods(t *testing.T) {
	server, err := memongo.StartWithOptions(&memongo.Options{
		MongoVersion:     "8.0.0",
		LogLevel:         memongolog.LogLevelWarn,
		ShouldUseReplica: true,
		ReplicaSetName:   "customRS",
	})
	require.NoError(t, err)
	defer server.Stop()

	// Test IsReplicaSet
	require.True(t, server.IsReplicaSet())

	// Test ReplicaSetName
	require.Equal(t, "customRS", server.ReplicaSetName())

	// Test DBPath returns a non-empty path
	require.NotEmpty(t, server.DBPath())
	require.Contains(t, server.DBPath(), "memongo")
}

func TestServerNotReplicaSet(t *testing.T) {
	server, err := memongo.StartWithOptions(&memongo.Options{
		MongoVersion: "8.0.0",
		LogLevel:     memongolog.LogLevelWarn,
	})
	require.NoError(t, err)
	defer server.Stop()

	// Test IsReplicaSet returns false
	require.False(t, server.IsReplicaSet())

	// Test ReplicaSetName returns empty string
	require.Empty(t, server.ReplicaSetName())
}

func TestWiredTigerCacheSize(t *testing.T) {
	server, err := memongo.StartWithOptions(&memongo.Options{
		MongoVersion:          "8.0.0",
		LogLevel:              memongolog.LogLevelWarn,
		WiredTigerCacheSizeGB: 0.25, // 256MB cache
	})
	require.NoError(t, err)
	defer server.Stop()

	// Verify server starts successfully with cache size limit
	err = server.Ping(context.Background())
	require.NoError(t, err)
}
