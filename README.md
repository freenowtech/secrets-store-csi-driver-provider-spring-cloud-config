secrets-store-csi-driver-provider-spring-cloud-config
=====================================================

The Spring Cloud Config provider for Secrets Store CSI driver allows you to get content stored in Spring Cloud Config and use the Secrets Store CSI driver interface to mount them into a Kubernetes pods.

Usage
-----

### Install the Secrets Store CSI Driver

---

**NOTE**

To install the Secrets Store CSI driver you need Kubernetes 1.15.x with the CSIInlineVolume feature gate or a 1.16.x+ cluster.

---

Make sure you have followed the Installation guide for the [Secrets Store CSI Driver](https://github.com/deislabs/secrets-store-csi-driver#usage).

Create a `SecretProviderClass` resource to provide Spring-Cloud-Config-specific parameters for the Secrets Store CSI driver.

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
