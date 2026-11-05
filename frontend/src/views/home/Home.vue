<template>
  <div class="ad-home">
    <div class="ad-column">
      <!-- 问候:衬线大字是 h1(方向文件 signature);事实行只陈述真实日期 -->
      <header class="ad-greeting">
        <h1>{{ greetingLine }}</h1>
        <p class="ad-factline">{{ dateLine }}</p>
      </header>

      <!-- 提问框:真实的 createSessions → 会话页(复用 creatChat 的首问机制) -->
      <section class="ad-composer">
        <textarea v-model="draft" rows="1" :placeholder="t('home.composerPlaceholder')"
          :aria-label="t('home.composerPlaceholder')" @keydown.enter.exact.prevent="sendQuestion"></textarea>
        <div class="ad-composer-bar">
          <button class="ad-send" :disabled="!draft.trim() || sending" @click="sendQuestion">
            {{ sending ? t('home.composerSending') : t('home.composerSend') }}
          </button>
        </div>
        <div v-if="suggestions.length" class="ad-chips">
          <button v-for="item in suggestions" :key="item.question" class="ad-chip" @click="sendSuggestion(item)">
            {{ item.question }}
          </button>
        </div>
      </section>

      <!-- 数据 tiles:知识库 / 文档 / 会话,全部来自真实接口;加载中显示 — -->
      <div class="ad-grid ad-tiles">
        <div class="ad-tile span4">
          <div class="k">{{ t('home.tileKb') }}</div>
          <div class="v">{{ fmtNum(kbCount) }}</div>
          <div class="c">{{ t('home.tileKbCaption') }}</div>
        </div>
        <div class="ad-tile span4">
          <div class="k">{{ t('home.tileDocs') }}</div>
          <div class="v">{{ fmtNum(kbDocs) }}</div>
          <div class="c">{{ kbCount ? t('home.tileDocsCaption', { n: kbCount }) : '' }}</div>
        </div>
        <div class="ad-tile span4">
          <div class="k">{{ t('home.tileSessions') }}</div>
          <div class="v">{{ fmtNum(sessionTotal) }}</div>
          <div class="c">{{ t('home.tileSessionsCaption') }}</div>
        </div>
      </div>

      <!-- 9+3 行:最近会话 + 知识库动态(动态无数据时整行让给会话面板) -->
      <div class="ad-grid">
        <section class="ad-panel" :class="recentDocs.length ? 'span9' : 'span12'">
          <div class="ad-panel-head">
            <h2>{{ t('home.recentSessions') }}</h2>
            <span v-if="sessionTotal !== null" class="ad-chip ad-chip--static">{{ t('home.countChip', { n: sessionTotal }) }}</span>
          </div>
          <div v-if="sessionsLoaded && sessions.length === 0" class="ad-empty">{{ t('home.noSessionsYet') }}</div>
          <div v-else class="ad-panel-list">
            <button v-for="s in sessions" :key="s.id" class="ad-rowcard" type="button" @click="openSession(s.id)">
              <div class="rc-main">
                <div class="rc-title">{{ s.title || t('menu.newSession') }}</div>
              </div>
              <div class="rc-side">{{ formatWhen(s.updated_at || s.created_at) }}</div>
            </button>
          </div>
        </section>
        <section v-if="recentDocs.length" class="ad-panel span3">
          <div class="ad-panel-head">
            <h2>{{ t('home.kbActivity') }}</h2>
          </div>
          <div class="ad-panel-list">
            <button v-for="(doc, idx) in recentDocs" :key="doc.kbId + '-' + idx" class="ad-rowcard" type="button"
              @click="openKb(doc.kbId)">
              <div class="rc-main">
                <div class="rc-title">{{ doc.name }}</div>
                <div class="rc-sub">{{ doc.kbName }}</div>
              </div>
              <div class="rc-side">{{ formatShortDate(doc.date) }}</div>
            </button>
          </div>
        </section>
      </div>

      <!-- 12 列行:空间知识库(含组织共享来的库;面板标题复用 menu.knowledgeBase) -->
      <div class="ad-grid">
        <section class="ad-panel span12">
          <div class="ad-panel-head">
            <h2>{{ t('menu.knowledgeBase') }}</h2>
            <span v-if="kbCount !== null" class="ad-chip ad-chip--static">{{ t('home.countChip', { n: kbCount }) }}</span>
            <router-link class="ad-all" to="/platform/knowledge-bases">{{ t('home.manage') }}</router-link>
          </div>
          <div v-if="kbsLoaded && kbRows.length === 0" class="ad-empty">{{ t('home.noKbs') }}</div>
          <button v-for="row in kbRows" :key="row.id" class="ad-rowcard" type="button" @click="openKb(row.id)">
            <div class="rc-main">
              <div class="rc-title">{{ row.name }}</div>
              <div v-if="row.sharedOrgName" class="rc-sub">{{ t('home.sharedFrom', { name: row.sharedOrgName }) }}</div>
            </div>
            <div class="rc-side">{{ t('knowledgeBase.documentCount', { count: row.docs }) }}</div>
          </button>
        </section>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// Askademic 首页(Phase C)。设计:.taste-skill/app/home.html(editorial-greeting,
