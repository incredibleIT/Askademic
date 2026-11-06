import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { fileURLToPath } from 'node:url'
import { runInNewContext } from 'node:vm'
import test from 'node:test'
import { compileScript, parse } from '@vue/compiler-sfc'
import ts from 'typescript'
import { createRenderer, h, nextTick, reactive, ref } from 'vue'

const require = createRequire(import.meta.url)
const filename = fileURLToPath(new URL('./DataSourceEditorDialog.vue', import.meta.url))
const { descriptor } = parse(readFileSync(filename, 'utf8'), { filename })
const script = compileScript(descriptor, { id: 'datasource-editor-test' }).content
  .replace('__expose();', '')
  .replace('return __returned__', '__expose(__returned__); return __returned__')
const compiled = ts.transpileModule(script, {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
}).outputText

// Lets un-awaited API calls (e.g. loadResources fired by nextStep) settle.
const flush = async () => {
  await new Promise(resolve => setImmediate(resolve))
  await nextTick()
}

async function fixture({
  configured = true,
  create = false,
  type = 'feishu',
  settings = {},
  resources = [],
  dataSource = null,
}: {
  configured?: boolean
  create?: boolean
  type?: string
  settings?: Record<string, unknown>
  resources?: any[]
  dataSource?: any
} = {}) {
  const calls: Array<{ method: string; args: any[] }> = []
  const warnings: string[] = []
  let storedToken = configured ? 'expired-token' : ''
  const api = {
    async validateCredentials(type: string, credentials: Record<string, string>) {
      calls.push({ method: 'validateCredentials', args: [type, { ...credentials }] })
      if (credentials.app_secret !== 'rotated-token') throw new Error('feishu API /app: status 401')
    },
    async validateConnection(id: string) {
      calls.push({ method: 'validateConnection', args: [id] })
      if (storedToken !== 'rotated-token') throw new Error('feishu API /app: status 401')
    },
    async updateDataSource(id: string, data: any) {
      calls.push({ method: 'updateDataSource', args: [id, JSON.parse(JSON.stringify(data))] })
      // The main update endpoint deliberately preserves stored credentials.
    },
    async putDataSourceCredentials(id: string, credentials: Record<string, string>) {
      calls.push({ method: 'putDataSourceCredentials', args: [id, { ...credentials }] })
      storedToken = credentials.app_secret
    },
    async createDataSource(data: any) {
      calls.push({ method: 'createDataSource', args: [JSON.parse(JSON.stringify(data))] })
      return { data: { id: 'source-temp' } }
    },
    // Lazy picker: the first call lists roots, later calls list one level.
    async listResources(id: string, parentId?: string) {
      calls.push({ method: 'listResources', args: [id, parentId] })
      return { data: resources.filter(r => (parentId ? r.parent_id === parentId : !r.parent_id)) }
    },
    async resolveResourceAncestors(id: string, ids: string[]) {
      calls.push({ method: 'resolveResourceAncestors', args: [id, ids] })
      return { data: { ancestors: [] } }
    },
  }
  const props = reactive({
    visible: false, kbId: 'kb-one',
    dataSource: create ? null : dataSource || {
      id: 'source-one', name: 'Feishu', type,
      credentials: { credentials: { configured } },
      config: { resource_ids: [], settings },
      sync_schedule: '0 0 */6 * * *', sync_mode: 'incremental',
      conflict_strategy: 'overwrite', sync_deletions: true,
    },
  })
  const exports: any = {}
  runInNewContext(compiled, {
    exports,
    require(name: string) {
      if (name === 'vue') return require('vue')
      if (name === 'vue-i18n') return { useI18n: () => ({ t: (key: string) => key }) }
      if (name === 'tdesign-vue-next') return { MessagePlugin: { warning(msg: string) { warnings.push(msg) }, success() {}, error() {} } }
      if (name === '@/api/datasource') return api
      return { default: {} }
    },
    URL, console,
  })
  const component = exports.default
  component.render = () => null
  const renderer = createRenderer<any, any>({
    createElement: () => ({}), createText: () => ({}), createComment: () => ({}),
    insert() {}, remove() {}, setElementText() {}, setText() {}, patchProp() {},
    parentNode: () => null, nextSibling: () => null,
  })
  const instance = ref<any>()
  const app = renderer.createApp({ render: () => h(component, { ...props, ref: instance }) })
  app.mount({})
  props.visible = true
  await nextTick()
  const vm = instance.value
  async function replace(token = 'rotated-token') {
    vm.enterReplaceCredentials()
    vm.form.config.credentials = { app_id: 'cli_xxxx', app_secret: token }
    await nextTick()
  }
  return { vm, calls, warnings, replace, storedToken: () => storedToken, close: () => app.unmount() }
}

