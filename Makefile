
build-client:
	docker build -t scaling-cool-client:latest ./client
	# && kind load docker-image scaling-cool-client:latest

build-server:
	docker build -t scaling-cool-server:latest ./wopi-server
	# && kind load docker-image scaling-cool-server:latest

create-cluster:
	#kind create cluster --config ./infra/k8s/kind-cluster.yaml && \
	kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.13.2/deploy/static/provider/kind/deploy.yaml && \
	#docker pull collabora/code:25.04.5.1.1 && \
	#kind load docker-image collabora/code:25.04.5.1.1

k8s-nginx:
	kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.13.2/deploy/static/provider/cloud/deploy.yaml && \
	kubectl port-forward -n ingress-nginx svc/ingress-nginx-controller 8080:80

k8s-deploy:
	kubectl apply -k infra/k8s/deploy/

k8s-clean:
	kubectl delete -k infra/k8s/deploy/ --ignore-not-found=true

k8s-cool-ls:
	MSYS_NO_PATHCONV=1 kubectl exec -it collabora-online-5887f8f967-cqw8d -- ls /etc

k8s-redeploy-wopi:
	kubectl rollout restart deployment/wopi-service