// seed 522964766)。数据诚实:课表/教务日程/教学周无数据源,不做;数字全部来自
// listKnowledgeBases / getSessionsList / listKnowledgeFiles 的真实返回。
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import { createSessions, getSessionsList } from '@/api/chat/index'
import { listKnowledgeBases, listKnowledgeFiles } from '@/api/knowledge-base/index'
import { getSuggestedQuestions } from '@/api/agent/index'
import type { SuggestedQuestion } from '@/api/agent/index'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { useMenuStore } from '@/stores/menu'
import { useOrganizationStore } from '@/stores/organization'
import { BRAND } from '@/config/brand'

const { t, locale } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const usemenuStore = useMenuStore()
const orgStore = useOrganizationStore()

/* ---------- 问候 ---------- */
const greeting = computed(() => {
  const h = new Date().getHours()
  if (h < 11) return t('home.greetingMorning')
  if (h < 18) return t('home.greetingAfternoon')
  return t('home.greetingEvening')
})
const tenantName = computed(() => authStore.tenant?.name || BRAND.name)
// 标点随语言包走:中文全角逗号+句号,西文半角
const greetingLine = computed(() => t('home.greetingFormat', { greeting: greeting.value, name: tenantName.value }))
const dateLine = computed(() => {
  try {
    return new Intl.DateTimeFormat(locale.value, { year: 'numeric', month: 'long', day: 'numeric', weekday: 'long' }).format(new Date())
  } catch {
    return ''
  }
})

/* ---------- 提问框 ---------- */
const draft = ref('')
const sending = ref(false)
const suggestions = ref<SuggestedQuestion[]>([])

const sendSuggestion = (item: SuggestedQuestion) => {
  if (sending.value) return
  draft.value = item.question
  void sendQuestion()
}

const sendQuestion = async () => {
  const value = draft.value.trim()
  if (!value || sending.value) return
  sending.value = true
  try {
    // 会话配置与 creatChat.createNewSession 同源:当前选中的知识库/文件与 Agent 配置
    const selectedKbs = settingsStore.settings.selectedKnowledgeBases || []
    const selectedFiles = settingsStore.settings.selectedFiles || []
    const sessionData: Record<string, unknown> = {
      agent_config: {
        enabled: true,
        max_iterations: settingsStore.agentConfig.maxIterations,
        temperature: settingsStore.agentConfig.temperature,
        knowledge_bases: selectedKbs,
        knowledge_ids: selectedFiles,
        allowed_tools: settingsStore.agentConfig.allowedTools,
      },
    }
    const res: any = await createSessions(sessionData)
    if (res?.data?.id) {
      const sessionId = String(res.data.id)
      const now = new Date().toISOString()
      usemenuStore.updataMenuChildren({
        title: t('createChat.newSessionTitle'),
        path: `chat/${sessionId}`,
        id: sessionId,
        isMore: false,
        isNoTitle: true,
        created_at: now,
        updated_at: now,
      })
      usemenuStore.changeIsFirstSession(true)
      usemenuStore.changeFirstQuery(value, [], '', [], [], null)
      draft.value = ''
      await router.push(`/platform/chat/${sessionId}`)
    } else {
      MessagePlugin.error(t('home.askFailed'))
    }
  } catch (error) {
    console.error('[home] create session error:', error)
    MessagePlugin.error(t('home.askFailed'))
  } finally {
    sending.value = false
  }
}