test('rotated Feishu credentials are tested without updating the saved data source', async () => {
  const f = await fixture()
  try {
    await f.replace()
    await f.vm.testConnection()
    assert.equal(f.vm.testResult, 'success')
    assert.deepEqual(f.calls, [{ method: 'validateCredentials', args: ['feishu', {
      app_id: 'cli_xxxx', app_secret: 'rotated-token',
    }] }])
    assert.equal(f.storedToken(), 'expired-token')
  } finally { f.close() }
})

test('Next tests the replacement and final save commits credentials before settings', async () => {
  const f = await fixture()
  try {
    await f.replace()
    await f.vm.nextStep()
    assert.equal(f.vm.step, 2)
    await f.vm.nextStep()
    assert.equal(f.vm.step, 3)
    await flush()
    await f.vm.handleSubmit()
    assert.deepEqual(f.calls.map(call => call.method), [
      'validateCredentials', 'listResources', 'putDataSourceCredentials', 'updateDataSource',
    ])
    assert.equal(f.storedToken(), 'rotated-token')
    assert.deepEqual(f.calls[3].args[1].config.credentials, {})
  } finally { f.close() }
})

test('invalid replacement stays on the credentials step and can be corrected', async () => {
  const f = await fixture()
  try {
    await f.replace('invalid-token')
    await f.vm.nextStep()
    assert.equal(f.vm.step, 1)
    assert.equal(f.vm.testResult, 'error')
    assert.match(f.vm.testErrorMsg, /401/)
    assert.equal(f.storedToken(), 'expired-token')
    await f.replace()
    await f.vm.nextStep()
    assert.equal(f.vm.step, 2)
  } finally { f.close() }
})

test('testing unchanged credentials still validates the stored token', async () => {
  const f = await fixture()
  try {
    await f.vm.testConnection()
    assert.equal(f.vm.testResult, 'error')
    assert.deepEqual(f.calls.map(call => call.method), ['updateDataSource', 'validateConnection'])
  } finally { f.close() }
})

test('an existing data source with no saved credentials tests the entered token', async () => {
  const f = await fixture({ configured: false })
  try {
    await f.replace()
    await f.vm.testConnection()
    assert.equal(f.vm.testResult, 'success')
    assert.deepEqual(f.calls.map(call => call.method), ['validateCredentials'])
    assert.equal(f.storedToken(), '')
  } finally { f.close() }
})

test('new Feishu data sources continue to test credentials without persistence', async () => {
  const f = await fixture({ create: true })
  try {
    f.vm.selectType(f.vm.connectorDefs.find((def: any) => def.type === 'feishu'))
    await f.replace()
    await f.vm.testConnection()
    assert.equal(f.vm.testResult, 'success')
    assert.deepEqual(f.calls.map(call => call.method), ['validateCredentials'])
  } finally { f.close() }
})

test('the single-library guard leaves Feishu and Lark Drive selections alone', async () => {
  for (const type of ['feishu_drive', 'lark_drive']) {
    const f = await fixture({ create: true })
    try {
      f.vm.selectType(f.vm.connectorDefs.find((def: any) => def.type === type))
      // Drive IDs are "folderToken:fileToken"; roots under two folders must
      // both stay selectable.
      f.vm.resources = [
        { external_id: 'folderA:fileA', name: 'A', type: 'file', has_children: false },
        { external_id: 'folderB:fileB', name: 'B', type: 'file', has_children: false },
      ]
      await nextTick()
      f.vm.toggleResource('folderA:fileA')
      f.vm.toggleResource('folderB:fileB')
      assert.deepEqual([...f.vm.selectedResourceIds].sort(), ['folderA:fileA', 'folderB:fileB'], type)
      assert.deepEqual(f.warnings, [], type)
    } finally { f.close() }
  }
})

