<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import bgIsometricUrl from '../assets/bg-isometric.png'
import type {
  DrawResponse,
  HistoryItem,
  ProbabilitiesResponse,
  QuizSubmitResponse,
  VillageQuiz,
} from '../types/gacha'
import type { ActiveModalType } from '../composables/useVillage'

const props = defineProps<{
  activeModal: ActiveModalType
  selectedQuiz: VillageQuiz | null
  selectedOptionIndex: number | null
  submitting: boolean
  balance: number
  completedCount: number
  totalQuizzes: number
  quizzes: VillageQuiz[]
  rules: string[]
  probabilities: ProbabilitiesResponse | null
  history: HistoryItem[]
  lastQuizResult: QuizSubmitResponse | null
  lastDrawResult: DrawResponse | null
  errorState: { code: string; message: string; retryable: boolean } | null
  hasPendingQuizKey: boolean
  bridgeNotice: string | null
  bridgeRttMs: number | null
}>()

const completedQuizzes = computed(() => props.quizzes.filter((q) => q.completed))

const termsAndConditions = [
  'Pilih kuis yang akan Anda kerjakan.',
  'Pilih jawaban berdasarkan petunjuk yang ada di setiap pertanyaan dalam waktu yang telah ditentukan.',
  'Selesaikan kuis dan lihat skor yang Anda dapatkan.',
  'Dapatkan hadiah menarik bagi Anda yang berhasil menjawab di atas skor minimal kuis.',
  'Bank Mandiri dapat menarik kembali kuis sesuai dengan kebijakan Bank.',
]

function formatWIB(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  const formatted = new Intl.DateTimeFormat('id-ID', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    timeZone: 'Asia/Jakarta',
    hour12: false,
  }).format(date)
  return `${formatted.replace('.', ':')} WIB`
}

const emit = defineEmits<{
  close: []
  openRewards: []
  selectOption: [index: number]
  submitQuiz: []
  drawGift: []
  probeIllegal: []
}>()

const dialogRef = ref<HTMLDialogElement | null>(null)

watch(
  () => props.activeModal,
  (val) => {
    const dlg = dialogRef.value
    if (!dlg) return
    if (val) {
      if (!dlg.open) dlg.showModal()
    } else if (dlg.open) {
      dlg.close()
    }
  },
)
</script>

