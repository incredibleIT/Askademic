<script setup lang="ts">
import { onMounted, onUnmounted, ref, shallowRef } from 'vue'
import DocumentPreview from './document-preview.vue'
import { RESOURCE_PREVIEW_EVENT, type LoadedProtectedFile } from '@/utils/protectedResource'

const file = shallowRef<LoadedProtectedFile | null>(null)
const visible = ref(false)
const version = ref(0)
// 首次打开前不挂载抽屉：TDesign 关闭态的 t-drawer 会在 body 留下
// 一层 opacity:0 的全屏遮罩（t-drawer__mask，背景 rgba(0,0,0,.6)），
// 命中测试里会被当作盖住整页内容的元素（design-taste-frontend
// inspect 的 covered-text FAIL 即来源于此）。懒挂载后无此问题。
const hasOpened = ref(false)
function open(event: Event) {
  const detail = (event as CustomEvent<LoadedProtectedFile>).detail
  if (!detail || !(detail.blob instanceof Blob) || !detail.blobURL?.startsWith('blob:')) return
  event.preventDefault()
  file.value = detail
  version.value++
  hasOpened.value = true
  visible.value = true
}
onMounted(() => window.addEventListener(RESOURCE_PREVIEW_EVENT, open))
onUnmounted(() => window.removeEventListener(RESOURCE_PREVIEW_EVENT, open))
</script>

<template>
  <t-drawer v-if="hasOpened" v-model:visible="visible" :header="file?.fileName" size="70%"
    :z-index="2200" attach="body" :footer="false" destroy-on-close>
    <template v-if="file">
      <a :href="file.blobURL" :download="file.fileName">{{ $t('agent.artifactDrawer.download') }}</a>
      <DocumentPreview :key="version" :source-blob="file.blob" :file-name="file.fileName"
        :file-type="file.blob.type" :active="visible" />
    </template>
  </t-drawer>
</template>

<style>
.protected-resource-card {
  display: inline-flex;
  align-items: center;
  max-width: 100%;
  padding: 14px 18px;
  border: 1px solid var(--td-component-border);
  border-radius: var(--app-radius-lg);
  color: var(--td-text-color-primary);
  background: var(--td-bg-color-container);
  overflow-wrap: anywhere;
  text-decoration: none;
}
.protected-resource-card::before { content: '↗'; margin-right: 12px; }
.protected-resource-card:hover { border-color: var(--td-brand-color); }
</style>
