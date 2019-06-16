docker build -t weibh/metrics .
docker push weibh/metrics
kubectl apply -f deploy.yaml
kubectl scale deploy -n kube-system  metrics --replicas=0
kubectl scale deploy  -n kube-system metrics --replicas=1
kubectl get pods  -n kube-system | grep metrics