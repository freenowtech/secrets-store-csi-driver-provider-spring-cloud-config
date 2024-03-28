# secrets-store-csi-driver-provider-spring-cloud-config

The Spring Cloud Config provider for Secrets Store CSI driver allows you to get content stored in Spring Cloud Config and use the Secrets Store CSI driver interface to mount them into a Kubernetes pods.

## Installation

### Requirements

- A running instance of [Spring Cloud Config Server](https://docs.spring.io/spring-cloud-config/docs/current/reference/html/)
- [Secrets Store CSI Driver installed](https://secrets-store-csi-driver.sigs.k8s.io/getting-started/installation.html)

### Installing the provider

To install the provider, use the YAML file in the deployment directory:

```shell
kubectl apply -f https://raw.githubusercontent.com/freenowtech/secrets-store-csi-driver-provider-spring-cloud-config/master/deployment/provider.yaml
```

## Usage

Create a `SecretProviderClass` resource to provide Spring-Cloud-Config-specific parameters for the Secrets Store CSI driver.

```yaml
apiVersion: secrets-store.csi.x-k8s.io/v1alpha1
kind: SecretProviderClass
metadata:
  name: spring-cloud-config-example
spec:
  provider: spring-cloud-config
  parameters:
    serverAddress: "http://configserver.example" # this url should point to config server
    application: "myapp" # the application you're retrieving the config for
    profile: "prod" # the profile for your application to pull
    fileName: "application.yaml" # the name of the file to create - supports extensions .yaml, .yml, .json and .properties
```

Afterwards you can reference your `SecretProviderClass` in your Pod Definition

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: secrets-store-example
spec:
  containers:
  - image: ubuntu:latest
    name: ubuntu
    command: ["/bin/bash"]
    args:
      - "-c"
      - "cat /secrets-store/application.yaml && sleep 300"
    volumeMounts:
    - name: secrets-store-inline
      mountPath: "/secrets-store"
      readOnly: true
  volumes:
    - name: secrets-store-inline
      csi:
        driver: secrets-store.csi.k8s.com
        readOnly: true
        volumeAttributes:
          secretProviderClass: "spring-cloud-config-example"
```
