import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildSettingsRouteQuery,
  integrationSectionKey,
  isIntegrationSection,
  normalizeSettingsSection,
  settingsQueryUnchanged,
} from './settingsRoute'

test('settings nav writes section, and keeps a system-settings panel only there', () => {
  assert.deepEqual(
    buildSettingsRouteQuery('models', {
      section: 'system-global',
      tab: 'im',
      agentId: 'agt_1',
    }),
    { section: 'models' },
  )
  assert.deepEqual(
    buildSettingsRouteQuery('general', { section: 'system-global' }),
    { section: 'general' },
  )
  assert.deepEqual(
    buildSettingsRouteQuery('runtime-queues', { section: 'system-global' }),
    { section: 'runtime-queues' },
  )
  assert.deepEqual(
    buildSettingsRouteQuery('system-global', { section: 'system-global', panel: 'users', tab: 'im' }),
    { section: 'system-global', panel: 'users' },
  )
  assert.deepEqual(
    buildSettingsRouteQuery('models', { section: 'system-global', panel: 'users' }),
    { section: 'models' },
  )
  assert.deepEqual(
    buildSettingsRouteQuery('system-global', { section: 'system-global', panel: 'billing' }),
    { section: 'system-global' },
  )
  assert.deepEqual(
    buildSettingsRouteQuery(integrationSectionKey('claw'), {
      section: 'integrations',
      tab: 'im',
    }),
    { section: 'integration-claw' },
  )
  assert.deepEqual(
    buildSettingsRouteQuery(integrationSectionKey('api'), {
      section: 'integrations',
      tab: 'im',
      agentId: 'agt_1',
    }),
    { section: 'integration-api', agentId: 'agt_1' },
  )
})

test('legacy api / integrations / bare-tab query strings normalize to nav keys', () => {
  assert.equal(normalizeSettingsSection('api'), 'integration-api')
  // Askademic 剪裁(2026-11-06/11-08):im/cli/claw 已隐藏,embed 已删除——裸 tab 名
  // 不再映射到 integration-*;旧书签/integrations 默认落第一个可见 tab(api)。
  assert.equal(normalizeSettingsSection('claw'), 'claw')
  assert.equal(normalizeSettingsSection('cli'), 'cli')
  assert.equal(normalizeSettingsSection('embed'), 'embed')
  assert.equal(normalizeSettingsSection('mcpserver'), 'integration-mcpserver')
  // 'mcp' is the MCP-client settings page, not the MCP server integration tab.
  assert.equal(normalizeSettingsSection('mcp'), 'mcp')
  assert.equal(normalizeSettingsSection('integrations', 'cli'), 'integration-api')
  assert.equal(normalizeSettingsSection('integrations', 'embed'), 'integration-api')
  assert.equal(normalizeSettingsSection('integrations'), 'integration-api')
  assert.equal(normalizeSettingsSection('integration-chrome'), 'integration-chrome')
  assert.equal(normalizeSettingsSection('system-global'), 'system-global')
  assert.equal(isIntegrationSection('integration-chrome'), true)
  assert.equal(isIntegrationSection('integration-cli'), false)
  assert.equal(isIntegrationSection('integration-embed'), false)
  assert.equal(isIntegrationSection('models'), false)
  assert.equal(isIntegrationSection('integration-unknown'), false)
})

test('canonical settings query skips a redundant replace', () => {
  assert.equal(
    settingsQueryUnchanged(
      { section: 'integration-claw' },
      { section: 'integration-claw' },
    ),
    true,
  )
  assert.equal(
    settingsQueryUnchanged(
      { section: 'integrations', tab: 'claw' },
      { section: 'integration-claw' },
    ),
    false,
  )
  assert.equal(
    settingsQueryUnchanged(
      { section: 'system-global', panel: 'users' },
      { section: 'system-global', panel: 'access' },
    ),
    false,
  )
})
