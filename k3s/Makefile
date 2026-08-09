TMP_DIR  := ./tmp
IMG_DIR  := ./images
PROJECT  := subscriptions
VER      ?= v0.0.1
REPLICAS := 3

create: ns-create \
	migrate-build migrate-install \
	server-build server-install \
	db-create server-create \
	migrate-run server-start

reload: server-reload

reload-with-migrate: migrate-build migrate-install \
	server-build server-install \
	server-stop \
	migrate-run \
	server-start

clean:
	-rm -fr $(TMP_DIR) $(IMG_DIR)

# ---------- Инфраструктура (создаётся один раз) ----------
ns-create:
	kubectl create namespace $(PROJECT) 2>/dev/null || true
	kubectl apply -n $(PROJECT) -f 00-configmap.yaml
	@kubectl get secret db-secret -n $(PROJECT) >/dev/null 2>&1 || \
		kubectl create secret generic db-secret -n $(PROJECT) \
		--from-literal=DB_PASSWORD="$$(head -c16 /dev/urandom | base64)"
	kubectl apply -n $(PROJECT) -f 03-db-service.yaml
	kubectl apply -n $(PROJECT) -f 06-server-service.yaml

ns-destroy: ## destroy all!
	kubectl delete namespace $(PROJECT)

# ---------- База данных ----------
db-create: ns-create
	kubectl apply -n $(PROJECT) -f 02-db-statefulset.yaml

db-delete:
	kubectl delete -n $(PROJECT) -f 02-db-statefulset.yaml   # StatefulSet удаляется, PVC остаётся

# ---------- Миграции ----------
migrate-build:
	@mkdir -p $(TMP_DIR)
	@mkdir -p $(IMG_DIR)
	docker build -t $(PROJECT)/migrate:$(VER) --target migrate ./..
	docker save $(PROJECT)/migrate:$(VER) -o $(TMP_DIR)/migrate-$(VER).tar.$$$$ && \
	mv $(TMP_DIR)/migrate-$(VER).tar.$$$$ $(IMG_DIR)/migrate-$(VER).tar

migrate-install:
	k3s ctr images import $(IMG_DIR)/migrate-$(VER).tar

# Запустить миграцию
migrate-run:
	kubectl delete job migrate -n $(PROJECT) --ignore-not-found
	kubectl wait -n $(PROJECT) --for=condition=ready --for=create pod -l app=db --timeout=60s
	kubectl apply -n $(PROJECT) -f 04-migrate-job.yaml
	kubectl wait -n $(PROJECT) --for=condition=complete --for=create job/migrate --timeout=60s

# ---------- Сервер ----------
server-build:
	@mkdir -p $(TMP_DIR)
	@mkdir -p $(IMG_DIR)
	docker build -t $(PROJECT)/server:$(VER) --target server ./..
	docker save $(PROJECT)/server:$(VER) -o $(TMP_DIR)/server-$(VER).tar.$$$$ && \
	mv $(TMP_DIR)/server-$(VER).tar.$$$$ $(IMG_DIR)/server-$(VER).tar

server-install:
	k3s ctr images import $(IMG_DIR)/server-$(VER).tar

# Создать/обновить Deployment (один раз)
server-create:
	kubectl apply -n $(PROJECT) -f 05-server-deployment.yaml

# Запустить поды (масштабировать до REPLICAS)
server-start:
	kubectl scale -n $(PROJECT) deployment/server --replicas=$(REPLICAS)

# Остановить поды (масштабировать до 0)
server-stop:
	kubectl scale -n $(PROJECT) deployment/server --replicas=0

# Пересобрать, загрузить и перезапустить (без остановки)
server-reload: server-build server-install
	kubectl rollout restart -n $(PROJECT) deployment/server
	kubectl rollout status -n $(PROJECT) deployment/server
