<script setup lang="ts">
/**
 * Askademic 品牌字标(用户设计稿矢量清理版)。
 * 亮/暗两份资产按 <html theme-mode> 切换(MutationObserver,无 CSS 竞争)。
 * favicon 仍是"问"磁贴(16px 下 9 字母不可读,字标系统的常规做法)。
 */
import { onBeforeUnmount, onMounted, ref } from 'vue'
import wordmarkLight from '@/assets/logo/askademic-wordmark.svg'
import wordmarkDark from '@/assets/logo/askademic-wordmark-dark.svg'

withDefaults(defineProps<{
  /** 渲染高度(px);宽度按 ~4.74:1 比例自动得出 */
  height?: number
  /** 无障碍名称;纯装饰场景传空串并让外层 aria-hidden */
  label?: string
}>(), { height: 22, label: 'Askademic' })

const isDark = ref(document.documentElement.getAttribute('theme-mode') === 'dark')
let observer: MutationObserver | null = null

onMounted(() => {
  observer = new MutationObserver(() => {
    isDark.value = document.documentElement.getAttribute('theme-mode') === 'dark'
  })
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ['theme-mode'] })
})
onBeforeUnmount(() => observer?.disconnect())
</script>

<template>
  <img
    class="brand-wordmark"
    :src="isDark ? wordmarkDark : wordmarkLight"
    :alt="label || ''"
    :style="{ height: `${height}px` }"
  >
</template>

<style scoped>
.brand-wordmark { display: block; width: auto; }
</style>
