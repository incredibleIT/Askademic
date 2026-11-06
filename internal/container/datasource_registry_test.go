package container

import (
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestConnectorRegistryIncludesDingTalk(t *testing.T) {
	registry, err := initConnectorRegistry()
	if err != nil {
		t.Fatalf("initConnectorRegistry() error = %v", err)
	}
	connector, err := registry.Get(types.ConnectorTypeDingTalk)
	if err != nil {
		t.Fatalf("DingTalk connector is not registered: %v", err)
	}
	if connector.Type() != types.ConnectorTypeDingTalk {
		t.Fatalf("connector.Type() = %q", connector.Type())
	}
}

func TestConnectorRegistryIncludesFeishu(t *testing.T) {
	registry, err := initConnectorRegistry()
	if err != nil {
		t.Fatalf("initConnectorRegistry() error = %v", err)
	}
	connector, err := registry.Get(types.ConnectorTypeFeishu)
	if err != nil {
		t.Fatalf("Feishu connector is not registered: %v", err)
	}
	if connector.Type() != types.ConnectorTypeFeishu {
		t.Fatalf("connector.Type() = %q", connector.Type())
	}
}

// Askademic 剪裁(2026-11-06):数据源只保留飞书家族与钉钉,其余连接器必须
// 不在注册表中——防止被删目录的注册代码被无意恢复。
func TestConnectorRegistryExcludesRemovedConnectors(t *testing.T) {
	registry, err := initConnectorRegistry()
	if err != nil {
		t.Fatalf("initConnectorRegistry() error = %v", err)
	}
	for _, removed := range []string{"notion", "confluence", "yuque", "ima", "rss", "gitlab", "seafile", "nextcloud", "github", "google_drive", "onedrive", "web_crawler", "slack", "imap"} {
		if _, err := registry.Get(removed); err == nil {
			t.Fatalf("connector %q should be removed but is registered", removed)
		}
	}
}
