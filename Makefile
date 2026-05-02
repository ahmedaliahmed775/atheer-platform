# Makefile — أوامر بناء وتشغيل سويتش Atheer
# يُرجى الرجوع إلى CLAUDE.md — بنية المستودع

# ── المتغيرات ──
BINARY     := atheer-switch
SWITCH_DIR := switch
BUILD_DIR  := $(SWITCH_DIR)/build
DOCKER_IMG := atheer-switch
GO         := go
CONFIG     := config.yaml

# ── الأهداف الافتراضية ──
.PHONY: all
all: build

# ── بناء الثنائي ──
.PHONY: build
build:
	cd $(SWITCH_DIR) && $(GO) build -ldflags="-s -w" -o build/$(BINARY) ./cmd/server

# ── بناء أمر الترحيل ──
.PHONY: build-migrate
build-migrate:
	cd $(SWITCH_DIR) && $(GO) build -ldflags="-s -w" -o build/migrate ./cmd/migrate

# ── تشغيل الاختبارات ──
.PHONY: test
test:
	cd $(SWITCH_DIR) && $(GO) test ./... -v -count=1

# ── تشغيل الاختبارات مع تغطية ──
.PHONY: test-cover
test-cover:
	cd $(SWITCH_DIR) && $(GO) test ./... -cover -count=1

# ── تشغيل الخادم محلياً ──
.PHONY: run
run:
	cd $(SWITCH_DIR) && $(GO) run ./cmd/server -config $(CONFIG)

# ── تشغيل الترحيلات فقط ──
.PHONY: migrate
migrate:
	cd $(SWITCH_DIR) && $(GO) run ./cmd/migrate -config $(CONFIG)

# ── تنسيق الكود ──
.PHONY: fmt
fmt:
	cd $(SWITCH_DIR) && $(GO) fmt ./...

# ── فحص الكود ──
.PHONY: vet
vet:
	cd $(SWITCH_DIR) && $(GO) vet ./...

# ── بناء صورة Docker ──
.PHONY: docker-build
docker-build:
	docker build -t $(DOCKER_IMG):latest $(SWITCH_DIR)

# ── تشغيل صورة Docker ──
.PHONY: docker-run
docker-run:
	docker run --rm -p 8080:8080 \
		-e DB_PASSWORD=$(DB_PASSWORD) \
		-e KMS_MASTER_KEY=$(KMS_MASTER_KEY) \
		-e JWT_SECRET=$(JWT_SECRET) \
		$(DOCKER_IMG):latest

# ── تنظيف ملفات البناء ──
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

# ── عرض المساعدة ──
.PHONY: help
help:
	@echo "أوامر سويتش Atheer:"
	@echo "  make build          — بناء الثنائي"
	@echo "  make build-migrate  — بناء أمر الترحيل"
	@echo "  make test           — تشغيل الاختبارات"
	@echo "  make test-cover     — تشغيل الاختبارات مع تغطية"
	@echo "  make run            — تشغيل الخادم محلياً"
	@echo "  make migrate        — تشغيل الترحيلات فقط"
	@echo "  make fmt            — تنسيق الكود"
	@echo "  make vet            — فحص الكود"
	@echo "  make docker-build   — بناء صورة Docker"
	@echo "  make docker-run     — تشغيل صورة Docker"
	@echo "  make clean          — تنظيف ملفات البناء"
