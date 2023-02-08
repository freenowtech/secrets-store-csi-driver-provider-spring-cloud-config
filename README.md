secrets-store-csi-driver-provider-spring-cloud-config
=====================================================

The Spring Cloud Config provider
for [Secrets Store CSI driver](https://github.com/kubernetes-sigs/secrets-store-csi-driver) allows you to get content
stored in Spring Cloud Config
and use the Secrets Store CSI driver interface to mount them into a Kubernetes pods.

Usage
-----

### Install the Secrets Store CSI Driver

Make sure you have followed the Installation guide for
the [Secrets Store CSI Driver](https://secrets-store-csi-driver.sigs.k8s.io/getting-started/installation.html).

Create a `SecretProviderClass` resource to provide Spring-Cloud-Config-specific parameters for the Secrets Store CSI
driver.

```yaml
apiVersion: secrets-store.csi.x-k8s.io/v1alpha1
kind: SecretProviderClass
metadata:
  name: spring-cloud-config-<your-application>
spec:
  provider: spring-cloud-config
  parameters:
    serverAddress: "<your-server-address>" # this url should point config server
    application: "<your-application>" # the application you're retrieving the config for
    profile: "<your-profile>" # the profile for your application to pull
    fileType: "json" # json or properties viable

```

Afterwards you can reference your `SecretProviderClass` in your Pod Definition

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: nginx-secrets-store-inline
spec:
  containers:
    - image: nginx
      name: nginx
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
          secretProviderClass: "spring-cloud-config-<your-application>"
```

Development
-----

### Local Testing

To test the secrets store, a local kubernetes should be created for example via minikube.
After cluster startup you can install the secrets-store-csi-driver and add the spring cloud config secret provider.

```shell
minikube start --kubernetes-version=<cluster-version>
cd <path-to-secrets-store-csi-driver-repo>
kustomize build <kustomize-dir> | kubectl apply --validate=true -f -
```

Afterwards the new image needs to be build, replacing the currently running provider.

```shell
make package
kubectl edit <secrets-store-provider-daemonset> 
kubectl logs <secrets-store-registrator-pod> # to verify whether the new provider was rolled out and registered
```

Now a new `SecretProviderClass` resource should be created to test whether both the secrets-store and the provider work
as expected
You can use the example from the [installation](#Install the Secrets Store CSI Driver).
If everything went as expected the secret file should exist inside the pod.
If not, the secrets-store logs should be checked.