/* ---------- 数据加载 ---------- */
const kbs = ref<any[]>([])
const kbsLoaded = ref(false)
const kbCount = ref<number | null>(null)
const kbDocs = ref<number | null>(null)
const sessions = ref<any[]>([])
const sessionsLoaded = ref(false)
const sessionTotal = ref<number | null>(null)
const recentDocs = ref<Array<{ name: string; kbId: string; kbName: string; date: string }>>([])

// 共享知识库(经组织共享到本空间,如教务处的制度文库):与 KB 列表页同源
// (GET /api/v1/shared-knowledge-bases,store 内有并发去重),按用户要求并入
// 本空间知识库面板与计数。
interface HomeKbRow {
  id: string
  name: string
  docs: number
  sharedOrgName: string
}
const kbRows = computed<HomeKbRow[]>(() => {
  const rows: HomeKbRow[] = kbs.value.map(kb => ({
    id: String(kb.id),
    name: String(kb.name || ''),
    docs: Number(kb?.knowledge_count) || 0,
    sharedOrgName: '',
  }))
  for (const shared of orgStore.sharedKnowledgeBases) {
    const kb = shared.knowledge_base
    if (!kb || rows.some(row => row.id === String(kb.id))) continue
    rows.push({
      id: String(kb.id),
      name: String(kb.name || ''),
      docs: Number(kb?.knowledge_count) || 0,
      sharedOrgName: String(shared.org_name || ''),
    })
  }
  return rows
})

onMounted(async () => {
  // 知识库与文档数(计数含共享库,与下方面板一致)
  const computeKbCounts = () => {
    kbCount.value = kbRows.value.length
    kbDocs.value = kbRows.value.reduce((sum, row) => sum + row.docs, 0)
  }
  try {
    const res: any = await listKnowledgeBases()
    kbs.value = Array.isArray(res?.data) ? res.data : []
    computeKbCounts()
  } catch (error) {
    console.error('[home] listKnowledgeBases failed:', error)
  } finally {
    kbsLoaded.value = true
  }
  // 共享知识库:store 有版本化去重,Promise 拿到后重算计数
  try {
    await orgStore.fetchSharedKnowledgeBases()
    computeKbCounts()
  } catch (error) {
    console.error('[home] fetchSharedKnowledgeBases failed:', error)
  }

  // 最近会话(列表 + 真实总数):拉 10 条,面板限高 ~5 行、其余列表内滚动查看
  try {
    const res: any = await getSessionsList(1, 10)
    sessions.value = Array.isArray(res?.data) ? res.data : []
    sessionTotal.value = Number(res?.total ?? sessions.value.length)
  } catch (error) {
    console.error('[home] getSessionsList failed:', error)
  } finally {
    sessionsLoaded.value = true
  }

  // 知识库动态:各库最近更新的文档(全体成员可读;按库的审计日志仅创建者/管理员可见,
  // 学生/教师会 403,故这里用文档列表充当"动态",拿不到就整块隐藏)。共享库同样并入
  // —— 班级里最常更新的是教务处共享来的制度文库。
  const targets = kbRows.value.slice(0, 4)
  if (targets.length) {
    const settled = await Promise.allSettled(targets.map(async (row) => {
      const res: any = await listKnowledgeFiles(row.id, { page: 1, page_size: 4, sort_by: 'updated_at', sort_order: 'desc' })
      const rows = Array.isArray(res?.data) ? res.data : []
      return rows.map((row2: any) => ({
        name: String(row2?.file_name || row2?.knowledge_title || ''),
        kbId: row.id,
        kbName: row.name,
        date: String(row2?.updated_at || row2?.created_at || ''),
      }))
    }))
    const merged = settled
      .filter((r): r is PromiseFulfilledResult<any[]> => r.status === 'fulfilled')
      .flatMap(r => r.value)
      .filter(doc => doc.name)
    merged.sort((a, b) => (a.date < b.date ? 1 : -1))
    recentDocs.value = merged.slice(0, 8)
  }

  // 建议问题:来自当前 Agent 的真实配置(与 creatChat 同源);没有 Agent 就不出 chips
  const agentId = settingsStore.selectedAgentId
  if (agentId) {
    try {
      const res: any = await getSuggestedQuestions(agentId, settingsStore.getSuggestedQuestionsParams())
      suggestions.value = (res?.data?.questions || []).slice(0, 4)
    } catch {
      suggestions.value = []
    }
  }
})

