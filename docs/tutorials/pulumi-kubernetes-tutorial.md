# Deploy to Kubernetes with Pulumi and Kukicha

**Level:** Intermediate
**Time:** 15 minutes
**Prerequisites:** [Kukicha installed](../../README.md), [Pulumi CLI](https://www.pulumi.com/docs/install/), a running Kubernetes cluster (minikube, Docker Desktop, or cloud)

You'll deploy an nginx web server to Kubernetes using Pulumi — entirely in Kukicha. Along the way you'll see how `onerr`, `reference of`, if-expressions, and lambdas cut the boilerplate that makes Go-based Pulumi programs hard to read.

## What You'll Build

A Kubernetes **Deployment** running nginx, fronted by a **Service** that gives it a reachable IP address. Three resources, one file, zero YAML.

---

## Step 0: Project Setup

Pulumi scaffolds a Go project. Since Kukicha is a strict superset of Go, we just rename the file:

```bash
mkdir quickstart && cd quickstart
pulumi new kubernetes-go      # accept the defaults
mv main.go main.kuki          # Kukicha takes over from here
```

> **Why does this work?** Kukicha compiles to Go. The `go.mod`, `go.sum`, and
> `Pulumi.yaml` that `pulumi new` generated are unchanged — Kukicha slots right
> in.

---

## Step 1: Deploy nginx

Replace the contents of `main.kuki` with:

```kukicha
import "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1" as appsv1
import "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1" as corev1
import "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1" as metav1
import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

function main()
    pulumi.Run(function(ctx reference pulumi.Context) error
        labels := pulumi.StringMap{"app": pulumi.String("nginx")}

        deployment := appsv1.NewDeployment(ctx, "nginx", reference of appsv1.DeploymentArgs{
            Spec: appsv1.DeploymentSpecArgs{
                Selector: reference of metav1.LabelSelectorArgs{
                    MatchLabels: labels,
                },
                Replicas: pulumi.Int(1),
                Template: reference of corev1.PodTemplateSpecArgs{
                    Metadata: reference of metav1.ObjectMetaArgs{Labels: labels},
                    Spec: reference of corev1.PodSpecArgs{
                        Containers: corev1.ContainerArray{
                            corev1.ContainerArgs{
                                Name:  pulumi.String("nginx"),
                                Image: pulumi.String("nginx"),
                            },
                        },
                    },
                },
            },
        }) onerr return

        ctx.Export("name", deployment.Metadata.Name())
        return empty
    )
```

Now deploy it:

```bash
pulumi up        # preview → select "yes"
```

Verify:

```bash
pulumi stack output name
# nginx-abc1234
```

### What changed from Go?

| Go | Kukicha |
|----|---------|
| `if err != nil { return err }` | `onerr return` — one line, same behavior |
| `&appsv1.DeploymentArgs{...}` | `reference of appsv1.DeploymentArgs{...}` — reads like English |
| `return nil` | `return empty` |
| Curly braces everywhere | 4-space indentation — the nesting is already there, braces just add noise |

> **Tip:** Every `reference of` above replaces Go's `&` operator. In deeply
> nested Pulumi structs the readability gain adds up fast.

---

## Step 2: Expose It with a Service

Add the `config` import and expand the program to create a Service. Here is the full updated file:

```kukicha
import "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1" as appsv1
import "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1" as corev1
import "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1" as metav1
import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
import "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"

function main()
    pulumi.Run(function(ctx reference pulumi.Context) error
        isMinikube := config.GetBool(ctx, "isMinikube")
        labels := pulumi.StringMap{"app": pulumi.String("nginx")}

        # --- Deployment (same as Step 1) ---

        deployment := appsv1.NewDeployment(ctx, "nginx", reference of appsv1.DeploymentArgs{
            Spec: appsv1.DeploymentSpecArgs{
                Selector: reference of metav1.LabelSelectorArgs{
                    MatchLabels: labels,
                },
                Replicas: pulumi.Int(1),
                Template: reference of corev1.PodTemplateSpecArgs{
                    Metadata: reference of metav1.ObjectMetaArgs{Labels: labels},
                    Spec: reference of corev1.PodSpecArgs{
                        Containers: corev1.ContainerArray{
                            corev1.ContainerArgs{
                                Name:  pulumi.String("nginx"),
                                Image: pulumi.String("nginx"),
                            },
                        },
                    },
                },
            },
        }) onerr return

        # --- Service ---

        serviceType := if isMinikube then "ClusterIP" else "LoadBalancer"

        service := corev1.NewService(ctx, "nginx", reference of corev1.ServiceArgs{
            Spec: reference of corev1.ServiceSpecArgs{
                Type:     pulumi.String(serviceType),
                Selector: labels,
                Ports: reference of corev1.ServicePortArray{
                    reference of corev1.ServicePortArgs{
                        Port:       pulumi.Int(80),
                        TargetPort: pulumi.Int(80),
                        Protocol:   pulumi.String("TCP"),
                    },
                },
            },
        }) onerr return

        # --- Export the IP ---

        ip := service.Status.ApplyT(function(status reference corev1.ServiceStatus) string
            if status.LoadBalancer.Ingress isnt empty and len(status.LoadBalancer.Ingress) > 0
                ingress := status.LoadBalancer.Ingress[0]
                if ingress.Hostname isnt empty
                    return dereference ingress.Hostname
                if ingress.Ip isnt empty
                    return dereference ingress.Ip
            return ""
        ).(pulumi.StringOutput)

        ctx.Export("name", deployment.Metadata.Name())
        ctx.Export("ip", ip)
        return empty
    )
```

If you're on minikube, set the config flag first:

```bash
pulumi config set isMinikube true    # skip this for cloud clusters
```

Deploy and test:

```bash
pulumi up
curl $(pulumi stack output ip)       # "Welcome to nginx!"
```

### New Kukicha features in this step

**If-expression** — Kukicha's ternary replaces a 4-line `if/else` block:

```kukicha
# Kukicha
serviceType := if isMinikube then "ClusterIP" else "LoadBalancer"

# Go equivalent
feType := "LoadBalancer"
if isMinikube {
    feType = "ClusterIP"
}
```

**Readable operators** — `isnt empty` and `and` replace `!= nil` and `&&`:

```kukicha
# Kukicha
if status.LoadBalancer.Ingress isnt empty and len(status.LoadBalancer.Ingress) > 0

# Go equivalent
if status.LoadBalancer.Ingress != nil && len(status.LoadBalancer.Ingress) > 0
```

**`dereference`** — reads the value behind a pointer, replacing Go's `*`:

```kukicha
return dereference ingress.Hostname   # Go: return *ingress.Hostname
```

---

## Step 3: Clean Up

Remove all cloud resources and delete the stack:

```bash
pulumi destroy       # preview → select "yes"
pulumi stack rm      # permanent — removes all state
```

---

## Cheat Sheet

Everything used in this tutorial, at a glance:

| Go pattern | Kukicha equivalent |
|---|---|
| `if err != nil { return err }` | `onerr return` |
| `&Struct{...}` | `reference of Struct{...}` |
| `*ptr` | `dereference ptr` |
| `nil` | `empty` |
| `!= nil` | `isnt empty` |
| `&&` | `and` |
| `{ }` braces | 4-space indentation |
| No ternary | `if COND then X else Y` |
| `func(x T) R { return expr }` | `(x T) => expr` |
| `func` | `function` (alias — both work) |

---

## Next Steps

- [Pulumi Kubernetes API docs](https://www.pulumi.com/registry/packages/kubernetes/) — every resource you can create
- [Kukicha language reference](../SKILL.md) — pipes, enums, onerr, and more
- [Web App Tutorial](web-app-tutorial.md) — build an HTTP service in Kukicha
