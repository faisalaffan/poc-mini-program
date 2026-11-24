<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import bgIsometricUrl from './assets/bg-isometric.png'
import VillageModals from './components/VillageModals.vue'
import VillagePinIcon from './components/VillagePinIcon.vue'
import { useVillage } from './composables/useVillage'

const IMAGE_RATIO = 852 / 1846
const frameRef = ref<HTMLDivElement | null>(null)

const {
  village,
  quizzes,
  balance,
  campaignBanner,
  probabilities,
  history,
  activeModal,
  selectedQuiz,
  selectedOptionIndex,
  lastQuizResult,
  lastDrawResult,
  pendingQuizIdempotencyKey,
  errorState,
  bridgeStatus,
  submitting,
  loadAll,
  openQuiz,
  openHelp,
  openRewards,
  closeModal,
  submitSelectedQuiz,
  drawGiftBox,
  testUnauthorizedBridgeCall,
} = useVillage()

function updateMapFrameGeometry() {
  const el = frameRef.value
  if (!el || typeof window === 'undefined') return
  const vw = window.innerWidth
  const vh = window.innerHeight
  const screenRatio = vw / Math.max(1, vh)

  let w = vw
  let h = vh
  if (screenRatio > IMAGE_RATIO) {
    w = vw
    h = vw / IMAGE_RATIO
  } else {
    h = vh
    w = vh * IMAGE_RATIO
  }

  el.style.width = `${w}px`
  el.style.height = `${h}px`
  el.style.left = `${(vw - w) / 2}px`
  el.style.top = `${(vh - h) / 2}px`
}

onMounted(() => {
  updateMapFrameGeometry()
  window.addEventListener('resize', updateMapFrameGeometry)
  void loadAll()
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', updateMapFrameGeometry)
})
</script>

<template>
  <main class="village-viewport">
    <div ref="frameRef" class="map-frame">
      <img :src="bgIsometricUrl" alt="Peta Livin' Village" class="map-bg" />

      <button
        v-for="quiz in quizzes"
        :key="quiz.id"
        type="button"
        class="map-pin"
        :class="[`pin-${quiz.pin_color}`, `pin-pos-${quiz.id}`]"
        :aria-label="`${quiz.building_name}: ${quiz.title}`"
        @click="openQuiz(quiz)"
      >
        <VillagePinIcon :completed="quiz.completed" />
      </button>
    </div>

    <header class="hud-top">
      <div class="campaign-banner">
        {{ campaignBanner }}
      </div>
      <div class="hud-actions">
        <button type="button" class="fab-square fab-help" aria-label="Panduan kuis" @click="openHelp">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden="true">
            <path
              d="M9.5 9a2.7 2.7 0 0 1 5.2.9c0 1.8-2.7 2.3-2.7 4.1"
              stroke="currentColor"
              stroke-width="2.6"
              stroke-linecap="round"
            />
            <circle cx="12" cy="18" r="1.4" fill="currentColor" />
          </svg>
        </button>
        <button type="button" class="fab-square fab-gift" aria-label="Saldo dan hadiah" @click="openRewards">
          <svg width="21" height="21" viewBox="0 0 24 24" fill="none" aria-hidden="true">
            <rect x="4" y="10" width="16" height="10" rx="2" fill="currentColor" fill-opacity="0.92" />
            <rect x="3" y="7" width="18" height="4" rx="1.5" fill="currentColor" />
            <rect x="10.8" y="7" width="2.4" height="13" fill="#1874e6" />
            <path d="M12 7c-2.5 0-4.2-1.4-3.7-3 .4-1.3 2.4-1.2 3.7 1.2 1.3-2.4 3.3-2.5 3.7-1.2.5 1.6-1.2 3-3.7 3Z" fill="#ffd644" />
          </svg>
          <span class="fab-badge">{{ village?.completed_count ?? 0 }}/{{ village?.total_quizzes ?? 3 }}</span>
        </button>
      </div>
    </header>

    <div v-if="errorState && !activeModal" class="hud-toast-error" role="alert">
      {{ errorState.message }}
    </div>

    <VillageModals
      :active-modal="activeModal"
      :selected-quiz="selectedQuiz"
      :selected-option-index="selectedOptionIndex"
      :submitting="submitting"
      :balance="balance"
      :completed-count="village?.completed_count ?? 0"
      :total-quizzes="village?.total_quizzes ?? 3"
      :quizzes="village?.quizzes ?? []"
      :rules="village?.rules ?? []"
      :probabilities="probabilities"
      :history="history"
      :last-quiz-result="lastQuizResult"
      :last-draw-result="lastDrawResult"
      :error-state="errorState"
      :has-pending-quiz-key="pendingQuizIdempotencyKey !== null"
      :bridge-notice="bridgeStatus.probeNotice"
      :bridge-rtt-ms="bridgeStatus.rttMs"
      @close="closeModal"
      @open-rewards="openRewards"
      @select-option="(idx) => (selectedOptionIndex = idx)"
      @submit-quiz="submitSelectedQuiz"
      @draw-gift="drawGiftBox"
      @probe-illegal="testUnauthorizedBridgeCall"
    />
  </main>
</template>