/* ---------- 跳转 ---------- */
const openSession = (sessionId: string) => {
  void router.push(`/platform/chat/${sessionId}`)
}
const openKb = (kbId: string) => {
  void router.push(`/platform/knowledge-bases/${kbId}`)
}

/* ---------- 展示格式 ---------- */
const fmtNum = (n: number | null) => (n === null ? '—' : String(n))
const pad2 = (n: number) => String(n).padStart(2, '0')
const formatShortDate = (iso?: string) => {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return `${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`
}
const formatWhen = (iso?: string) => {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const startOfToday = new Date().setHours(0, 0, 0, 0)
  const time = `${pad2(d.getHours())}:${pad2(d.getMinutes())}`
  if (d.getTime() >= startOfToday) return `${t('time.today')} ${time}`
  if (d.getTime() >= startOfToday - 86400000) return `${t('time.yesterday')} ${time}`
  return formatShortDate(iso)
}
</script>

<style lang="less" scoped>
/* 编辑式问候外壳:白色 ground,居中 1100px 12 列。令牌全部取 --ad-*。
   注意:真实应用没有全局 border-box(踩坑记录 2),本页自行声明。 */
.ad-home {
  height: 100%;
  overflow-y: auto;
  background: var(--ad-ground);
  box-sizing: border-box;

  *,
  *::before,
  *::after {
    box-sizing: border-box;
  }
}

.ad-column {
  width: 1100px;
  max-width: calc(100% - 48px);
  margin: 0 auto;
  padding: 72px 0 64px;
}

.ad-grid {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  gap: 16px;
  /* 同一行的面板等高拉伸(默认 stretch):最近会话比知识库动态矮时,
     面板底色补齐整行,不留断开的空白 */
  margin-top: 16px;
}

.ad-tiles {
  margin-top: 50px;
}

.span3 { grid-column: span 3; }
.span4 { grid-column: span 4; }
.span9 { grid-column: span 9; }
.span12 { grid-column: span 12; }

/* ---------- 问候 ---------- */
.ad-greeting h1 {
  margin: 0;
  font: 700 48px/1.15 var(--ad-font-serif);
  letter-spacing: 0.2px;
  color: var(--ad-text);
}

.ad-factline {
  margin: 10px 0 0;
  font: 400 14px/1.6 var(--app-font-family);
  color: var(--ad-muted);
}

/* ---------- 提问框(唯一允许带阴影的浮动元素) ---------- */
.ad-composer {
  margin-top: 34px;
  background: var(--ad-raised);
  border: 1px solid var(--ad-border);
  border-radius: 20px;
  padding: 14px 16px 12px;
  box-shadow: 0 2px 6px rgba(31, 32, 29, 0.05);
}

.ad-composer textarea {
  display: block;
  width: 100%;
  border: 0;
  outline: 0;
  resize: none;
  background: transparent;
  font: 400 15px/1.6 var(--app-font-family);
  color: var(--ad-text);
  height: 24px;
}

.ad-composer textarea::placeholder {
  color: var(--ad-faint);
}

.ad-composer-bar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 10px;
}

.ad-send {
  height: 32px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: var(--ad-primary);
  color: #fff;
  font: 500 13px/1 var(--app-font-family);
  cursor: pointer;
  transition: background-color 150ms ease;
}

.ad-send:hover:not(:disabled) {
  background: var(--ad-primary-hover);
}

.ad-send:disabled {
  background: var(--ad-control);
  color: var(--ad-muted);
  cursor: not-allowed;
}

.ad-chips {
  display: flex;
  gap: 8px;
  margin-top: 12px;
  flex-wrap: wrap;
}

.ad-chip {
  border: 1px solid var(--ad-border);
  background: var(--ad-raised);
  border-radius: 999px;
  height: 28px;
  padding: 0 12px;
  font: 400 12px/1 var(--app-font-family);
  color: var(--ad-muted);
  cursor: pointer;
  transition: border-color 150ms ease, color 150ms ease;
}

.ad-chip:hover {
  border-color: var(--ad-control);
  color: var(--ad-text);
}