<template>
  <dialog ref="dialogRef" class="village-dialog" @cancel.prevent="emit('close')">
    <div v-if="activeModal === 'quiz' && selectedQuiz" class="dialog-body">
      <div class="dialog-header">
        <div class="header-meta">
          <h2 class="dialog-title">{{ selectedQuiz.building_name }}</h2>
          <div class="meta-row">
            <span class="subtitle-text">{{ selectedQuiz.title }}</span>
            <span class="points-pill">+{{ selectedQuiz.reward_points }} Poin</span>
          </div>
        </div>
        <button type="button" class="close-x" aria-label="Tutup" @click="emit('close')">
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
            <path d="M4 4L12 12M12 4L4 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
          </svg>
        </button>
      </div>

      <p class="quiz-question">{{ selectedQuiz.question }}</p>

      <template v-if="selectedQuiz.completed">
        <div class="status-box status-success" role="status">
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none" aria-hidden="true">
            <circle cx="10" cy="10" r="9" stroke="currentColor" stroke-width="1.8" />
            <path d="M6.5 10.5L8.8 12.8L13.5 7.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          <div>
            <strong>Kuis selesai</strong>
            <p>Hadiah +{{ selectedQuiz.reward_points }} poin sudah masuk ke saldo Anda.</p>
          </div>
        </div>
        <button type="button" class="btn btn-secondary" @click="emit('openRewards')">
          Lihat Saldo &amp; Kotak Hadiah
        </button>
      </template>

      <div v-else class="option-list">
        <button
          v-for="(opt, idx) in selectedQuiz.options"
          :key="idx"
          type="button"
          class="option-btn"
          :class="{ selected: selectedOptionIndex === idx }"
          @click="emit('selectOption', idx)"
        >
          <span class="option-radio" aria-hidden="true">
            <span v-if="selectedOptionIndex === idx" class="option-radio-dot"></span>
          </span>
          <span>{{ opt }}</span>
        </button>
      </div>

      <div v-if="errorState" class="status-box status-error" role="alert">
        <div>
          <strong>{{ errorState.code }}</strong>
          <p>{{ errorState.message }}</p>
        </div>
      </div>

      <div
        v-if="lastQuizResult"
        class="status-box"
        :class="lastQuizResult.correct ? 'status-success' : 'status-error'"
        role="status"
      >
        <div>
          <strong>{{ lastQuizResult.correct ? `Benar! +${lastQuizResult.reward_points} Poin` : 'Belum Tepat' }}</strong>
          <p>{{ lastQuizResult.explanation }}</p>
        </div>
      </div>

      <button
        v-if="!selectedQuiz.completed"
        type="button"
        class="btn btn-primary"
        :disabled="selectedOptionIndex === null || submitting"
        data-testid="submit-quiz-btn"
        @click="emit('submitQuiz')"
      >
        {{
          submitting
            ? 'Mengirim Jawaban...'
            : hasPendingQuizKey
              ? 'Coba Lagi (Idempotency Key Sama)'
              : 'Kirim Jawaban'
        }}
      </button>
    </div>

    <div v-else-if="activeModal === 'help'" class="subpage-shell">
      <div class="subpage-hero" :style="{ backgroundImage: `url(${bgIsometricUrl})`, backgroundPosition: 'center 42%' }">
        <div class="subpage-nav">
          <button type="button" class="subpage-back" @click="emit('close')">
            <span>‹</span>
            <span>Cara Main</span>
          </button>
          <button type="button" class="subpage-close-pill" aria-label="Tutup" @click="emit('close')">✕</button>
        </div>
        <div class="subpage-gold-line" />
        <div class="subpage-ribbon">Cara Main</div>
      </div>

      <div class="subpage-content">
        <div class="steps-blue-card">
          <div class="step-col">
            <div class="step-icon-circle">📋</div>
            <p class="step-label">Pilih satu tema kuis</p>
          </div>
          <div class="step-col">
            <div class="correct-pill">✓ Correct</div>
            <p class="step-label">Jawab semua pertanyaannya</p>
          </div>
          <div class="step-col">
            <div class="step-gift-box">🎁</div>
            <p class="step-label">Menangkan reward!</p>
          </div>
        </div>

        <h3 class="tnc-heading">Syarat dan Ketentuan</h3>
        <ol class="tnc-list">
          <li v-for="(item, idx) in termsAndConditions" :key="idx">{{ item }}</li>
        </ol>

        <button type="button" class="btn btn-secondary probe-btn" data-testid="probe-illegal-bridge" @click="emit('probeIllegal')">
          Uji Blokir Bridge Ilegal (F7)
        </button>
        <p v-if="bridgeNotice" class="mono">{{ bridgeNotice }}</p>
      </div>
    </div>

    <div v-else-if="activeModal === 'rewards'" class="subpage-shell">
      <div class="subpage-hero" :style="{ backgroundImage: `url(${bgIsometricUrl})`, backgroundPosition: 'center top' }">
        <div class="subpage-nav">
          <button type="button" class="subpage-back" @click="emit('close')">
            <span>‹</span>
            <span>Reward Anda</span>
          </button>
          <button type="button" class="subpage-close-pill" aria-label="Tutup" @click="emit('close')">✕</button>
        </div>
        <div class="hero-center-badge">📋</div>
        <div class="subpage-gold-line" />
        <div class="subpage-ribbon">Reward Anda</div>
      </div>

      <div class="subpage-content">
        <div v-if="errorState" class="status-box status-error" role="alert">
          <div>
            <strong>{{ errorState.code }}</strong>
            <p>{{ errorState.message }}</p>
          </div>
        </div>

        <div v-if="lastDrawResult" class="status-box status-success" role="status">
          <div>
            <strong>Dapat {{ lastDrawResult.reward.name }}</strong>
            <p>+{{ lastDrawResult.reward.reward_points }} Livin'poin ditambahkan ke saldo Anda.</p>
          </div>
        </div>

        <div v-if="completedQuizzes.length === 0 && history.length === 0" class="reward-blue-card reward-empty">
          <div class="livin-coin"><span>livin'</span><small>poin</small></div>
          <div>
            <div class="reward-title">Belum ada reward</div>
            <div class="reward-subtitle">Selesaikan kuis di Livin' Village untuk mendapatkan Livin'poin.</div>
          </div>
        </div>

        <template v-else>
          <div v-for="item in history" :key="item.draw_id" class="reward-blue-card">
            <div class="livin-coin"><span>livin'</span><small>poin</small></div>
            <div>
              <div class="reward-title">{{ item.reward_points }} Livin'poin</div>
              <div class="reward-subtitle">Didapat pada {{ formatWIB(item.created_at) }}</div>
            </div>
          </div>

          <div v-for="quiz in completedQuizzes" :key="quiz.id" class="reward-blue-card">
            <div class="livin-coin"><span>livin'</span><small>poin</small></div>
            <div>
              <div class="reward-title">{{ quiz.reward_points }} Livin'poin</div>
              <div class="reward-subtitle">{{ quiz.building_name }} · {{ quiz.title }}</div>
            </div>
          </div>
        </template>

        <div class="reward-footer-bar">
          <div>
            <div class="balance-caption">Total Saldo</div>
            <div class="balance-inline">{{ balance.toLocaleString('id-ID') }} Livin'poin</div>
          </div>
          <button
            type="button"
            class="btn btn-primary btn-compact"
            :disabled="submitting || balance < (probabilities?.draw_cost ?? 100)"
            @click="emit('drawGift')"
          >
            {{ submitting ? 'Membuka...' : `Kotak Hadiah (${probabilities?.draw_cost ?? 100} Poin)` }}
          </button>
        </div>
      </div>
    </div>
  </dialog>
