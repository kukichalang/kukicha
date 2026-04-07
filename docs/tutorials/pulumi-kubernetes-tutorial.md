# Deploy to Kubernetes with Pulumi and Kukicha

**Level:** Intermediate
**Time:** 15 minutes
**Prerequisites:** [Kukicha installed](../../README.md), [Pulumi CLI](https://www.pulumi.com/docs/install/), a running Kubernetes cluster (minikube, Docker Desktop, or cloud)

You'll deploy an nginx web server to Kubernetes using Pulumi — entirely in Kukicha. The Go Pulumi SDK is famously verbose (deeply nested structs, `pulumi.String()` wrappers everywhere). We'll tame it by splitting resource definitions into helpers and wiring them together with **pipes**, **onerr**, and **if-expressions**.

## What You'll Build

A Kubernetes **Deployment** running nginx, fronted by a **Service** that gives it a reachable IP address. Two files, zero YAML.

---

## Step 0: Project Setup

Pulumi scaffolds a Go project. Since Kukicha compiles to Go, we just rename the file and create a second one for our resource helpers:

```bash
mkdir quickstart && cd quickstart
pulumi new kubernetes-go          # accept the defaults
mv main.go main.kuki              # entry point
touch resources.kuki              # resource helpers
```

> **Why does this work?** `kukicha build quickstart/` merges all `.kuki` files
> in a directory into a single Go package and compiles it. The `go.mod`,
> `go.sum`, and `Pulumi.yaml` that `pulumi new` generated stay unchanged.

---

## Step 1: Define the Resources

These helpers wrap the verbose Pulumi structs. You write them once and never look at them again. Put this in `resources.kuki`:

```kukicha
import "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1" as appsv1
import "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1" as corev1
import "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1" as metav1
import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

function NewNginxDeployment(ctx reference pulumi.Context, name string, labels pulumi.StringMap) (reference appsv1.Deployment, error)
    return appsv1.NewDeployment(ctx, name, reference of appsv1.DeploymentArgs{
        Spec: appsv1.DeploymentSpecArgs{
            Selector: reference of metav1.LabelSelectorArgs{MatchLabels: labels},
            Replicas: pulumi.Int(1),
            Template: reference of corev1.PodTemplateSpecArgs{
                Metadata: reference of metav1.ObjectMetaArgs{Labels: labels},
                Spec: reference of corev1.PodSpecArgs{
                    Containers: corev1.ContainerArray{
                        corev1.ContainerArgs{
                            Name:  pulumi.String(name),
                            Image: pulumi.String("nginx"),
                        },
                    },
                },
            },
        },
    })

function NewNginxService(ctx reference pulumi.Context, name string, labels pulumi.StringMap, serviceType string) (reference corev1.Service, error)
    return corev1.NewService(ctx, name, reference of corev1.ServiceArgs{
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
    })

function ServiceIP(svc reference corev1.Service) pulumi.StringOutput
    return svc.Status.ApplyT(function(status reference corev1.ServiceStatus) string
        if status.LoadBalancer.Ingress isnt empty and len(status.LoadBalancer.Ingress) > 0
            ingress := status.LoadBalancer.Ingress[0]
            if ingress.Hostname isnt empty
                return dereference ingress.Hostname
            if ingress.Ip isnt empty
                return dereference ingress.Ip
        return ""
    ).(pulumi.StringOutput)
```

Verbose? Yes — that's the Pulumi Go SDK. But now it's **contained**. Your main program never sees it.

---

## Step 2: Wire It Together with Pipes

Here's `main.kuki` — the part you'll actually read and maintain:

```kukicha
import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
import "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"

function main()
    pulumi.Run(function(ctx reference pulumi.Context) error
        serviceType := if config.GetBool(ctx, "isMinikube") then "ClusterIP" else "LoadBalancer"
        labels := pulumi.StringMap{"app": pulumi.String("nginx")}

        dep := NewNginxDeployment(ctx, "nginx", labels) onerr return
        svc := NewNginxService(ctx, "nginx", labels, serviceType) onerr return

        dep.Metadata.Name() |> ctx.Export("name", _)
        svc |> ServiceIP() |> ctx.Export("ip", _)
        return empty
    )
```

That's **12 lines** of logic. Compare that to the 60+ line single-function Go version from the Pulumi tutorial.

### What's happening here?

**Pipes (`|>`)** — chain values through functions, left to right:

```kukicha
svc |> ServiceIP() |> ctx.Export("ip", _)

# Reads as: take the service → extract its IP → export it as "ip"
# Without pipes: ctx.Export("ip", ServiceIP(svc))
```

The `_` placeholder tells the pipe where to insert the value when it's not the last argument — `ctx.Export("ip", _)` becomes `ctx.Export("ip", <piped value>)`.

**If-expression** — a one-line ternary:

```kukicha
serviceType := if config.GetBool(ctx, "isMinikube") then "ClusterIP" else "LoadBalancer"
```

**`onerr return`** — propagates errors in one line instead of three:

```kukicha
dep := NewNginxDeployment(ctx, "nginx", labels) onerr return

# Go equivalent:
# dep, err := NewNginxDeployment(ctx, "nginx", labels)
# if err != nil {
#     return err
# }
```

---

## Step 3: Deploy

```bash
pulumi config set isMinikube true    # skip this for cloud clusters
pulumi up                            # preview → select "yes"
```

Test it:

```bash
curl $(pulumi stack output ip)       # "Welcome to nginx!"
```

---

## Step 4: Clean Up

```bash
pulumi destroy       # preview → select "yes"
pulumi stack rm      # permanent — removes all state
```

---

## Kukicha vs Go vs HCL

The Pulumi Go SDK is verbose by nature — `pulumi.String()`, nested `Args` structs, and type assertions are the price of Go's type system. Kukicha can't remove those, but it can keep them out of your main logic:

| | Lines of main logic | Boilerplate visible? |
|---|---|---|
| **Go (Pulumi tutorial)** | ~65 in one function | Yes — structs, `if err`, braces everywhere |
| **Kukicha (this tutorial)** | ~12 in `main.kuki` | No — hidden in `resources.kuki` helpers |
| **HCL** | ~20 | No — declarative by design |

The helpers in `resources.kuki` are a one-time cost. As you add more resources, `main.kuki` stays clean — each resource is one `onerr return` line, each export is one pipe.

---

## Cheat Sheet

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
| `f(b, a)` | `a \|> f(b, _)` |
| `func` | `function` (alias — both work) |

---

## Next Steps

- [Pulumi Kubernetes API docs](https://www.pulumi.com/registry/packages/kubernetes/) — every resource you can create
- [Kukicha language reference](../SKILL.md) — pipes, enums, onerr, and more
- [Web App Tutorial](web-app-tutorial.md) — build an HTTP service in Kukicha
