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

Afterward, reference your `SecretProviderClass` in your Pod Definition

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

## Development

### Run the binary locally

#### Requirements

- [Go](https://go.dev/doc/install)
- [gRPCurl](https://github.com/fullstorydev/grpcurl?tab=readme-ov-file#installation)


#### Steps to execute

1. Build the binary:
   ```shell
   go build
   ```
1. Start the binary:
   ```shell
   ./secrets-store-csi-driver-provider-spring-cloud-config
   ```
1. In a new terminal window, create the directory `.dev`:
   ```shell
   mkdir -p .dev
   ```
1. Download the grpc protobuf definitions:
   ```shell
   curl -L -o .dev/service.proto https://raw.githubusercontent.com/kubernetes-sigs/secrets-store-csi-driver/main/provider/v1alpha1/service.proto
   ```
1. Create the payload `.dev/mount.json`:
   ```json
   {
     "attributes": "{\"serverAddress\":\"<your-server-address>\",\"application\":\"<your application>\",\"profile\":\"<your profile>\",\"fileName\":\"application.yaml\"}",
     "secrets": "{}",
     "targetPath": "./.dev",
     "permission": "420"
   }
   ```
   **Make sure to replace the placeholders**
1. Send the payload to the provider:
   ```shell
   cat ./.dev/mount.json | grpcurl -unix -plaintext -proto ./.dev/service.proto -d @ ./spring-cloud-config.sock v1alpha1.CSIDriverProvider/Mount
   ```
1. Verify that the file has been created:
   ```shell
   cat ./.dev/application.yaml
   # Should display YAML content
   ```


## Release

Follow these steps to release a new version:

1. Create a new release [via the GitHub UI](https://github.com/freenowtech/secrets-store-csi-driver-provider-spring-cloud-config/releases/new).
2. Set `v0.x.y` as the tag and the release title.
   
   If the release contains at least one feature, increase `x` by one and set `y` to `0`.
   
   If the release contains bug fixes only, increase `y` by one.
3. Let GitHub generate the release notes by clicking the "Generate release notes" button.
4. Click the "Publish release" button.
