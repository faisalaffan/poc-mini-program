.PHONY: db-up db-down web-install web-test web-build backend-test backend-run backend-run-local ios-gen ios-test verify-all

LOCAL_DATABASE_URL ?= postgres://gacha:gacha_poc_password@127.0.0.1:5432/gacha_poc?sslmode=disable

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

web-install:
	cd web-gacha && npm install

web-test:
	cd web-gacha && npm run typecheck && npm test -- --run

web-build:
	cd web-gacha && npm run build

backend-test: db-up
	cd backend && TEST_DATABASE_URL="$(LOCAL_DATABASE_URL)" go test -v -race ./...

# Memakai .env di root repo (bisa mengarah ke PostgreSQL k3s VPS)
backend-run: web-build
	cd backend && go run ./cmd/server

backend-run-local: db-up web-build
	cd backend && \
		PORT=8080 \
		APP_ENV=poc \
		BUILD_VERSION=0.1.0-poc \
		DATABASE_URL="$(LOCAL_DATABASE_URL)" \
		JWT_SECRET="poc-local-dev-secret-change-in-ocp-32bytes" \
		ALLOWED_ORIGINS="http://127.0.0.1:8080,http://localhost:8080,http://macbook.singapura-parore.ts.net:8080,https://macbook.singapura-parore.ts.net" \
		WEB_DIST_DIR="../web-gacha/dist" \
		go run ./cmd/server

ios-gen:
	cd ios && xcodegen generate

ios-test: ios-gen
	xcodebuild test -project ios/GachaPOC.xcodeproj -scheme GachaPOC -destination 'platform=iOS Simulator,name=iPhone 16 POC'

verify-all: web-test web-build backend-test ios-test