test('a DingTalk data source reports the uploaded-file switch, defaulting to off', async () => {
  const off = await fixture({ type: 'dingtalk', settings: {} })
  try {
    // The connector treats a missing key as false, so an existing source that
    // predates this control must render as off — and stay untouched.
    assert.equal(off.vm.dingtalkIncludeUploadedFiles, false)
    assert.equal(off.vm.form.config.settings.include_uploaded_files, undefined)
  } finally { off.close() }

  const on = await fixture({ type: 'dingtalk', settings: { include_uploaded_files: true } })
  try {
    assert.equal(on.vm.dingtalkIncludeUploadedFiles, true)
  } finally { on.close() }

  const explicitOff = await fixture({ type: 'dingtalk', settings: { include_uploaded_files: false } })
  try {
    assert.equal(explicitOff.vm.dingtalkIncludeUploadedFiles, false)
  } finally { explicitOff.close() }
})

test('toggling the DingTalk uploaded-file switch writes the settings key', async () => {
  const f = await fixture({ type: 'dingtalk', settings: {} })
  try {
    f.vm.dingtalkIncludeUploadedFiles = true
    assert.equal(f.vm.form.config.settings.include_uploaded_files, true)
    f.vm.dingtalkIncludeUploadedFiles = false
    assert.equal(f.vm.form.config.settings.include_uploaded_files, false)
    // The switch is scoped to DingTalk: no other connector's key is written.
    assert.equal(f.vm.form.config.settings.include_sheets, undefined)
  } finally { f.close() }
})

for (const type of ['feishu', 'lark', 'feishu_drive', 'lark_drive']) {
  test(`${type} preserves inherited parse mode when saving unrelated settings`, async () => {
    const settings = { timezone: 'Asia/Shanghai', resync_required: true }
    const f = await fixture({ type, settings })
    try {
      assert.equal(f.vm.feishuParseMode, '')
      assert.equal(f.vm.form.config.settings.parse_mode, undefined)
      f.vm.form.name = 'Renamed source'
      await f.vm.handleSubmit()
      const saved = f.calls.find(call => call.method === 'updateDataSource')
      assert.ok(saved)
      assert.deepEqual(saved.args[1].config.settings, settings)
      assert.equal('parse_mode' in settings, false, 'opening the editor must not mutate props')
    } finally { f.close() }
  })
}

test('Feishu explicit mode changes are saved without mutating the source configuration', async () => {
  const settings = { parse_mode: 'export', timezone: 'Asia/Shanghai' }
  const f = await fixture({ type: 'feishu', settings })
  try {
    assert.equal(f.vm.feishuParseMode, 'export')
    f.vm.feishuParseMode = 'blocks'
    assert.equal(settings.parse_mode, 'export')
    assert.equal(f.vm.buildConfigPayload().settings.parse_mode, 'blocks')
    f.vm.feishuParseMode = ''
    assert.deepEqual(JSON.parse(JSON.stringify(f.vm.buildConfigPayload().settings)), { timezone: 'Asia/Shanghai' })
    assert.equal(settings.parse_mode, 'export')
  } finally { f.close() }
})

test('new Feishu sources keep the blocks default and do not leak it to another connector', async () => {
  const f = await fixture({ create: true })
  try {
    f.vm.selectType(f.vm.connectorDefs.find((def: any) => def.type === 'feishu'))
    assert.equal(f.vm.feishuParseMode, 'blocks')
    f.vm.selectType(f.vm.connectorDefs.find((def: any) => def.type === 'dingtalk'))
    assert.equal(f.vm.form.config.settings.parse_mode, undefined)
  } finally { f.close() }
})

