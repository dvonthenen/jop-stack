# JOP Demo

## Before we begin...

This step-by-step walkthrough was initially forked from Kelsey Hightower's CNCF demo repo (located here: https://github.com/kelseyhightower/cloud-native-demo/), but now has been heavily modified since the original demo uses Zipkin and this repo use Jaeger. I was tasked look into standing up a JOP stack, but a lot of the explanation of what was going on, how to deploy the individual components, and etc was missing from that repo. Additionally, a number of changes have been made to the Kubernetes YAML files (going from ReplicaSets to Deployments, using NodePorts), the Prometheus configuration file itself (adds in Kubernetes, and Prometheus monitoring), and even the code to the Sample Application itself (I was not able to get the program to run correctly without these modifications).

Before we begin, you will need to have an existing Kubernetes cluster up and running with a decent amount of memory and CPU on each Kubernetes worker node. If you need to stand up a cluster, there is documentation available on the official Kubernetes site for [setup](https://kubernetes.io/docs/setup/). It is assumed that you have a good understanding and grasp operating/navigating a Kubernetes cluster.

To simplify and condense this walkthrough, you can find all configuration files this GitHub repo.

## Installation

### Storage Persistency with Prometheus
<pre>
# We need to create a k8s secret to hold onto the username and
# password for ScaleIO
cd configs
kubectl create -f scaleio-default.yaml
kubectl create -f scaleio-kubesystem.yaml
cd ..

# We need to define the configuration parameters both for ScaleIO
# and also hooking up k8s persistent volume claims
cd scaleio
kubectl create -f storageclass.yaml
kubectl create -f persistvolumeclaim-default.yaml
kubectl create -f persistvolumeclaim-kubesystem.yaml
cd ..
</pre>

### Deploying Jaeger
<pre>
# We need to open up the Jaeger port to access the UI.
# If you are behind a firewall whether it's on-prem or in your
# favorite cloud like GCE, don't forget to open up the NodePort
# that is allocated!
cd services
kubectl create -f jaeger.yaml
cd ..

# Let's deploy Jaeger
cd deployments
kubectl create -f jaeger1-cassandra-scratch.yaml
# Wait for cassandra to deploy before deploying next next two components
kubectl create -f jaeger2-schema-agents.yaml
kubectl create -f jaeger3-collector-ui.yaml
cd ..
</pre>

### Deploying Prometheus
<pre>
# Save the configuration file in Kubernetes for use by Prometheus
cd configs
kubectl create configmap prometheus --from-file=config.yaml --namespace=kube-system
cd ..

# We need to open up the Prometheus port to access the UI.
# If you are behind a firewall whether it's on-prem or in your
# favorite cloud like GCE, don't forget to open up the NodePort
# that is allocated!
cd services
kubectl create -f prometheus.yaml
cd ..

# Let's deploy Prometheus
# prometheus-scratch.yaml is for non-persistent deployment of Prometheus
# prometheus-simple.yaml is for deploying Prometheus using a pre-created ScaleIO volume named prometheus
# prometheus-dynamic.yaml is for deploying Prometheus using a k8s persistent volume claim
cd deployments
kubectl create -f prometheus-scratch.yaml
OR
kubectl create -f prometheus-simple.yaml
OR
kubectl create -f prometheus-dynamic.yaml
cd ..
</pre>

### Deploying the Sample Application!
<pre>
# We need to open up the Sample Application port to access the Front UI.
# If you are behind a firewall whether it's on-prem or in your
# favorite cloud like GCE, don't forget to open up the NodePort
# that is allocated!
cd services
kubectl create -f backend.yaml
kubectl create -f frontend.yaml
cd ..

# Let's deploy the Sample Application
cd deployments
kubectl create -f backend.yaml
kubectl create -f frontend.yaml
cd ..

# Let's hit the frontend service using curl!
curl http://<PUBLIC_IP_ADDRESS>:<NodePort>
</pre>

## Teardown

Run the following commands to remove the ZOP stack from your Kubernetes cluster.
<pre>
kubectl delete deployment frontend
kubectl delete deployment backend
kubectl delete deployment prometheus --namespace=kube-system
kubectl delete all,daemonset -l jaeger-infra --namespace=kube-system
kubectl delete service frontend
kubectl delete service backend
kubectl delete service prometheus --namespace=kube-system
kubectl delete storageclass sio-small
kubectl delete persistentvolumeclaim pvc-sio-small
kubectl delete persistentvolumeclaim pvc-sio-small --namespace=kube-system
kubectl delete secret sio-secret
kubectl delete secret sio-secret --namespace=kube-system
kubectl delete configmap prometheus --namespace=kube-system
</pre>