</template>

<style scoped>
.subpage-shell {
  position: fixed;
  inset: 0;
  background: #ffffff;
  color: #1f242d;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  z-index: 50;
}

.subpage-hero {
  position: relative;
  height: 184px;
  flex-shrink: 0;
  background-size: cover;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  margin-bottom: 24px;
}

.subpage-hero::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(180deg, rgba(140, 204, 245, 0.85) 0%, rgba(140, 204, 245, 0) 55%);
  pointer-events: none;
}

.subpage-nav {
  position: relative;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px 0;
}

.subpage-back {
  background: none;
  border: 0;
  color: #ffffff;
  font-size: 18px;
  font-weight: 700;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  text-shadow: 0 1px 3px rgba(0, 0, 0, 0.22);
}

.subpage-close-pill {
  border: 1px solid rgba(255, 255, 255, 0.45);
  background: rgba(0, 0, 0, 0.22);
  color: #ffffff;
  width: 34px;
  height: 30px;
  border-radius: 999px;
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
}

.hero-center-badge {
  position: relative;
  z-index: 2;
  align-self: center;
  width: 72px;
  height: 72px;
  border-radius: 50%;
  border: 3px solid #ffffff;
  background: linear-gradient(135deg, #8fd3ff, #2987f0);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
  margin-bottom: 14px;
}

.subpage-gold-line {
  height: 5px;
  background: #ffc229;
  width: 100%;
}

.subpage-ribbon {
  position: absolute;
  left: 50%;
  bottom: -18px;
  transform: translateX(-50%);
  background: linear-gradient(180deg, #ffb833 0%, #f59412 100%);
  border: 1.5px solid #ffd959;
  color: #ffffff;
  font-size: 16px;
  font-weight: 700;
  padding: 9px 40px;
  border-radius: 12px;
  white-space: nowrap;
  box-shadow: 0 3px 8px rgba(0, 0, 0, 0.16);
}

.subpage-content {
  padding: 12px 18px 32px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.steps-blue-card {
  background: #005ae2;
  border-radius: 14px;
  padding: 20px 12px;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  text-align: center;
  color: #ffffff;
}

.step-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
}

.step-icon-circle {
  width: 52px;
  height: 52px;
  border-radius: 50%;
  background: linear-gradient(135deg, #8fd3ff, #2987f0);
  border: 2.5px solid #ffffff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
}

.correct-pill {
  margin-top: 10px;
  margin-bottom: 10px;
  background: linear-gradient(180deg, #7ddf40 0%, #57ba24 100%);
  border: 1.5px solid rgba(255, 255, 255, 0.7);
  border-radius: 999px;
  padding: 6px 12px;
  font-size: 12px;
  font-weight: 700;
}

.step-gift-box {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  border: 2px solid #ffd640;
  background: linear-gradient(180deg, #42abff 0%, #146ee6 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
}

.step-label {
  margin: 0;
  font-size: 12px;
  font-weight: 600;
  line-height: 1.3;
}

.tnc-heading {
  margin: 4px 0 0;
  text-align: center;
  font-size: 16px;
  font-weight: 700;
  color: #21252d;
}

.tnc-list {
  margin: 0;
  padding-left: 20px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 14px;
  line-height: 1.45;
  color: #333842;
}

.reward-blue-card {
  background: #005ae2;
  border-radius: 12px;
  padding: 16px;
  color: #ffffff;
  display: flex;
  align-items: center;
  gap: 14px;
}

.livin-coin {
  width: 44px;
  height: 44px;
  flex-shrink: 0;
  border-radius: 50%;
  border: 2px solid #fff2a6;
  background: linear-gradient(135deg, #ffe04d 0%, #f2a314 100%);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  line-height: 1;
  font-weight: 800;
}

.livin-coin span {
  font-size: 10px;
}

.livin-coin small {
  font-size: 8px;
}

.reward-title {
  font-size: 16px;
  font-weight: 700;
}

.reward-subtitle {
  margin-top: 4px;
  font-size: 12px;
  opacity: 0.9;
}

.reward-footer-bar {
  margin-top: 4px;
  padding: 12px 14px;
  background: #f2f4f7;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.balance-inline {
  font-size: 15px;
  font-weight: 700;
}

.btn-compact {
  width: auto;
  padding: 10px 14px;
  font-size: 13px;
}

.probe-btn {
  margin-top: 8px;
}
</style>
