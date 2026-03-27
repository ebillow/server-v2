export GOPROXY=https://goproxy.cn,direct
export GO111MODULE=on

# 默认目标
.DEFAULT_GOAL := build

# Source directory
SOURCEDIR ?= .

# Output directory
OUT=./bin

APPS := $(shell \
  for d in $(SOURCEDIR)/*/; do \
    [ -d "$$d" ] || continue; \
    found=0; \
    for f in "$$d"/*.go; do \
      [ -f "$$f" ] || continue; \
      if grep -q '^package[[:space:]]*main' "$$f"; then found=1; break; fi; \
    done; \
    if [ $$found -eq 1 ]; then basename "$$d"; fi; \
  done \
)
APPS := $(sort $(filter-out test bin pkg tool, $(APPS)))

# 调试用：打印 APPS
.PHONY: print-apps
print-apps:
	@printf "%s\n" $(APPS)

# version flags
buildTime := $(shell date +%Y-%m-%dT%T%z)
gitCommit := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
gitTag := $(shell git --no-pager tag --points-at HEAD 2>/dev/null || echo)
define versionFlags
-X server/pkg/version.BuildTime=$(buildTime) \
-X server/pkg/version.GitCommit=$(gitCommit) \
-X server/pkg/version.GitTag=$(gitTag)
endef

BINARIES := $(patsubst %,$(OUT)/%,$(APPS))
# 确保输出目录存在
$(OUT):
	@mkdir -p $(OUT)

# 编译
.PHONY: build
build: | $(OUT)
	go build -ldflags="-s -w $(versionFlags)" -trimpath -v -o /dev/null $(foreach app,$(APPS),./$(app))

.PHONY: FORCE
FORCE:

.PHONY: build-all
build-all: $(OUT) $(BINARIES)

$(OUT)/%: FORCE
	@echo "...Building $* ..."
	go build -ldflags="-s -w $(versionFlags)" -trimpath -o $(OUT)/$* $(SOURCEDIR)/$*
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w $(versionFlags)" -tags=release -trimpath -o $(OUT)/$*.bin $(SOURCEDIR)/$*

# 跑单元测试
test:
	SI_TEST_SKIP_SLOW=1 SI_TEST_REDIS_DB=11 go test -covermode=set -coverprofile=.coverage.txt -count=1 -p=1 \
		./pkg/idgen \
		./pkg/db \
	;

# 查看测试覆盖情况
view-coverage:
	go tool cover -html=.coverage.txt

# 代码风格检查
lint:
	@bash ./tools/lint.sh <<< 'done'

lint-fix:
	@bash ./tools/lint.sh fix <<< 'done'

check: lint test

# Clean target
clean:
	rm -rf $(OUT)

# 生成消息代码
proto:
	@bash ./tools/proto/build.sh <<< 'done'

# 整理依赖
tidy:
	go mod tidy -go=1.26 -v

# 生成 model 的 helper 方法
model:
	bash ./tools/generate_model.sh <<< 'done'

# 生成 repository 的 mock stub
mock:
	go generate -v \
        ./internal/login/repository/... \
        ./internal/lobby/repository/... \
        ./internal/club/repository/... \
        ./internal/social/repository/... \
    ;

# 生成策划配置加载器集合
table:
	go run ./cmd/tableGenerater/template_generator_v2.go $(name)


# Phony targets
.PHONY: test view-coverage lint lint-fix check clean proto tidy model mock table
