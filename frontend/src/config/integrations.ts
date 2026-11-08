import type { DeploymentCapabilityKey } from './deploymentCapabilities'

export const CHROME_EXTENSION_URL =
  'https://chromewebstore.google.com/detail/jpemjbopikggjlmikmclgbmkhhopjdgd?utm_source=item-share-cb'

// ClawHub 技能卡已随 'claw' tab 移除;ClawHub 作为技能安装注册源(tenant_skill_source)不受影响。
export type IntegrationTab = 'im' | 'embed' | 'api' | 'mcpserver' | 'cli' | 'chrome' | 'claw'

// Askademic 剪裁(2026-11-06/11-08):im/cli/claw 隐藏,embed 整体移除。
// im=IM 渠道(教务不用,后端 26787 行保留);cli=教用户安装已删除的 CLI;
// claw=ClawHub 的 WeKnora 品牌技能卡。恢复=把对应条目加回下面两个数组。
export const INTEGRATION_TABS: IntegrationTab[] = ['api', 'mcpserver', 'chrome']

/** Aligns with Settings.vue SECTION_MIN_ROLE.api and router.go g.Owner() on /api-principal-config. */
export type IntegrationTabRole = 'viewer' | 'contributor' | 'admin' | 'owner'

export const INTEGRATION_TAB_MIN_ROLE: Partial<Record<IntegrationTab, IntegrationTabRole>> = {
  api: 'owner',
}

export const INTEGRATION_TAB_CAPABILITY: Partial<Record<IntegrationTab, DeploymentCapabilityKey>> = {
  im: 'integrations.im',
  api: 'integrations.api',
  mcpserver: 'integrations.mcpserver',
}

export type IntegrationPreviewIcon =
  | { type: 'icon'; name: string }
  | { type: 'emoji'; value: string }

/** Sidebar hover preview + Integrations modal nav — add new entries here. */
export const INTEGRATION_PREVIEW_ITEMS: Array<{
  key: IntegrationTab
  icon: IntegrationPreviewIcon
}> = [
  { key: 'api', icon: { type: 'icon', name: 'secured' } },
  { key: 'mcpserver', icon: { type: 'icon', name: 'tools' } },
  { key: 'chrome', icon: { type: 'icon', name: 'extension' } },
]
