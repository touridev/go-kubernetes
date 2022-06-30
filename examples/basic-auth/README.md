# Support for HTTP Basic Authentication

NGINX supports authenticating requests with [ngx_http_auth_basic_module](https://nginx.org/en/docs/http/ngx_http_auth_basic_module.html).

The Ingress controller provides the following 2 annotations for configuring Basic Auth validation:

* Required: ```nginx.org/basic-auth-secret: "secret"``` -- specifies a Secret resource with a htpasswd user list. The htpasswd must be stored in the `htpasswd` data field. The type of the secret must be `nginx.org/htpasswd`.
* Optional: ```nginx.org/basic-auth-realm: "realm"``` -- specifies a realm.

```

## Example 1: The Same Htpasswd for All Paths

In the following example we enable HTTP Basic authentication for the cafe-ingress Ingress for all paths using the same htpasswd `cafe-htpasswd`:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cafe-ingress
  annotations:
    nginx.org/basic-auth-secret: "cafe-passwd"
    nginx.org/basic-auth-realm: "Cafe App"
spec:
  tls:
  - hosts:
    - cafe.example.com
    secretName: cafe-secret
  rules:
  - host: cafe.example.com
    http:
      paths:
      - path: /tea
        backend:
          service:
            name: tea-svc
            port:
              number: 80
      - path: /coffee
        backend:
          service:
            name: coffee-svc
            port:
              number: 80
```
* The keys must be deployed separately in the Secret `cafe-jwk`.
* The realm is  `Cafe App`.

## Example 2: a Separate Htpasswd Per Path

In the following example we enable Basic Auth validation for the [mergeable Ingresses](../mergeable-ingress-types) with a separate Basic Auth user:password list per path:

* Master:
  ```yaml
  apiVersion: networking.k8s.io/v1
  kind: Ingress
  metadata:
    name: cafe-ingress-master
    annotations:
      kubernetes.io/ingress.class: "nginx"
      nginx.org/mergeable-ingress-type: "master"
  spec:
    tls:
    - hosts:
      - cafe.example.com
      secretName: cafe-secret
    rules:
    - host: cafe.example.com
  ```

* Tea minion:
  ```yaml
  apiVersion: networking.k8s.io/v1
  kind: Ingress
  metadata:
    name: cafe-ingress-tea-minion
    annotations:
      nginx.org/mergeable-ingress-type: "minion"
      nginx.org/basic-auth-secret: "tea-passwd"
      nginx.org/basic-auth-realm: "Tea"
  spec:
    rules:
    - host: cafe.example.com
      http:
        paths:
        - path: /tea
          pathType: Prefix
          backend:
            service:
              name: tea-svc
              port:
                number: 80
  ```

* Coffee minion:
  ```yaml
  apiVersion: networking.k8s.io/v1
  kind: Ingress
  metadata:
    name: cafe-ingress-coffee-minion
    annotations:
      nginx.org/mergeable-ingress-type: "minion"
      nginx.org/basic-auth-secret: "coffee-passwd"
      nginx.org/basic-auth-realm: "Coffee"
  spec:
    rules:
    - host: cafe.example.com
      http:
        paths:
        - path: /coffee
          pathType: Prefix
          backend:
            service:
              name: coffee-svc
              port:
                number: 80
  ```
