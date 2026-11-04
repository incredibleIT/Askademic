import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./Login.vue', import.meta.url), 'utf8')
const stackStyles = source.slice(
  source.indexOf('@media (max-width: 1000px)'),
  source.indexOf('@media (max-width: 480px)'),
)
const phoneStyles = source.slice(source.indexOf('@media (max-width: 480px)'))

test('narrow login stacks the stage so the form column comes before the product sheet', () => {
  // 桌面端 stage 用 row-reverse（表单居右）；窄屏改 column 后，
  // DOM 顺序（表单在前）自然把表单排到产品切片上方，无需 order 补丁。
  assert.match(stackStyles, /\.stage\s*\{[\s\S]*?flex-direction:\s*column;/)
  const desktopStage = source.slice(
    source.indexOf('.stage {'),
    source.indexOf('@media'),
  )
  assert.match(desktopStage, /flex-direction:\s*row-reverse;/)
})

test('phone login keeps the language switch compact and clear of the form', () => {
  assert.match(phoneStyles, /\.lang-switch\s*\{[\s\S]*?top:\s*14px;/)
})
