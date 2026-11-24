package postgres

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/001_init.sql
var initSchemaSQL string

//go:embed migrations/002_village_quiz.sql
var villageQuizSchemaSQL string

//go:embed migrations/003_sdui_screens.sql
var sduiScreensSchemaSQL string

// NewPool membuka koneksi pool pgx dan memverifikasi konektivitas ke PostgreSQL.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return pool, nil
}

// MigrateAndSeed menjalankan DDL idempotent dan mengisi tabel probabilitas + user demo sintetis.
func MigrateAndSeed(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, initSchemaSQL); err != nil {
		return fmt.Errorf("apply schema migration 001: %w", err)
	}
	if _, err := pool.Exec(ctx, villageQuizSchemaSQL); err != nil {
		return fmt.Errorf("apply schema migration 002: %w", err)
	}
	if _, err := pool.Exec(ctx, sduiScreensSchemaSQL); err != nil {
		return fmt.Errorf("apply schema migration 003: %w", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin seed tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Seed probability table v1.0.0 (100 poin/draw)
	const seedTableSQL = `
		INSERT INTO probability_tables (version, draw_cost, is_active)
		VALUES ('v1.0.0', 100, TRUE)
		ON CONFLICT (version) DO NOTHING;
	`
	if _, err := tx.Exec(ctx, seedTableSQL); err != nil {
		return fmt.Errorf("seed probability_tables: %w", err)
	}

	const seedTiersSQL = `
		INSERT INTO reward_tiers (id, table_version, code, name, rarity, reward_points, weight_bps, display_order)
		VALUES
			('v1_common',   'v1.0.0', 'PTS_50',   '50 Poin Harian',    'COMMON',   50,   5000, 1),
			('v1_uncommon', 'v1.0.0', 'PTS_100',  '100 Poin Cashback', 'UNCOMMON', 100,  3000, 2),
			('v1_rare',     'v1.0.0', 'PTS_250',  '250 Poin Bonus',    'RARE',     250,  1400, 3),
			('v1_epic',     'v1.0.0', 'PTS_500',  '500 Poin Spesial',  'EPIC',     500,  500,  4),
			('v1_jackpot',  'v1.0.0', 'PTS_2000', '2.000 Poin Jackpot','JACKPOT',  2000, 100,  5)
		ON CONFLICT (id) DO NOTHING;
	`
	if _, err := tx.Exec(ctx, seedTiersSQL); err != nil {
		return fmt.Errorf("seed reward_tiers: %w", err)
	}

	if err := ensureSyntheticUserTx(ctx, tx, "demo-user-001", "Nasabah POC Sintetis", 1000); err != nil {
		return err
	}

	const seedQuizzesSQL = `
		INSERT INTO village_quizzes (
			id, building_name, title, pin_color, x_ratio, y_ratio,
			reward_points, question, options, correct_index, explanation, display_order
		)
		VALUES
			(
				'quiz_tower',
				'Menara Utama',
				'Kuis Keamanan Transaksi',
				'blue',
				0.63,
				0.44,
				150,
				'Di mana token sesi login super app seharusnya disimpan saat membuka mini program WebView?',
				'["localStorage browser agar cepat", "Keychain/Secure Storage native dengan token berumur pendek ke WebView", "Query string URL halaman web", "Cookie tanpa HttpOnly"]'::jsonb,
				1,
				'Sesuai standar MASVS & arsitektur POC, master token wajib berada di Keychain native dan hanya short-lived token yang diberikan lewat bridge whitelist.',
				1
			),
			(
				'quiz_branch',
				'Cabang Kota',
				'Kuis Keandalan Ledger',
				'orange',
				0.21,
				0.54,
				100,
				'Mengapa mutasi poin hadiah dicatat di ledger append-only dengan Idempotency-Key?',
				'["Agar retry jaringan tidak mengkredit hadiah dua kali & saldo bisa direkonsiliasi", "Supaya ukuran database lebih kecil", "Agar client bisa mengubah saldo sendiri", "Untuk melewati proses audit log"]'::jsonb,
				0,
				'Append-only ledger mencegah perubahan saldo diam-diam, dan Idempotency-Key menjamin permintaan ulang karena timeout menghasilkan respons yang sama persis.',
				2
			),
			(
				'quiz_bakery',
				'Kafe Livin'' Village',
				'Kuis Server-Driven UI',
				'red',
				0.75,
				0.69,
				120,
				'Bagaimana renderer SDUI yang aman menangani skema JSON atau aksi baru yang belum dikenal aplikasi?',
				'["Mengeksekusi string kode JS secara dinamis", "Memvalidasi whitelist komponen/aksi dan fallback ke layar bawaan saat versi tidak cocok", "Mengabaikan schema_version", "Menampilkan layar kosong selamanya"]'::jsonb,
				1,
				'Renderer SDUI memvalidasi schema_version serta whitelist komponen/aksi, dan otomatis memakai layar fallback bawaan bila menerima skema yang belum didukung.',
				3
			)
		ON CONFLICT (id) DO NOTHING;
	`
	if _, err := tx.Exec(ctx, seedQuizzesSQL); err != nil {
		return fmt.Errorf("seed village_quizzes: %w", err)
	}

	const seedSDUIScreensSQL = `
		INSERT INTO sdui_screens (
			screen_id, variant, schema_version, navigation_title, background_asset,
			supported_actions, components
		)
		VALUES
			(
				'livin_village_home',
				'default',
				1,
				'Livin'' Village',
				'bg-isometric',
				'["open_quiz","open_help","open_rewards","refresh"]'::jsonb,
				'[
					{"id":"village_banner","type":"floating_banner","props":{"text":"Mainkan kuis Livin\u0027 Village & dapatkan hadiah menarik pada 1 Jun 2026 - 31 Dec 2026","tone":"primary"}},
					{"id":"fab_help","type":"floating_action","props":{"icon":"help","color":"amber","slot":"top_right_1","label":"Panduan"},"action":"open_help"},
					{"id":"fab_rewards","type":"floating_action","props":{"icon":"gift","color":"blue","slot":"top_right_2","label":"Hadiah"},"action":"open_rewards"},
					{"id":"pin_quiz_tower","type":"map_pin","props":{"quiz_id":"quiz_tower","building_name":"Menara Utama","title":"Kuis Keamanan Transaksi","pin_color":"blue","x_ratio":0.63,"y_ratio":0.44,"reward_points":150},"action":"open_quiz"},
					{"id":"pin_quiz_branch","type":"map_pin","props":{"quiz_id":"quiz_branch","building_name":"Cabang Kota","title":"Kuis Keandalan Ledger","pin_color":"orange","x_ratio":0.21,"y_ratio":0.54,"reward_points":100},"action":"open_quiz"},
					{"id":"pin_quiz_bakery","type":"map_pin","props":{"quiz_id":"quiz_bakery","building_name":"Kafe Livin\u0027 Village","title":"Kuis Server-Driven UI","pin_color":"red","x_ratio":0.75,"y_ratio":0.69,"reward_points":120},"action":"open_quiz"}
				]'::jsonb
			),
			(
				'livin_village_home',
				'promo',
				1,
				'Livin'' Village',
				'bg-isometric',
				'["open_quiz","open_help","open_rewards","refresh"]'::jsonb,
				'[
					{"id":"village_banner","type":"floating_banner","props":{"text":"PROMO SDUI · Selesaikan 3 kuis Livin\u0027 Village & kumpulkan total 370 poin hadiah!","tone":"promo"}},
					{"id":"fab_help","type":"floating_action","props":{"icon":"help","color":"amber","slot":"top_right_1","label":"Panduan"},"action":"open_help"},
					{"id":"fab_rewards","type":"floating_action","props":{"icon":"gift","color":"blue","slot":"top_right_2","label":"Hadiah"},"action":"open_rewards"},
					{"id":"pin_quiz_tower","type":"map_pin","props":{"quiz_id":"quiz_tower","building_name":"Menara Utama","title":"Kuis Keamanan Transaksi","pin_color":"orange","x_ratio":0.63,"y_ratio":0.44,"reward_points":150},"action":"open_quiz"},
					{"id":"pin_quiz_branch","type":"map_pin","props":{"quiz_id":"quiz_branch","building_name":"Cabang Kota","title":"Kuis Keandalan Ledger","pin_color":"orange","x_ratio":0.21,"y_ratio":0.54,"reward_points":100},"action":"open_quiz"},
					{"id":"pin_quiz_bakery","type":"map_pin","props":{"quiz_id":"quiz_bakery","building_name":"Kafe Livin\u0027 Village","title":"Kuis Server-Driven UI","pin_color":"orange","x_ratio":0.75,"y_ratio":0.69,"reward_points":120},"action":"open_quiz"}
				]'::jsonb
			),
			(
				'livin_village_home',
				'compact',
				1,
				'Livin'' Village',
				'bg-isometric',
				'["open_quiz","open_help","open_rewards","refresh"]'::jsonb,
				'[
					{"id":"village_banner","type":"floating_banner","props":{"text":"Kuis Livin\u0027 Village · 1 Jun 2026 - 31 Dec 2026","tone":"compact"}},
					{"id":"fab_help","type":"floating_action","props":{"icon":"help","color":"amber","slot":"top_right_1","label":"Panduan"},"action":"open_help"},
					{"id":"fab_rewards","type":"floating_action","props":{"icon":"gift","color":"blue","slot":"top_right_2","label":"Hadiah"},"action":"open_rewards"},
					{"id":"pin_quiz_tower","type":"map_pin","props":{"quiz_id":"quiz_tower","building_name":"Menara Utama","title":"Kuis Keamanan Transaksi","pin_color":"blue","x_ratio":0.63,"y_ratio":0.44,"reward_points":150},"action":"open_quiz"},
					{"id":"pin_quiz_branch","type":"map_pin","props":{"quiz_id":"quiz_branch","building_name":"Cabang Kota","title":"Kuis Keandalan Ledger","pin_color":"orange","x_ratio":0.21,"y_ratio":0.54,"reward_points":100},"action":"open_quiz"},
					{"id":"pin_quiz_bakery","type":"map_pin","props":{"quiz_id":"quiz_bakery","building_name":"Kafe Livin\u0027 Village","title":"Kuis Server-Driven UI","pin_color":"red","x_ratio":0.75,"y_ratio":0.69,"reward_points":120},"action":"open_quiz"}
				]'::jsonb
			)
		ON CONFLICT (screen_id, variant) DO NOTHING;
	`
	if _, err := tx.Exec(ctx, seedSDUIScreensSQL); err != nil {
		return fmt.Errorf("seed sdui_screens: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed tx: %w", err)
	}
	return nil
}
