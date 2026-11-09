<template>
  <Transition name="intro-fade">
    <div v-if="visible" ref="stageRef" class="brand-intro" role="img"
      :aria-label="t('brandIntro.ariaLabel')" @click="skip">
      <div class="brand-intro__stage">
        <svg class="brand-intro__mark" viewBox="133 136 1806 381" aria-hidden="true" v-html="markSvg"></svg>
        <div class="brand-intro__cite" ref="citeRef"></div>
        <button class="brand-intro__skip" type="button" @click.stop="skip">
          {{ t('brandIntro.skip') }}
        </button>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
/**
 * Askademic 开场动画「破土问学」。
 * 字母如苗齐升 → 双叶与 i 芽展开 → 引证线划过 → 上浮交还页面。
 * 每会话一次(sessionStorage)；点击任意处跳过；prefers-reduced-motion 呈现完整静态字标。
 * 概念与规格见 docs/ASKADEMIC_INTRO_RESEARCH.md。
 */
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import gsap from 'gsap'
import { BRAND } from '@/config/brand'
import darkMarkRaw from '@/assets/logo/askademic-wordmark-dark.svg?raw'

const { t } = useI18n()
const visible = ref(BRAND.introEnabled && !sessionStorage.getItem('askademic_intro_played'))
const stageRef = ref<HTMLElement | null>(null)
const citeRef = ref<HTMLElement | null>(null)
// 字标资产已自带 data-unit(字母组)与 data-leaf(叶组)钩子;mask id 全局唯一无需改名
const markSvg = darkMarkRaw
  .replace(/^[\s\S]*?<svg[^>]*>/, '')
  .replace(/<\/svg>\s*$/, '')

let done = false

function finish() {
  if (done) return
  done = true
  sessionStorage.setItem('askademic_intro_played', '1')
  visible.value = false
}

function reducedMotion() {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

onMounted(() => {
  if (!visible.value) return
  const stage = stageRef.value
  if (!stage) return
  const units = stage.querySelectorAll<HTMLElement>('g[data-unit]')
  const leaves = Array.from(stage.querySelectorAll<SVGPathElement>('g[data-leaf] > path'))
    .sort((a, b) => {
      const ax = parseFloat(a.getAttribute('transform')?.match(/translate\(([\d.]+)/)?.[1] ?? '0')
      const bx = parseFloat(b.getAttribute('transform')?.match(/translate\(([\d.]+)/)?.[1] ?? '0')
      return ax - bx
    })
  const cite = citeRef.value
  // S = 1.3 → 总长 3.0s；跳过时 0.35 倍速体面播完
  var S = 1.3

  if (reducedMotion()) {
    gsap.set(units, { yPercent: 0, opacity: 1 })
    gsap.set(leaves, { scale: 1, opacity: 1 })
    gsap.set(cite, { scaleX: 1, opacity: 0.9 })
    setTimeout(finish, 1400)
    return
  }

  gsap.set(units, { yPercent: 118, opacity: 0 })
  gsap.set(leaves, { scale: 0, opacity: 0, transformOrigin: '50% 100%' })
  gsap.set(cite, { scaleX: 0, opacity: 0.9 })

  var tl = gsap.timeline({ defaults: { ease: 'power4.out' }, onComplete: finish })
  tl.fromTo(leaves[1],
    { y: -260, opacity: 0, scale: 0.6 },
    { y: 0, opacity: 1, scale: 1, duration: 0.5 * S, ease: 'power3.out' }, 0.05 * S)
  tl.to(units, { yPercent: 0, opacity: 1, duration: 0.9 * S, stagger: 0.036 * S }, 0.3 * S)
  tl.to(leaves[0], { scale: 1, opacity: 1, duration: 0.55 * S, ease: 'back.out(1.4)' }, 0.95 * S)
  tl.to(leaves[2], { scale: 1, opacity: 1, duration: 0.55 * S, ease: 'back.out(1.4)' }, 1.05 * S)
  tl.to(leaves[3], { scale: 1, opacity: 1, duration: 0.55 * S, ease: 'back.out(1.4)' }, 1.15 * S)
  tl.to(cite, { scaleX: 1, duration: 0.6 * S, ease: 'power2.inOut' }, 1.25 * S)
  tl.to(stage.querySelector('.brand-intro__mark'), { y: -10, duration: 0.4 * S, ease: 'power2.inOut' }, 1.9 * S)
  tl.to(cite, { opacity: 0.55, duration: 0.4 * S }, 1.9 * S)
})

function skip() {
  if (done) return
  if (reducedMotion()) { finish(); return }
  // 体面跳过：压到 0.35 倍速播完而不是瞬跳
  visible.value && gsap.globalTimeline.timeScale(0.35)
  // 简化：直接完成并淡出
  done = true
  sessionStorage.setItem('askademic_intro_played', '1')
  gsap.to(stageRef.value, { opacity: 0, duration: 0.35, onComplete: () => { visible.value = false } })
}
</script>

<style scoped>
.brand-intro {
  position: fixed; inset: 0; z-index: 3000;
  background: #0d2b28;
  display: grid; place-items: center;
  cursor: pointer;
}
.brand-intro__stage {
  position: relative; width: 100%; height: 100%;
  display: grid; place-items: center;
}
.brand-intro__mark { width: min(760px, 74vw); height: auto; display: block; }
.brand-intro__cite {
  position: absolute; left: 12%; bottom: 24%; height: 2px; width: 76%;
  background: #1b6e66; opacity: 0.9;
  transform: scaleX(0); transform-origin: left;
}
.brand-intro__skip {
  position: absolute; right: 18px; bottom: 16px;
  font-size: 12px; letter-spacing: 0.12em;
  color: rgba(232, 232, 231, 0.42);
  border: 0; background: none; cursor: pointer;
  font-family: inherit; padding: 6px 8px;
  transition: color 180ms ease;
}
.brand-intro__skip:hover { color: rgba(232, 232, 231, 0.85); }
.brand-intro__skip:focus-visible { outline: 1px solid rgba(232,232,231,0.5); outline-offset: 3px; }
.intro-fade-leave-active { transition: opacity 450ms cubic-bezier(0.55, 0.055, 0.675, 0.19); }
.intro-fade-leave-to { opacity: 0; }
@media (max-width: 640px) {
  .brand-intro__mark { width: 86vw; }
}
</style>
