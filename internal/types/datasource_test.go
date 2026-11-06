package types

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDataSourceConfig_ToJSON_EncryptsStringCredentials(t *testing.T) {
	withAESKey(t, testAESKey32)
	cfg := &DataSourceConfig{
		Type: "feishu",
		Credentials: map[string]interface{}{
			"token":           "ghp-real-secret",
			"client_secret":   "cs-real-secret",
			"refresh_count":   42,   // non-string, must pass through untouched
			"refresh_enabled": true, // non-string, must pass through untouched
		},
		Settings: map[string]interface{}{"branch": "main"},
	}
	blob, err := cfg.ToJSON()
	assert.NoError(t, err)
	s := string(blob)

	// Plaintext secrets must not appear in the persisted JSON.
	assert.NotContains(t, s, "ghp-real-secret")
	assert.NotContains(t, s, "cs-real-secret")

	// String credentials are encrypted with the enc:v1: prefix.
	var raw map[string]json.RawMessage
	assert.NoError(t, json.Unmarshal(blob, &raw))
	var creds map[string]interface{}
	assert.NoError(t, json.Unmarshal(raw["credentials"], &creds))
	assert.True(t, strings.HasPrefix(creds["token"].(string), "enc:v1:"))
	assert.True(t, strings.HasPrefix(creds["client_secret"].(string), "enc:v1:"))

	// Non-string credentials survive verbatim (some connectors persist
	// integers / bools alongside the secrets — refresh_count and friends).
	assert.EqualValues(t, 42, creds["refresh_count"])
	assert.EqualValues(t, true, creds["refresh_enabled"])

	// In-memory caller's map must not be mutated.
	assert.Equal(t, "ghp-real-secret", cfg.Credentials["token"])
	assert.Equal(t, "cs-real-secret", cfg.Credentials["client_secret"])
}

func TestDataSourceConfig_ToJSON_PassthroughWhenNoKey(t *testing.T) {
	withAESKey(t, "")
	cfg := &DataSourceConfig{
		Credentials: map[string]interface{}{"token": "plain-token"},
	}
	blob, err := cfg.ToJSON()
	assert.NoError(t, err)
	assert.Contains(t, string(blob), "plain-token")
}

func TestFeishuCredentialsEncryption(t *testing.T) {
	withAESKey(t, testAESKey32)
	cfg := DataSourceConfig{Type: ConnectorTypeFeishu, Credentials: map[string]any{
		"app_id": "cli_test", "app_secret": "app-secret-for-tests",
	}}
	blob, err := cfg.ToJSON()
	assert.NoError(t, err)
	assert.NotContains(t, string(blob), "app-secret-for-tests")
	parsed, err := (&DataSource{Config: blob}).ParseConfig()
	assert.NoError(t, err)
	assert.Equal(t, "app-secret-for-tests", parsed.Credentials["app_secret"])
	assert.Equal(t, "app-secret-for-tests", cfg.Credentials["app_secret"])
}

func TestDataSource_ParseConfig_Roundtrip(t *testing.T) {
	withAESKey(t, testAESKey32)
	original := &DataSourceConfig{
		Type: "feishu",
		Credentials: map[string]interface{}{
			"token":   "tok-roundtrip",
			"team_id": "team-public-id",
		},
		ResourceIDs: []string{"page-1"},
	}
	blob, err := original.ToJSON()
	assert.NoError(t, err)

	ds := &DataSource{Config: blob}
	parsed, err := ds.ParseConfig()
	assert.NoError(t, err)
	assert.Equal(t, "tok-roundtrip", parsed.Credentials["token"])
	assert.Equal(t, "team-public-id", parsed.Credentials["team_id"])
	assert.Equal(t, []string{"page-1"}, parsed.ResourceIDs)
}

func TestDataSource_ParseConfig_LegacyPlaintext(t *testing.T) {
	// Rows persisted before encryption was wired in have no enc:v1: prefix
	// — DecryptStoredSecret returns them verbatim. Historical data must
	// keep working without an offline migration.
	withAESKey(t, testAESKey32)
	legacy := JSON([]byte(`{"type":"feishu","credentials":{"token":"legacy-plain"}}`))
	ds := &DataSource{Config: legacy}
	parsed, err := ds.ParseConfig()
	assert.NoError(t, err)
	assert.Equal(t, "legacy-plain", parsed.Credentials["token"])
}

func TestDataSource_ParseConfig_MissingKeyDegradesGracefully(t *testing.T) {
	// Encrypt with key, drop the key, then load. ParseConfig must succeed
	// so the data source row is still visible — the credential blanks
	// out so HasCredentials() returns false and the UI shows
	// "not configured". A loud Scan failure would break the whole list.
	withAESKey(t, testAESKey32)
	cfg := &DataSourceConfig{
		Type: "feishu",
		Credentials: map[string]interface{}{
			"token":         "secret",
			"refresh_count": 42, // non-string survives regardless
		},
		ResourceIDs: []string{"repo-1"},
	}
	blob, err := cfg.ToJSON()
	assert.NoError(t, err)

	withAESKey(t, "")
	ds := &DataSource{Config: blob}
	parsed, err := ds.ParseConfig()
	assert.NoError(t, err)
	assert.Equal(t, "", parsed.Credentials["token"], "encrypted credential must blank out, not leak ciphertext")
	assert.EqualValues(t, 42, parsed.Credentials["refresh_count"], "non-string credentials unaffected")
	assert.Equal(t, []string{"repo-1"}, parsed.ResourceIDs, "non-credential config survives")
	assert.False(t, parsed.HasCredentials() && parsed.Credentials["token"] != "",
		"HasCredentials should not report a usable credential when decrypt failed")
}

func TestDataSourceConfig_HasConfiguredCredentials(t *testing.T) {
	empty := DataSourceConfig{Credentials: nil}
	assert.False(t, empty.HasConfiguredCredentials(ConnectorTypeFeishu))

	withSecret := DataSourceConfig{
		Credentials: map[string]interface{}{"app_secret": "s"},
	}
	assert.True(t, withSecret.HasConfiguredCredentials(ConnectorTypeFeishu))
}

// TestSubtreeChildID_MatchesPrefix locks the producer/consumer contract: a child
// ID built by SubtreeChildID must start with the parent's SubtreeChildPrefix
// (what the sweep queries), and the prefix must not match a sibling node whose
// external_id merely shares a leading substring.
func TestSubtreeChildID_MatchesPrefix(t *testing.T) {
	parent := "node123"
	prefix := SubtreeChildPrefix(parent)
	assert.Equal(t, "node123#", prefix)

	fileChild := SubtreeChildID(parent, "file", "tokABC")
	imageChild := SubtreeChildID(parent, "image", "tokXYZ")
	assert.Equal(t, "node123#file#tokABC", fileChild)
	assert.Equal(t, "node123#image#tokXYZ", imageChild)

	// Every child of this node is swept by the prefix.
	assert.True(t, strings.HasPrefix(fileChild, prefix))
	assert.True(t, strings.HasPrefix(imageChild, prefix))

	// A sibling node "node1234" (shares the "node123" leading substring but is a
	// different node) must NOT be caught by node123's prefix — the '#' guards it.
	sibling := SubtreeChildID("node1234", "file", "tokABC")
	assert.False(t, strings.HasPrefix(sibling, prefix),
		"sibling node whose id shares a leading substring must not match the prefix")
}