.ad-chip--static {
  cursor: default;
  background: transparent;
}

/* ---------- tiles ---------- */
.ad-tile {
  background: var(--ad-surface);
  border-radius: 22px;
  padding: 16px;
  height: 128px;
  display: flex;
  flex-direction: column;
}

.ad-tile .k {
  font: 400 13px/1.3 var(--app-font-family);
  color: var(--ad-muted);
}

.ad-tile .v {
  font: 400 32px/1.1 var(--app-font-family);
  font-variant-numeric: tabular-nums;
  color: var(--ad-text);
  margin-top: auto;
}

.ad-tile .c {
  font: 400 13px/1.4 var(--app-font-family);
  color: var(--ad-faint);
  margin-top: 4px;
  min-height: 18px;
}

/* ---------- panels + rowcards ---------- */
.ad-panel {
  background: var(--ad-surface);
  border-radius: 22px;
  padding: 20px;
  min-width: 0;
}

.ad-panel-head {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-bottom: 12px;
}

.ad-panel-head h2 {
  margin: 0;
  font: 600 14px/1.3 var(--app-font-family);
  color: var(--ad-text);
}

.ad-panel-head .ad-chip {
  margin-left: auto;
  height: 22px;
  padding: 0 9px;
  display: inline-flex;
  align-items: center;
}

.ad-panel-head .ad-all {
  margin-left: auto;
  font: 400 12px/1 var(--app-font-family);
  color: var(--ad-muted);
  text-decoration: none;
}

.ad-panel-head .ad-chip + .ad-all {
  margin-left: 0;
}

.ad-panel-head .ad-all:hover {
  color: var(--ad-text);
  text-decoration: underline;
  text-underline-offset: 3px;
}

/* 面板列表限高:条数多时面板不再无限下伸,列表内部滚动。
   5 行(5×63 + 4×8 = 347px),同行两面板由 grid stretch 保持等高。 */
.ad-panel-list {
  max-height: 350px;
  overflow-y: auto;
  min-width: 0;
  scrollbar-width: thin;
  scrollbar-color: var(--ad-control) transparent;
}

.ad-panel-list::-webkit-scrollbar {
  width: 6px;
}

.ad-panel-list::-webkit-scrollbar-thumb {
  background: var(--ad-control);
  border-radius: 999px;
}

.ad-panel-list::-webkit-scrollbar-track {
  background: transparent;
}

.ad-rowcard {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  text-align: left;
  background: var(--ad-raised);
  border: 1px solid var(--ad-border);
  border-radius: 14px;
  height: 63px;
  padding: 0 14px;
  cursor: pointer;
  font-family: var(--app-font-family);
  transition: border-color 150ms ease;
}

.ad-rowcard + .ad-rowcard {
  margin-top: 8px;
}

.ad-rowcard:hover {
  border-color: var(--ad-control);
}

.rc-main {
  min-width: 0;
}

.rc-title {
  font: 400 14px/1.4 var(--app-font-family);
  color: var(--ad-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.rc-sub {
  font: 400 13px/1.4 var(--app-font-family);
  color: var(--ad-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.rc-side {
  margin-left: auto;
  text-align: right;
  font: 400 13px/1.4 var(--app-font-family);
  color: var(--ad-muted);
  font-variant-numeric: tabular-nums;
  flex: none;
}

.ad-empty {
  display: grid;
  place-items: center;
  min-height: 96px;
  padding: 16px;
  border: 1px dashed var(--ad-control);
  border-radius: 16px;
  color: var(--ad-muted);
  font: 400 13px/1.6 var(--app-font-family);
  text-align: center;
}

/* ---------- phone ---------- */
@media (max-width: 760px) {
  .ad-column {
    width: auto;
    max-width: none;
    padding: 24px 16px 48px;
  }

  .ad-greeting h1 {
    font-size: 26px;
  }

  .ad-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .span9,
  .span12 {
    grid-column: span 2;
  }

  .span3 {
    grid-column: span 2;
  }

  .ad-tile.span4 {
    grid-column: span 1;
  }

  /* 3 块 tiles 在 2 列栅格下:最后一块(奇数)占满一行,不留半空行 */
  .ad-tile.span4:last-child:nth-child(odd) {
    grid-column: span 2;
  }
}
</style>
