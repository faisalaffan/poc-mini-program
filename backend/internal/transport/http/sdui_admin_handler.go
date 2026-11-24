package http

import (
	"net/http"
	"os"
	"path/filepath"
)

// NewSDUIBackgroundHandler menyajikan gambar peta isometric untuk preview visual di SDUI Configurator.
func NewSDUIBackgroundHandler(webDistDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		candidates := []string{
			"../bg-isometric.png",
			"bg-isometric.png",
			"/app/bg-isometric.png",
		}
		if webDistDir != "" {
			if matches, err := filepath.Glob(filepath.Join(webDistDir, "assets", "bg-isometric*.png")); err == nil {
				candidates = append(candidates, matches...)
			}
		}

		for _, path := range candidates {
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				w.Header().Set("Cache-Control", "public, max-age=3600")
				http.ServeFile(w, r, path)
				return
			}
		}
		http.NotFound(w, r)
	}
}

// NewSDUIAdminHandler menyajikan UI studio untuk mengonfigurasi layout layar SDUI di PostgreSQL.
func NewSDUIAdminHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sduiAdminHTML))
	}
}

const sduiAdminHTML = `<!doctype html>
<html lang="id">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>SDUI Studio · Livin' Village</title>
  <style>
    :root {
      --bg: #ffffff;
      --canvas: #f1f3f6;
      --panel: #fafafa;
      --border: #e4e6eb;
      --text: #111318;
      --muted: #667085;
      --accent: #005ae2;
      --accent-hover: #0049b8;
      --danger: #dc2626;
      --ok: #15803d;
      --mono: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    }
    * { box-sizing: border-box; }
    html, body { height: 100%; margin: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Inter", "Segoe UI", sans-serif;
      font-size: 13px;
      color: var(--text);
      background: var(--bg);
      display: flex;
      flex-direction: column;
      overflow: hidden;
    }
    @media (max-width: 1024px) {
      body { overflow: auto; height: auto; }
    }

    /* Top Bar */
    .topbar {
      height: 52px;
      flex-shrink: 0;
      border-bottom: 1px solid var(--border);
      background: var(--bg);
      padding: 0 16px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
    }
    .topbar-left, .topbar-right {
      display: flex;
      align-items: center;
      gap: 12px;
    }
    .title-group {
      display: flex;
      align-items: baseline;
      gap: 8px;
    }
    .title-group strong {
      font-size: 14px;
      font-weight: 600;
      letter-spacing: -0.01em;
    }
    .title-group span {
      font-size: 12px;
      color: var(--muted);
    }

    /* Segmented Variant Switcher */
    .seg {
      display: inline-flex;
      background: var(--canvas);
      border: 1px solid var(--border);
      border-radius: 7px;
      padding: 2px;
      gap: 2px;
    }
    .seg button {
      border: 0;
      background: transparent;
      color: var(--muted);
      font-size: 12px;
      font-weight: 500;
      padding: 4px 10px;
      border-radius: 5px;
      cursor: pointer;
    }
    .seg button.active {
      background: var(--bg);
      color: var(--text);
      box-shadow: 0 1px 2px rgba(0, 0, 0, 0.06);
    }

    .status-pill {
      font-size: 12px;
      color: var(--muted);
      max-width: 320px;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
    .status-pill.ok { color: var(--ok); }
    .status-pill.err { color: var(--danger); }

    .btn {
      font: inherit;
      font-size: 12px;
      font-weight: 500;
      border: 1px solid var(--border);
      background: var(--bg);
      color: var(--text);
      padding: 6px 12px;
      border-radius: 7px;
      cursor: pointer;
      text-decoration: none;
      display: inline-flex;
      align-items: center;
      gap: 6px;
    }
    .btn:hover { background: var(--panel); }
    .btn-primary {
      background: var(--accent);
      border-color: var(--accent);
      color: #fff;
    }
    .btn-primary:hover { background: var(--accent-hover); }
    .btn-ghost {
      border-color: transparent;
      background: transparent;
      color: var(--muted);
      padding: 6px 8px;
    }
    .btn-ghost:hover { color: var(--text); background: var(--canvas); }
    .btn-danger-ghost {
      border-color: transparent;
      background: transparent;
      color: var(--muted);
      padding: 4px 6px;
      font-size: 12px;
    }
    .btn-danger-ghost:hover { color: var(--danger); background: #fef2f2; }

    /* 3-Pane Studio Workspace */
    .workspace {
      flex: 1;
      min-height: 0;
      display: grid;
      grid-template-columns: 272px 1fr 360px;
    }
    @media (max-width: 1024px) {
      .workspace { grid-template-columns: 1fr; }
    }

    /* Left Rail: Layers */
    .layers-pane {
      border-right: 1px solid var(--border);
      background: var(--bg);
      display: flex;
      flex-direction: column;
      min-height: 0;
    }
    .pane-head {
      padding: 12px 14px;
      border-bottom: 1px solid var(--border);
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 8px;
    }
    .pane-head h2 {
      margin: 0;
      font-size: 12px;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.04em;
      color: var(--muted);
    }
    .layers-list {
      flex: 1;
      overflow-y: auto;
      padding: 8px;
      display: flex;
      flex-direction: column;
      gap: 2px;
    }
    .layer-row {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 8px;
      padding: 8px 10px;
      border-radius: 7px;
      cursor: pointer;
      border: 1px solid transparent;
    }
    .layer-row:hover { background: var(--panel); }
    .layer-row.active {
      background: #eef4ff;
      border-color: #c7dcff;
    }
    .layer-main {
      min-width: 0;
      display: flex;
      flex-direction: column;
      gap: 2px;
    }
    .layer-id {
      font-family: var(--mono);
      font-size: 12px;
      font-weight: 500;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
    .layer-type {
      font-size: 11px;
      color: var(--muted);
    }
    .pane-foot {
      padding: 10px 12px;
      border-top: 1px solid var(--border);
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 8px;
    }

    /* Center Canvas */
    .canvas-pane {
      background-color: var(--canvas);
      background-image: radial-gradient(#d5d9e2 1px, transparent 1px);
      background-size: 16px 16px;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 18px;
      min-height: 0;
      position: relative;
      overflow: hidden;
    }
    .device-wrap {
      height: min(640px, calc(100vh - 130px));
      aspect-ratio: 852 / 1846;
      position: relative;
      border-radius: 30px;
      border: 4px solid #181a1f;
      background: #7ec8f2 url('/sdui-admin/bg-isometric.png') center/cover no-repeat;
      box-shadow: 0 20px 48px rgba(16, 24, 40, 0.16);
      overflow: hidden;
      cursor: crosshair;
      user-select: none;
    }
    .preview-banner {
      position: absolute;
      top: 44px;
      left: 50%;
      transform: translateX(-50%);
      width: 68%;
      background: linear-gradient(180deg, #3da3ff, #1775e8);
      border: 1.5px solid #87ccff;
      color: #fff;
      font-size: 9.5px;
      font-weight: 600;
      line-height: 1.3;
      text-align: center;
      padding: 6px 8px;
      border-radius: 10px;
      pointer-events: none;
    }
    .preview-actions {
      position: absolute;
      top: 44px;
      right: 8px;
      display: flex;
      flex-direction: column;
      gap: 6px;
      pointer-events: none;
    }
    .preview-fab {
      width: 26px;
      height: 26px;
      border-radius: 7px;
      border: 1.5px solid #fff;
      color: #fff;
      display: flex;
      align-items: center;
      justify-content: center;
      box-shadow: 0 3px 8px rgba(0, 0, 0, 0.22);
    }
    .preview-fab.help { background: linear-gradient(180deg, #ffd645, #e6940a); }
    .preview-fab.gift { background: linear-gradient(180deg, #4fb3ff, #1773e6); }
    .preview-pin {
      position: absolute;
      transform: translate(-50%, -50%);
      width: 32px;
      height: 32px;
      border-radius: 50%;
      border: 2px solid #fff;
      color: #fff;
      display: flex;
      align-items: center;
      justify-content: center;
      box-shadow: 0 4px 10px rgba(0, 0, 0, 0.26);
      cursor: pointer;
    }
    .preview-pin.blue { background: #1775e8; }
    .preview-pin.orange { background: #f59e0b; }
    .preview-pin.red { background: #dc2626; }
    .preview-pin.selected {
      box-shadow: 0 0 0 3px #111318, 0 6px 14px rgba(0, 0, 0, 0.3);
    }
    .pin-tag {
      position: absolute;
      top: 34px;
      left: 50%;
      transform: translateX(-50%);
      background: rgba(17, 19, 24, 0.85);
      color: #fff;
      font-size: 9px;
      font-weight: 500;
      padding: 2px 6px;
      border-radius: 4px;
      white-space: nowrap;
      pointer-events: none;
    }
    .canvas-caption {
      margin-top: 10px;
      font-size: 11px;
      color: var(--muted);
      font-variant-numeric: tabular-nums;
    }

    /* Right Inspector */
    .inspector-pane {
      border-left: 1px solid var(--border);
      background: var(--bg);
      display: flex;
      flex-direction: column;
      min-height: 0;
    }
    .inspector-body {
      flex: 1;
      overflow-y: auto;
      padding: 14px 16px 24px;
    }
    .section {
      padding-bottom: 14px;
      margin-bottom: 14px;
      border-bottom: 1px solid var(--border);
    }
    .section:last-child { border-bottom: 0; margin-bottom: 0; }
    .section-title {
      font-size: 11px;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.04em;
      color: var(--muted);
      margin: 0 0 10px;
    }
    .row-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
    .field { display: flex; flex-direction: column; gap: 4px; margin-bottom: 10px; }
    .field:last-child { margin-bottom: 0; }
    .field label {
      font-size: 11px;
      font-weight: 500;
      color: var(--muted);
    }
    input, select, textarea {
      font: inherit;
      font-size: 12.5px;
      border: 1px solid var(--border);
      border-radius: 6px;
      padding: 7px 9px;
      background: var(--bg);
      color: var(--text);
      outline: none;
    }
    input:focus, select:focus, textarea:focus {
      border-color: var(--accent);
    }
    input[readonly] {
      background: var(--panel);
      color: var(--muted);
    }
    .swatches {
      display: flex;
      gap: 8px;
      padding-top: 2px;
    }
    .swatch {
      width: 24px;
      height: 24px;
      border-radius: 50%;
      border: 2px solid transparent;
      cursor: pointer;
    }
    .swatch.active { box-shadow: 0 0 0 2px #fff, 0 0 0 4px var(--text); }
    .swatch.blue { background: #1775e8; }
    .swatch.orange { background: #f59e0b; }
    .swatch.red { background: #dc2626; }

    .opt-list { display: flex; flex-direction: column; gap: 6px; }
    .opt-row {
      display: flex;
      align-items: center;
      gap: 6px;
    }
    .opt-row input[type="text"] { flex: 1; min-width: 0; }
    .opt-flag {
      display: inline-flex;
      align-items: center;
      gap: 4px;
      font-size: 11px;
      font-weight: 600;
      padding: 5px 8px;
      border-radius: 6px;
      border: 1px solid var(--border);
      background: var(--bg);
      color: var(--muted);
      cursor: pointer;
      white-space: nowrap;
    }
    .opt-flag.correct {
      border-color: #16a34a;
      background: #dcfce7;
      color: #15803d;
    }

    textarea.json-editor {
      width: 100%;
      height: calc(100vh - 190px);
      min-height: 320px;
      font-family: var(--mono);
      font-size: 11.5px;
      line-height: 1.5;
      resize: vertical;
    }
  </style>
</head>
<body>
  <header class="topbar">
    <div class="topbar-left">
      <div class="title-group">
        <strong>Livin' Village</strong>
        <span>SDUI Configurator</span>
      </div>
      <div id="variantSeg" class="seg">
        <button type="button" data-variant="default" class="active">default</button>
        <button type="button" data-variant="promo">promo</button>
        <button type="button" data-variant="compact">compact</button>
      </div>
    </div>

    <div class="topbar-right">
      <span id="statusPill" class="status-pill">Memuat...</span>
      <button id="resetQuizBtn" type="button" class="btn">Reset Kuis User</button>
      <button id="loadBtn" type="button" class="btn">Muat Ulang</button>
      <a href="/swagger/" class="btn btn-ghost">Swagger</a>
      <a href="/logout" class="btn btn-ghost">Keluar</a>
      <button id="saveBtn" type="button" class="btn btn-primary">Simpan ke DB</button>
    </div>
  </header>

  <main class="workspace">
    <!-- Left: Layers -->
    <aside class="layers-pane">
      <div class="pane-head">
        <h2>Komponen (<span id="compCount">0</span>)</h2>
      </div>
      <div id="compList" class="layers-list"></div>
      <div class="pane-foot">
        <button id="addPinBtn" type="button" class="btn">+ Pin Kuis</button>
        <button id="addActionBtn" type="button" class="btn">+ Tombol</button>
      </div>
    </aside>

    <!-- Center: Live Map Canvas -->
    <section class="canvas-pane">
      <div id="phoneFrame" class="device-wrap"></div>
      <div id="canvasCaption" class="canvas-caption">Klik peta untuk memposisikan pin terpilih</div>
    </section>

    <!-- Right: Inspector / JSON -->
    <aside class="inspector-pane">
      <div class="pane-head">
        <h2>Inspector</h2>
        <div class="seg">
          <button type="button" id="tabForm" class="active">Properti</button>
          <button type="button" id="tabJson">JSON</button>
        </div>
      </div>

      <div id="formView" class="inspector-body">
        <div class="section">
          <h3 class="section-title">Layar</h3>
          <div class="row-2">
            <div class="field">
              <label>Screen ID</label>
              <input id="screenIdInput" readonly />
            </div>
            <div class="field">
              <label>Schema Version</label>
              <input id="schemaVersionInput" type="number" min="1" />
            </div>
          </div>
          <div class="field">
            <label>Navigation Title</label>
            <input id="navTitleInput" />
          </div>
          <div class="field">
            <label>Background Image URL / Asset</label>
            <input id="bgAssetInput" placeholder="/assets/bg-isometric.png" />
          </div>
        </div>

        <div id="editorPane" class="section" style="display:none;">
          <h3 class="section-title">Komponen Terpilih</h3>
          <div class="field">
            <label>ID Komponen</label>
            <input id="editId" />
          </div>
          <div class="row-2">
            <div class="field">
              <label>Tipe</label>
              <select id="editType">
                <option value="floating_banner">floating_banner</option>
                <option value="floating_action">floating_action</option>
                <option value="map_pin">map_pin</option>
              </select>
            </div>
            <div class="field">
              <label>Action</label>
              <select id="editAction">
                <option value="">(none)</option>
                <option value="open_quiz">open_quiz</option>
                <option value="open_help">open_help</option>
                <option value="open_rewards">open_rewards</option>
                <option value="draw">draw</option>
                <option value="refresh">refresh</option>
              </select>
            </div>
          </div>

          <div id="propsBanner" class="field" style="display:none;">
            <label>Teks Banner</label>
            <input id="editBannerText" />
          </div>

          <div id="propsAction" class="row-2" style="display:none;">
            <div class="field">
              <label>Icon</label>
              <select id="editActionIcon">
                <option value="help">help (?)</option>
                <option value="gift">gift (Hadiah)</option>
              </select>
            </div>
            <div class="field">
              <label>Badge Key</label>
              <input id="editBadgeKey" placeholder="quiz_progress" />
            </div>
          </div>

          <div id="propsPin" style="display:none;">
            <div class="field">
              <label>Judul Pin</label>
              <input id="editPinTitle" />
            </div>
            <div class="field">
              <label>Warna Pin</label>
              <div class="swatches" id="colorSwatches">
                <button type="button" class="swatch blue" data-color="blue" aria-label="Biru"></button>
                <button type="button" class="swatch orange" data-color="orange" aria-label="Oranye"></button>
                <button type="button" class="swatch red" data-color="red" aria-label="Merah"></button>
              </div>
            </div>
            <div class="row-2" style="margin-top:10px;">
              <div class="field">
                <label>x_ratio</label>
                <input id="editXRatio" type="number" step="0.01" min="0" max="1" />
              </div>
              <div class="field">
                <label>y_ratio</label>
                <input id="editYRatio" type="number" step="0.01" min="0" max="1" />
              </div>
            </div>
            <div class="row-2" style="margin-top:10px;">
              <div class="field">
                <label>Nama Gedung</label>
                <input id="editBuildingName" />
              </div>
              <div class="field">
                <label>Hadiah Poin</label>
                <input id="editRewardPoints" type="number" min="0" />
              </div>
            </div>
            <div class="field" style="margin-top:10px;">
              <label>Pertanyaan Kuis</label>
              <textarea id="editQuestion" rows="2"></textarea>
            </div>
            <div class="field" style="margin-top:10px;">
              <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:4px;">
                <label style="margin:0;">Opsi Jawaban &amp; Kunci Benar</label>
                <button type="button" id="addOptionBtn" class="btn" style="padding:3px 8px;font-size:11px;">+ Opsi</button>
              </div>
              <div id="optionsList" class="opt-list"></div>
            </div>
          </div>
        </div>
      </div>

      <div id="jsonView" class="inspector-body" style="display:none;">
        <div class="field">
          <textarea id="jsonBox" class="json-editor"></textarea>
        </div>
        <button id="applyJsonBtn" type="button" class="btn" style="width:100%;justify-content:center;">Terapkan JSON ke Preview</button>
      </div>
    </aside>
  </main>

  <script>
    const ICON_CLIPBOARD = '<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><rect x="8" y="2" width="8" height="4" rx="1"/><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/><path d="M9 12h6M9 16h4"/></svg>';
    const ICON_GIFT = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="8" width="18" height="4" rx="1"/><path d="M12 8v13M19 12v7a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2v-7"/><path d="M7.5 8a2.5 2.5 0 0 1 0-5A4.8 8 0 0 1 12 8a4.8 8 0 0 1 4.5-5 2.5 2.5 0 0 1 0 5"/></svg>';
    const ICON_HELP = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/><circle cx="12" cy="17" r="0.8" fill="currentColor"/></svg>';

    let token = '';
    let currentVariant = 'default';
    let screen = null;
    let quizzes = [];
    let selectedIndex = -1;

    const el = (id) => document.getElementById(id);

    function setStatus(msg, kind = '') {
      const pill = el('statusPill');
      pill.textContent = msg;
      pill.className = 'status-pill' + (kind ? ' ' + kind : '');
    }

    async function ensureAuth() {
      if (token) return token;
      let deviceId = localStorage.getItem('sdui_admin_device_id');
      if (!deviceId) {
        deviceId = 'sdui-admin-' + Math.random().toString(36).slice(2, 10);
        localStorage.setItem('sdui_admin_device_id', deviceId);
      }
      const res = await fetch('/api/v1/auth/session', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ device_id: deviceId })
      });
      if (!res.ok) throw new Error('Gagal membuat sesi auth (' + res.status + ')');
      const data = await res.json();
      token = data.token || data.access_token || '';
      if (!token) throw new Error('Token sesi tidak ditemukan');
      return token;
    }

    async function apiFetch(path, opts = {}) {
      await ensureAuth();
      const send = () => fetch(path, {
        ...opts,
        headers: {
          'Authorization': 'Bearer ' + token,
          'Content-Type': 'application/json',
          ...(opts.headers || {})
        }
      });
      let res = await send();
      if (res.status === 401) {
        token = '';
        await ensureAuth();
        res = await send();
      }
      const data = await res.json();
      if (!res.ok) throw new Error(data?.error?.message || ('HTTP ' + res.status));
      return data;
    }

    async function loadScreen() {
      setStatus('Memuat...');
      try {
        const [screenData, villageData] = await Promise.all([
          apiFetch('/api/v1/sdui/village-screen?variant=' + encodeURIComponent(currentVariant) + '&include_answers=true'),
          apiFetch('/api/v1/village/state')
        ]);
        screen = screenData;
        quizzes = villageData.quizzes || [];
        selectedIndex = screen.components.findIndex((c) => c.type === 'map_pin');
        if (selectedIndex < 0 && screen.components.length > 0) selectedIndex = 0;
        renderAll();
        setStatus('Dimuat · ' + screen.variant + ' (' + screen.components.length + ')', 'ok');
      } catch (err) {
        setStatus(err.message, 'err');
      }
    }

    async function saveScreen() {
      if (!screen) return;
      setStatus('Menyimpan...');
      try {
        const payload = {
          schema_version: Number(screen.schema_version) || 1,
          screen_id: screen.screen_id || 'livin_village_home',
          variant: currentVariant,
          navigation_title: screen.navigation_title || "Livin' Village",
          background_asset: screen.background_asset || 'bg-isometric',
          supported_actions: screen.supported_actions || ['open_quiz', 'open_help', 'open_rewards', 'draw', 'refresh'],
          components: screen.components
        };
        const updated = await apiFetch('/api/v1/sdui/village-screen', {
          method: 'PUT',
          body: JSON.stringify(payload)
        });
        screen = updated;
        renderAll();
        setStatus('Tersimpan ke PostgreSQL (' + updated.variant + ')', 'ok');
      } catch (err) {
        setStatus(err.message, 'err');
      }
    }

    function renderAll() {
      if (!screen) return;
      el('screenIdInput').value = screen.screen_id || '';
      el('navTitleInput').value = screen.navigation_title || '';
      el('bgAssetInput').value = screen.background_asset || '/assets/bg-isometric.png';
      el('schemaVersionInput').value = screen.schema_version || 1;
      el('compCount').textContent = screen.components.length;
      el('jsonBox').value = JSON.stringify(screen, null, 2);
      renderPreview();
      renderList();
      renderEditor();
    }

    function renderPreview() {
      const frame = el('phoneFrame');
      const bg = screen.background_asset || '/assets/bg-isometric.png';
      const bgUrl = (bg.startsWith('http://') || bg.startsWith('https://') || bg.startsWith('/'))
        ? bg
        : '/assets/bg-isometric.png';
      frame.style.backgroundImage = 'url("' + bgUrl.replace(/"/g, '%22') + '")';
      frame.innerHTML = '';

      const banner = screen.components.find((c) => c.type === 'floating_banner');
      if (banner) {
        const div = document.createElement('div');
        div.className = 'preview-banner';
        div.textContent = banner.props?.text || '';
        frame.appendChild(div);
      }

      const actionsWrap = document.createElement('div');
      actionsWrap.className = 'preview-actions';
      screen.components.filter((c) => c.type === 'floating_action').forEach((act) => {
        const btn = document.createElement('div');
        const isGift = act.props?.icon === 'gift';
        btn.className = 'preview-fab ' + (isGift ? 'gift' : 'help');
        btn.innerHTML = isGift ? ICON_GIFT : ICON_HELP;
        actionsWrap.appendChild(btn);
      });
      frame.appendChild(actionsWrap);

      screen.components.forEach((comp, idx) => {
        if (comp.type !== 'map_pin') return;
        const pin = document.createElement('div');
        const color = comp.props?.pin_color || 'blue';
        pin.className = 'preview-pin ' + color + (idx === selectedIndex ? ' selected' : '');
        pin.style.left = ((comp.props?.x_ratio ?? 0.5) * 100) + '%';
        pin.style.top = ((comp.props?.y_ratio ?? 0.5) * 100) + '%';
        pin.innerHTML = ICON_CLIPBOARD;
        const lbl = document.createElement('span');
        lbl.className = 'pin-tag';
        lbl.textContent = comp.props?.title || comp.id;
        pin.appendChild(lbl);
        pin.addEventListener('click', (e) => {
          e.stopPropagation();
          selectedIndex = idx;
          renderAll();
        });
        frame.appendChild(pin);
      });

      const active = screen.components[selectedIndex];
      if (active && active.type === 'map_pin') {
        el('canvasCaption').textContent = (active.props?.title || active.id) + ' · x=' + (active.props?.x_ratio ?? 0.5) + ' y=' + (active.props?.y_ratio ?? 0.5) + ' (klik peta untuk pindah)';
      } else {
        el('canvasCaption').textContent = 'Pilih map_pin di kiri lalu klik peta untuk atur posisi';
      }
    }

    function renderList() {
      const list = el('compList');
      list.innerHTML = '';
      screen.components.forEach((comp, idx) => {
        const row = document.createElement('div');
        row.className = 'layer-row' + (idx === selectedIndex ? ' active' : '');
        const main = document.createElement('div');
        main.className = 'layer-main';
        const idSpan = document.createElement('span');
        idSpan.className = 'layer-id';
        idSpan.textContent = comp.id;
        const typeSpan = document.createElement('span');
        typeSpan.className = 'layer-type';
        typeSpan.textContent = comp.type + (comp.action ? ' · ' + comp.action : '');
        main.appendChild(idSpan);
        main.appendChild(typeSpan);

        const delBtn = document.createElement('button');
        delBtn.type = 'button';
        delBtn.className = 'btn-danger-ghost';
        delBtn.textContent = '×';
        delBtn.title = 'Hapus komponen';
        delBtn.onclick = (e) => {
          e.stopPropagation();
          screen.components.splice(idx, 1);
          if (selectedIndex >= screen.components.length) selectedIndex = screen.components.length - 1;
          renderAll();
        };

        row.appendChild(main);
        row.appendChild(delBtn);
        row.onclick = () => { selectedIndex = idx; renderAll(); };
        list.appendChild(row);
      });
    }

    function renderEditor() {
      const pane = el('editorPane');
      if (selectedIndex < 0 || selectedIndex >= screen.components.length) {
        pane.style.display = 'none';
        return;
      }
      pane.style.display = 'block';
      const comp = screen.components[selectedIndex];
      comp.props = comp.props || {};

      el('editId').value = comp.id || '';
      el('editType').value = comp.type || 'map_pin';
      el('editAction').value = comp.action || '';

      el('propsBanner').style.display = comp.type === 'floating_banner' ? 'flex' : 'none';
      el('propsAction').style.display = comp.type === 'floating_action' ? 'grid' : 'none';
      el('propsPin').style.display = comp.type === 'map_pin' ? 'block' : 'none';

      if (comp.type === 'floating_banner') {
        el('editBannerText').value = comp.props.text || '';
      } else if (comp.type === 'floating_action') {
        el('editActionIcon').value = comp.props.icon || 'help';
        el('editBadgeKey').value = comp.props.badge_key || '';
      } else if (comp.type === 'map_pin') {
        if (!comp.props.quiz_id) comp.props.quiz_id = comp.id;
        el('editPinTitle').value = comp.props.title || '';
        const pinColor = comp.props.pin_color || 'blue';
        document.querySelectorAll('#colorSwatches .swatch').forEach((btn) => {
          btn.classList.toggle('active', btn.dataset.color === pinColor);
        });
        el('editXRatio').value = comp.props.x_ratio ?? 0.5;
        el('editYRatio').value = comp.props.y_ratio ?? 0.5;
        el('editBuildingName').value = comp.props.building_name || '';
        el('editRewardPoints').value = comp.props.reward_points ?? 150;
        el('editQuestion').value = comp.props.question || '';
        renderOptionsList(comp);
      }
    }

    function renderOptionsList(comp) {
      const wrap = el('optionsList');
      wrap.innerHTML = '';
      if (!Array.isArray(comp.props.options)) comp.props.options = [];
      const correctIdx = Number(comp.props.correct_index ?? 0);

      comp.props.options.forEach((optText, idx) => {
        const row = document.createElement('div');
        row.className = 'opt-row';

        const flagBtn = document.createElement('button');
        flagBtn.type = 'button';
        flagBtn.className = 'opt-flag' + (idx === correctIdx ? ' correct' : '');
        flagBtn.textContent = idx === correctIdx ? '✓ Benar' : 'Pilih';
        flagBtn.onclick = () => {
          comp.props.correct_index = idx;
          renderOptionsList(comp);
          el('jsonBox').value = JSON.stringify(screen, null, 2);
        };

        const input = document.createElement('input');
        input.type = 'text';
        input.value = optText;
        input.oninput = () => {
          comp.props.options[idx] = input.value;
          el('jsonBox').value = JSON.stringify(screen, null, 2);
        };

        const delBtn = document.createElement('button');
        delBtn.type = 'button';
        delBtn.className = 'btn-danger-ghost';
        delBtn.textContent = '×';
        delBtn.onclick = () => {
          comp.props.options.splice(idx, 1);
          if (comp.props.correct_index >= comp.props.options.length) {
            comp.props.correct_index = Math.max(0, comp.props.options.length - 1);
          }
          renderOptionsList(comp);
          el('jsonBox').value = JSON.stringify(screen, null, 2);
        };

        row.appendChild(flagBtn);
        row.appendChild(input);
        row.appendChild(delBtn);
        wrap.appendChild(row);
      });
    }

    function syncSelectedFromForm() {
      if (selectedIndex < 0 || !screen) return;
      const comp = screen.components[selectedIndex];
      const prevId = comp.id;
      comp.id = el('editId').value.trim();
      comp.type = el('editType').value;
      const act = el('editAction').value;
      if (act) comp.action = act; else delete comp.action;

      comp.props = comp.props || {};
      if (comp.type === 'floating_banner') {
        comp.props.text = el('editBannerText').value;
      } else if (comp.type === 'floating_action') {
        comp.props.icon = el('editActionIcon').value;
        const bk = el('editBadgeKey').value.trim();
        if (bk) comp.props.badge_key = bk; else delete comp.props.badge_key;
      } else if (comp.type === 'map_pin') {
        if (!comp.props.quiz_id || comp.props.quiz_id === prevId) {
          comp.props.quiz_id = comp.id;
        }
        comp.props.title = el('editPinTitle').value;
        comp.props.x_ratio = Number(el('editXRatio').value);
        comp.props.y_ratio = Number(el('editYRatio').value);
        comp.props.building_name = el('editBuildingName').value;
        comp.props.reward_points = Number(el('editRewardPoints').value) || 0;
        comp.props.question = el('editQuestion').value;
      }
      renderAll();
    }

    document.querySelectorAll('#variantSeg button').forEach((btn) => {
      btn.addEventListener('click', () => {
        currentVariant = btn.dataset.variant;
        document.querySelectorAll('#variantSeg button').forEach((b) => b.classList.toggle('active', b === btn));
        loadScreen();
      });
    });

    document.querySelectorAll('#colorSwatches .swatch').forEach((btn) => {
      btn.addEventListener('click', () => {
        if (selectedIndex < 0 || !screen) return;
        const comp = screen.components[selectedIndex];
        comp.props = comp.props || {};
        comp.props.pin_color = btn.dataset.color;
        renderAll();
      });
    });

    el('tabForm').addEventListener('click', () => {
      el('tabForm').classList.add('active');
      el('tabJson').classList.remove('active');
      el('formView').style.display = 'block';
      el('jsonView').style.display = 'none';
    });

    el('tabJson').addEventListener('click', () => {
      el('tabJson').classList.add('active');
      el('tabForm').classList.remove('active');
      el('jsonView').style.display = 'block';
      el('formView').style.display = 'none';
    });

    el('phoneFrame').addEventListener('click', (e) => {
      if (selectedIndex < 0 || !screen) return;
      const comp = screen.components[selectedIndex];
      if (!comp || comp.type !== 'map_pin') return;
      const rect = e.currentTarget.getBoundingClientRect();
      const x = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));
      const y = Math.max(0, Math.min(1, (e.clientY - rect.top) / rect.height));
      comp.props = comp.props || {};
      comp.props.x_ratio = Math.round(x * 100) / 100;
      comp.props.y_ratio = Math.round(y * 100) / 100;
      renderAll();
    });

    ['editId', 'editType', 'editAction', 'editBannerText', 'editActionIcon', 'editBadgeKey', 'editPinTitle', 'editXRatio', 'editYRatio', 'editBuildingName', 'editRewardPoints', 'editQuestion']
      .forEach((id) => {
        el(id).addEventListener('change', syncSelectedFromForm);
        if (el(id).tagName === 'TEXTAREA' || el(id).tagName === 'INPUT') {
          el(id).addEventListener('input', () => {
            if (selectedIndex < 0 || !screen) return;
            const comp = screen.components[selectedIndex];
            comp.props = comp.props || {};
            if (id === 'editQuestion') comp.props.question = el(id).value;
            if (id === 'editBuildingName') comp.props.building_name = el(id).value;
            if (id === 'editPinTitle') comp.props.title = el(id).value;
            if (id === 'editRewardPoints') comp.props.reward_points = Number(el(id).value) || 0;
            el('jsonBox').value = JSON.stringify(screen, null, 2);
          });
        }
      });

    el('addOptionBtn').addEventListener('click', () => {
      if (selectedIndex < 0 || !screen) return;
      const comp = screen.components[selectedIndex];
      comp.props = comp.props || {};
      if (!Array.isArray(comp.props.options)) comp.props.options = [];
      comp.props.options.push('Opsi baru');
      renderOptionsList(comp);
      el('jsonBox').value = JSON.stringify(screen, null, 2);
    });

    el('navTitleInput').addEventListener('change', () => { if (screen) { screen.navigation_title = el('navTitleInput').value; renderAll(); } });
    el('bgAssetInput').addEventListener('change', () => { if (screen) { screen.background_asset = el('bgAssetInput').value.trim(); renderAll(); } });
    el('schemaVersionInput').addEventListener('change', () => { if (screen) { screen.schema_version = Number(el('schemaVersionInput').value) || 1; renderAll(); } });

    el('addPinBtn').addEventListener('click', () => {
      if (!screen) return;
      const idx = screen.components.filter((c) => c.type === 'map_pin').length + 1;
      const pinId = 'pin_custom_' + idx;
      screen.components.push({
        id: pinId,
        type: 'map_pin',
        action: 'open_quiz',
        props: {
          quiz_id: pinId,
          title: 'Kuis Baru ' + idx,
          building_name: 'Gedung Baru ' + idx,
          reward_points: 150,
          question: 'Pertanyaan kuis baru?',
          options: ['Opsi A', 'Opsi B', 'Opsi C'],
          correct_index: 0,
          pin_color: 'blue',
          x_ratio: 0.5,
          y_ratio: 0.5
        }
      });
      selectedIndex = screen.components.length - 1;
      renderAll();
    });

    el('addActionBtn').addEventListener('click', () => {
      if (!screen) return;
      const idx = screen.components.filter((c) => c.type === 'floating_action').length + 1;
      screen.components.push({
        id: 'btn_custom_' + idx,
        type: 'floating_action',
        action: 'open_help',
        props: { icon: 'help' }
      });
      selectedIndex = screen.components.length - 1;
      renderAll();
    });

    el('applyJsonBtn').addEventListener('click', () => {
      try {
        screen = JSON.parse(el('jsonBox').value);
        selectedIndex = screen.components.length > 0 ? 0 : -1;
        renderAll();
        setStatus('JSON diterapkan · klik Simpan ke DB', 'ok');
      } catch (err) {
        setStatus('JSON tidak valid: ' + err.message, 'err');
      }
    });

    el('loadBtn').addEventListener('click', loadScreen);
    el('saveBtn').addEventListener('click', saveScreen);
    el('resetQuizBtn').addEventListener('click', async () => {
      setStatus('Mereset kuis user...');
      try {
        const res = await apiFetch('/api/v1/village/quizzes/reset', {
          method: 'POST',
          body: JSON.stringify({})
        });
        await loadScreen();
        setStatus('Kuis user ' + res.user_id + ' direset (' + res.deleted_submissions + ' jawaban dihapus)', 'ok');
      } catch (err) {
        setStatus('Gagal reset kuis: ' + err.message, 'err');
      }
    });

    loadScreen();
  </script>
</body>
</html>
`
