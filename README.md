# docker-demo

```console
$ go run main.go -image tigerworks/labels

$ ./make.sh
$ docker tag appscode/docker-image-puller gcr.io/tigerworks-kube/docker-image-puller
$ docker push gcr.io/tigerworks-kube/docker-image-puller

$ kubectl create serviceaccount image-puller
$ kubectl create clusterrolebinding image-puller --clusterrole=cluster-admin --serviceaccount=default:image-puller
$ kubectl run image-puller --image=appscode/docker-image-puller --serviceaccount=image-puller
```

## Docs
- https://kubernetes.io/docs/concepts/containers/images/#updating-images
- https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/
