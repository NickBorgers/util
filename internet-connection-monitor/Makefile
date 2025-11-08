.PHONY: help grafana-dashboard-demo demo-start demo-stop demo-logs demo-status demo-clean \
        build run test clean setup-es verify-grafana open-grafana open-prometheus \
        es-health es-indices es-sample monitor-logs es-logs grafana-logs all-logs \
        quick-start quick-test quick-stop build-binary run-binary dev watch-json

# Default target
.DEFAULT_GOAL := help

# Colors for output
CYAN := \033[0;36m
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

##@ Quick Start Commands

quick-start: build ## ⚡ Quick start: Build image and run monitor (fastest way to test)
	@echo "$(CYAN)╔════════════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(CYAN)║  Quick Start - Internet Connection Monitor                    ║$(NC)"
	@echo "$(CYAN)╚════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "$(YELLOW)Starting monitor in standalone mode...$(NC)"
	@docker run --rm --name quick-monitor \
		-e INTER_TEST_DELAY=3s \
		-e LOG_FORMAT=json \
		internet-connection-monitor:latest &
	@sleep 3
	@echo ""
	@echo "$(GREEN)✓ Monitor is running!$(NC)"
	@echo ""
	@echo "$(YELLOW)Viewing test results (Ctrl+C to stop):$(NC)"
	@echo "$(CYAN)────────────────────────────────────────────────────────────────$(NC)"
	@docker logs -f quick-monitor 2>&1 | grep -v "ERROR: could not unmarshal"

quick-test: build ## 🧪 Quick test: Run monitor for 30 seconds and show results
	@echo "$(CYAN)Running quick 30-second test...$(NC)"
	@timeout 30 docker run --rm \
		-e INTER_TEST_DELAY=5s \
		-e SITES="google.com,github.com,example.com" \
		internet-connection-monitor:latest 2>&1 \
		| grep -v "ERROR: could not unmarshal" || true
	@echo ""
	@echo "$(GREEN)✓ Test complete$(NC)"

quick-stop: ## 🛑 Stop quick-start monitor
	@docker stop quick-monitor 2>/dev/null || echo "$(YELLOW)No quick-monitor running$(NC)"

watch-json: ## 👀 Watch test results in real-time (formatted JSON)
	@echo "$(CYAN)Watching test results (Ctrl+C to stop)...$(NC)"
	@echo "$(CYAN)────────────────────────────────────────────────────────────────$(NC)"
	@docker logs -f internet-monitor 2>&1 \
		| grep -v "ERROR: could not unmarshal" \
		| grep "^{" \
		| jq -C '.' || docker logs -f quick-monitor 2>&1 | grep "^{"

##@ Local Development Commands

build-binary: ## 🔧 Build Go binary locally (to build/ directory)
	@echo "$(YELLOW)Building Go binary...$(NC)"
	@mkdir -p build
	@go build -o build/internet-monitor ./cmd/monitor
	@echo "$(GREEN)✓ Binary built: build/internet-monitor$(NC)"
	@ls -lh build/internet-monitor

run-binary: build-binary ## ▶️  Run locally built binary (requires Chrome installed)
	@echo "$(YELLOW)Running locally built binary...$(NC)"
	@echo "$(YELLOW)Note: Requires Chrome/Chromium installed on system$(NC)"
	@echo ""
	@./build/internet-monitor

dev: ## 🔄 Development mode: Build and run with quick reload
	@echo "$(CYAN)Development mode - Building and running...$(NC)"
	@$(MAKE) build-binary
	@echo ""
	@echo "$(YELLOW)Starting monitor (Ctrl+C to stop)...$(NC)"
	@SITES="example.com,google.com" INTER_TEST_DELAY=5s ./build/internet-monitor

##@ Demo Commands

grafana-dashboard-demo: ## 🚀 Start full Grafana dashboard demo (Elasticsearch + Grafana + Monitor)
	@echo "$(CYAN)╔════════════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(CYAN)║  Internet Connection Monitor - Grafana Dashboard Demo         ║$(NC)"
	@echo "$(CYAN)╚════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "$(YELLOW)Starting full monitoring stack...$(NC)"
	@cd deployments && docker-compose -f docker-compose.with-stack.yml up -d
	@echo ""
	@echo "$(YELLOW)Waiting for services to be ready...$(NC)"
	@sleep 5
	@echo "$(GREEN)✓ Elasticsearch starting...$(NC)"
	@sleep 10
	@echo "$(GREEN)✓ Grafana starting...$(NC)"
	@sleep 10
	@echo "$(GREEN)✓ Monitor starting...$(NC)"
	@sleep 5
	@echo ""
	@echo "$(CYAN)════════════════════════════════════════════════════════════════$(NC)"
	@echo "$(GREEN)✓ Demo stack is running!$(NC)"
	@echo "$(CYAN)════════════════════════════════════════════════════════════════$(NC)"
	@echo ""
	@echo "$(YELLOW)Access points:$(NC)"
	@echo "  $(GREEN)Grafana Dashboard:$(NC)     http://localhost:3000"
	@echo "  $(GREEN)  Username:$(NC)            admin"
	@echo "  $(GREEN)  Password:$(NC)            admin"
	@echo ""
	@echo "  $(GREEN)Elasticsearch:$(NC)         http://localhost:9200"
	@echo "  $(GREEN)Prometheus:$(NC)            http://localhost:9091"
	@echo "  $(GREEN)Monitor Metrics:$(NC)       http://localhost:9090/metrics"
	@echo ""
	@echo "$(YELLOW)Useful commands:$(NC)"
	@echo "  make demo-logs          - View all container logs"
	@echo "  make monitor-logs       - View monitor test results"
	@echo "  make demo-status        - Check service health"
	@echo "  make es-indices         - List Elasticsearch indices"
	@echo "  make verify-grafana     - Verify Grafana is working"
	@echo "  make demo-stop          - Stop the demo"
	@echo "  make demo-clean         - Stop and remove all data"
	@echo ""
	@echo "$(CYAN)════════════════════════════════════════════════════════════════$(NC)"
	@echo "$(YELLOW)Note: It may take 30-60 seconds for all services to be fully ready$(NC)"
	@echo "$(CYAN)════════════════════════════════════════════════════════════════$(NC)"

demo-start: grafana-dashboard-demo ## Alias for grafana-dashboard-demo

demo-stop: ## 🛑 Stop the demo stack (keeps data)
	@echo "$(YELLOW)Stopping demo stack...$(NC)"
	@cd deployments && docker-compose -f docker-compose.with-stack.yml stop
	@echo "$(GREEN)✓ Demo stack stopped$(NC)"
	@echo "$(YELLOW)Data is preserved. Use 'make grafana-dashboard-demo' to restart$(NC)"

demo-down: ## ⏹️  Stop and remove containers (keeps volumes)
	@echo "$(YELLOW)Stopping and removing demo containers...$(NC)"
	@cd deployments && docker-compose -f docker-compose.with-stack.yml down
	@echo "$(GREEN)✓ Containers removed$(NC)"
	@echo "$(YELLOW)Data volumes preserved. Use 'make grafana-dashboard-demo' to restart$(NC)"

demo-clean: ## 🗑️  Stop demo and remove all data (DESTRUCTIVE)
	@echo "$(RED)WARNING: This will delete all monitoring data!$(NC)"
	@echo -n "$(YELLOW)Continue? [y/N] $(NC)" && read ans && [ $${ans:-N} = y ]
	@echo "$(YELLOW)Stopping containers and removing volumes...$(NC)"
	@cd deployments && docker-compose -f docker-compose.with-stack.yml down -v
	@echo "$(GREEN)✓ All demo data cleaned$(NC)"

demo-restart: demo-stop grafana-dashboard-demo ## 🔄 Restart the demo stack

demo-status: ## 📊 Show status of all demo services
	@echo "$(CYAN)════════════════════════════════════════════════════════════════$(NC)"
	@echo "$(CYAN)  Demo Stack Status$(NC)"
	@echo "$(CYAN)════════════════════════════════════════════════════════════════$(NC)"
	@cd deployments && docker-compose -f docker-compose.with-stack.yml ps
	@echo ""
	@echo "$(YELLOW)Service Health Checks:$(NC)"
	@echo -n "  Elasticsearch: "
	@curl -s -o /dev/null -w "%{http_code}" http://localhost:9200/_cluster/health 2>/dev/null | grep -q 200 && echo "$(GREEN)✓ Healthy$(NC)" || echo "$(RED)✗ Not responding$(NC)"
	@echo -n "  Grafana:       "
	@curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/api/health 2>/dev/null | grep -q 200 && echo "$(GREEN)✓ Healthy$(NC)" || echo "$(RED)✗ Not responding$(NC)"
	@echo -n "  Prometheus:    "
	@curl -s -o /dev/null -w "%{http_code}" http://localhost:9091/-/healthy 2>/dev/null | grep -q 200 && echo "$(GREEN)✓ Healthy$(NC)" || echo "$(RED)✗ Not responding$(NC)"
	@echo -n "  Monitor:       "
	@curl -s -o /dev/null -w "%{http_code}" http://localhost:9090/metrics 2>/dev/null | grep -q 200 && echo "$(GREEN)✓ Healthy$(NC)" || echo "$(RED)✗ Not responding$(NC)"

demo-logs: ## 📜 View logs from all demo services (monitor logs filtered)
	@cd deployments && docker-compose -f docker-compose.with-stack.yml logs -f 2>&1 | grep -v "ERROR: could not unmarshal"

##@ Logs and Debugging

monitor-logs: ## 📝 View internet-monitor test results (filtered, clean output)
	@echo "$(CYAN)Monitoring test results (Ctrl+C to exit):$(NC)"
	@echo "$(YELLOW)Filtering out harmless CDP warnings...$(NC)"
	@cd deployments && docker-compose -f docker-compose.with-stack.yml logs -f internet-monitor 2>&1 | grep -v "ERROR: could not unmarshal"

es-logs: ## 📝 View Elasticsearch logs
	@cd deployments && docker-compose -f docker-compose.with-stack.yml logs -f elasticsearch

grafana-logs: ## 📝 View Grafana logs
	@cd deployments && docker-compose -f docker-compose.with-stack.yml logs -f grafana

all-logs: demo-logs ## 📝 View all container logs

##@ Elasticsearch Commands

es-health: ## 🏥 Check Elasticsearch cluster health
	@echo "$(CYAN)Elasticsearch Cluster Health:$(NC)"
	@curl -s http://localhost:9200/_cluster/health?pretty

es-indices: ## 📑 List Elasticsearch indices
	@echo "$(CYAN)Elasticsearch Indices:$(NC)"
	@curl -s http://localhost:9200/_cat/indices/internet-connection-monitor-*?v

es-count: ## 🔢 Count documents in Elasticsearch
	@echo "$(CYAN)Document Count:$(NC)"
	@curl -s http://localhost:9200/internet-connection-monitor-*/_count?pretty

es-sample: ## 🔍 Show a sample test result from Elasticsearch
	@echo "$(CYAN)Sample Test Result (most recent):$(NC)"
	@curl -s http://localhost:9200/internet-connection-monitor-*/_search?size=1&sort=@timestamp:desc&pretty | jq '.hits.hits[0]._source'

es-stats: ## 📊 Show Elasticsearch storage stats
	@echo "$(CYAN)Index Statistics:$(NC)"
	@curl -s http://localhost:9200/_cat/indices/internet-connection-monitor-*?v&h=index,docs.count,store.size,pri.store.size

es-delete-all: ## ⚠️  Delete all test data from Elasticsearch (DESTRUCTIVE)
	@echo "$(RED)WARNING: This will delete all test data!$(NC)"
	@echo -n "$(YELLOW)Continue? [y/N] $(NC)" && read ans && [ $${ans:-N} = y ]
	@curl -X DELETE "http://localhost:9200/internet-connection-monitor-*"
	@echo "$(GREEN)✓ All test data deleted$(NC)"

##@ Grafana Commands

verify-grafana: ## ✅ Verify Grafana is accessible and dashboard exists
	@echo "$(CYAN)Verifying Grafana...$(NC)"
	@echo -n "  API Health: "
	@curl -s http://localhost:3000/api/health | jq -r '.database' | grep -q ok && echo "$(GREEN)✓$(NC)" || echo "$(RED)✗$(NC)"
	@echo -n "  Datasource: "
	@curl -s http://admin:admin@localhost:3000/api/datasources | jq -r '.[0].name' | grep -q Elasticsearch && echo "$(GREEN)✓ Elasticsearch connected$(NC)" || echo "$(RED)✗ Not found$(NC)"
	@echo "$(YELLOW)  Dashboard URL:$(NC) http://localhost:3000/d/internet-conn-mon/internet-connection-monitor"

open-grafana: ## 🌐 Open Grafana in browser (macOS/Linux)
	@echo "$(CYAN)Opening Grafana in browser...$(NC)"
	@command -v open >/dev/null 2>&1 && open http://localhost:3000 || \
	 command -v xdg-open >/dev/null 2>&1 && xdg-open http://localhost:3000 || \
	 echo "$(YELLOW)Please open http://localhost:3000 in your browser$(NC)"

open-prometheus: ## 🌐 Open Prometheus in browser (macOS/Linux)
	@echo "$(CYAN)Opening Prometheus in browser...$(NC)"
	@command -v open >/dev/null 2>&1 && open http://localhost:9091 || \
	 command -v xdg-open >/dev/null 2>&1 && xdg-open http://localhost:9091 || \
	 echo "$(YELLOW)Please open http://localhost:9091 in your browser$(NC)"

##@ Development Commands

build: ## 🔨 Build the monitor Docker image
	@echo "$(YELLOW)Building internet-connection-monitor image...$(NC)"
	@docker build -t internet-connection-monitor:latest .
	@echo "$(GREEN)✓ Build complete$(NC)"

run: ## ▶️  Run monitor in standalone mode (no Elasticsearch)
	@echo "$(YELLOW)Starting monitor in standalone mode...$(NC)"
	@cd deployments && docker-compose up -d
	@echo "$(GREEN)✓ Monitor running$(NC)"
	@echo "$(YELLOW)View logs:$(NC) make logs-standalone"

stop: ## ⏹️  Stop standalone monitor
	@cd deployments && docker-compose stop

logs-standalone: ## 📜 View standalone monitor logs (filtered)
	@echo "$(CYAN)Viewing monitor logs (CDP warnings filtered)...$(NC)"
	@cd deployments && docker-compose logs -f internet-monitor 2>&1 | grep -v "ERROR: could not unmarshal"

clean-standalone: ## 🗑️  Stop and remove standalone monitor
	@cd deployments && docker-compose down

##@ Setup and Configuration

setup-es: ## 🔧 Manually setup Elasticsearch (index template and ILM policy)
	@echo "$(YELLOW)Setting up Elasticsearch...$(NC)"
	@echo "$(CYAN)Creating ILM policy...$(NC)"
	@curl -X PUT "http://localhost:9200/_ilm/policy/internet-monitor-ilm-policy" \
	  -H 'Content-Type: application/json' \
	  -d @elasticsearch-ilm-policy.json
	@echo ""
	@echo "$(CYAN)Creating index template...$(NC)"
	@curl -X PUT "http://localhost:9200/_index_template/internet-connection-monitor" \
	  -H 'Content-Type: application/json' \
	  -d @elasticsearch-index-template.json
	@echo ""
	@echo "$(GREEN)✓ Elasticsearch setup complete$(NC)"

env-example: ## 📄 Create .env file from example
	@if [ -f deployments/.env ]; then \
		echo "$(YELLOW)deployments/.env already exists. Skipping...$(NC)"; \
	else \
		cp deployments/.env.example deployments/.env; \
		echo "$(GREEN)✓ Created deployments/.env from example$(NC)"; \
		echo "$(YELLOW)Edit deployments/.env to customize configuration$(NC)"; \
	fi

##@ Monitoring and Testing

test-monitor: ## 🧪 Run a quick test to verify monitor is working
	@echo "$(CYAN)Testing monitor connectivity...$(NC)"
	@echo ""
	@echo "$(YELLOW)Checking if monitor is running...$(NC)"
	@docker ps | grep internet-monitor > /dev/null || (echo "$(RED)✗ Monitor not running$(NC)" && exit 1)
	@echo "$(GREEN)✓ Monitor container is running$(NC)"
	@echo ""
	@echo "$(YELLOW)Checking Prometheus metrics endpoint...$(NC)"
	@curl -s http://localhost:9090/metrics | head -5
	@echo "$(GREEN)✓ Metrics endpoint responding$(NC)"
	@echo ""
	@echo "$(YELLOW)Waiting for test results...$(NC)"
	@sleep 10
	@echo "$(YELLOW)Recent test results:$(NC)"
	@cd deployments && docker-compose -f docker-compose.with-stack.yml logs --tail=5 internet-monitor
	@echo ""
	@echo "$(GREEN)✓ Monitor is functioning$(NC)"

snmp-test: ## 🔌 Test SNMP connectivity (requires snmpwalk)
	@echo "$(CYAN)Testing SNMP...$(NC)"
	@command -v snmpwalk >/dev/null 2>&1 || (echo "$(RED)snmpwalk not installed$(NC)" && exit 1)
	@snmpwalk -v2c -c public localhost:161 .1.3.6.1.4.1

watch-tests: ## 👀 Watch test results in real-time
	@echo "$(CYAN)Watching test results (Ctrl+C to exit)...$(NC)"
	@cd deployments && docker-compose -f docker-compose.with-stack.yml logs -f internet-monitor | jq -R 'fromjson? | select(.site != null) | {time: ."@timestamp", site: .site.name, success: .status.success, latency: .timings.total_duration_ms}'

##@ Documentation

readme: ## 📖 Open main README
	@command -v mdcat >/dev/null 2>&1 && mdcat README.md || cat README.md

design: ## 📐 Open design document
	@command -v mdcat >/dev/null 2>&1 && mdcat DESIGN.md || cat DESIGN.md

es-guide: ## 📚 Open Elasticsearch and Grafana guide
	@command -v mdcat >/dev/null 2>&1 && mdcat ELASTICSEARCH_AND_GRAFANA.md || cat ELASTICSEARCH_AND_GRAFANA.md

deploy-guide: ## 🚀 Open deployment guide
	@command -v mdcat >/dev/null 2>&1 && mdcat deployments/README.md || cat deployments/README.md

##@ Utility Commands

ps: demo-status ## 📊 Alias for demo-status

logs: demo-logs ## 📜 Alias for demo-logs

shell-monitor: ## 🐚 Open shell in monitor container
	@cd deployments && docker-compose -f docker-compose.with-stack.yml exec internet-monitor /bin/sh

shell-es: ## 🐚 Open shell in Elasticsearch container
	@cd deployments && docker-compose -f docker-compose.with-stack.yml exec elasticsearch /bin/bash

shell-grafana: ## 🐚 Open shell in Grafana container
	@cd deployments && docker-compose -f docker-compose.with-stack.yml exec grafana /bin/bash

clean-docker: ## 🧹 Remove all stopped containers and unused images
	@echo "$(YELLOW)Cleaning Docker...$(NC)"
	@docker system prune -f
	@echo "$(GREEN)✓ Docker cleaned$(NC)"

##@ Help

help: ## 📚 Show this help message
	@echo "$(CYAN)╔════════════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(CYAN)║  Internet Connection Monitor - Makefile Commands              ║$(NC)"
	@echo "$(CYAN)╚════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf ""} /^[a-zA-Z_-]+:.*?##/ { printf "  $(CYAN)%-20s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(YELLOW)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(YELLOW)╔══════════════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(YELLOW)║  Quick Start Workflows                                          ║$(NC)"
	@echo "$(YELLOW)╚══════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "$(GREEN)1. Just test it quickly (30 seconds):$(NC)"
	@echo "   $(CYAN)make quick-test$(NC)"
	@echo ""
	@echo "$(GREEN)2. Run and watch live results:$(NC)"
	@echo "   $(CYAN)make quick-start$(NC)          # Build & run in background"
	@echo "   $(CYAN)Ctrl+C to stop$(NC)            # Stops viewing (keeps running)"
	@echo "   $(CYAN)make quick-stop$(NC)           # Stop the monitor"
	@echo ""
	@echo "$(GREEN)3. Full stack with Grafana dashboard:$(NC)"
	@echo "   $(CYAN)make grafana-dashboard-demo$(NC)  # Start everything"
	@echo "   $(CYAN)make demo-status$(NC)             # Check health"
	@echo "   $(CYAN)make monitor-logs$(NC)            # View test results"
	@echo "   Open $(GREEN)http://localhost:3000$(NC)    (admin/admin)"
	@echo ""
	@echo "$(GREEN)4. Local development (requires Chrome):$(NC)"
	@echo "   $(CYAN)make build-binary$(NC)         # Build Go binary"
	@echo "   $(CYAN)make dev$(NC)                  # Build & run locally"
	@echo ""
	@echo "$(GREEN)5. Watch formatted JSON output:$(NC)"
	@echo "   $(CYAN)make watch-json$(NC)           # Pretty-print test results"
	@echo ""
