package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type rootArgs struct {
	yes                bool
	gracePeriod        time.Duration
	timeout            time.Duration
	whitelistResources *[]string
}

func main() {
	err := rootCmd.Execute()
	must(err)
}

func init() {
	apiServer := ""
	kubeconfigArgs.APIServer = &apiServer
	kubeconfigArgs.AddFlags(rootCmd.PersistentFlags())

	rest.SetDefaultWarningHandler(rest.NewWarningWriter(io.Discard, rest.WarningWriterOptions{}))
	set := &flag.FlagSet{}
	klog.InitFlags(set)
	rootCmd.PersistentFlags().AddGoFlagSet(set)

	rootCmd.Flags().BoolVarP(&rootFlags.yes, "yes", "", rootFlags.yes, "Force remove all finalizers")
	rootCmd.Flags().DurationVarP(&rootFlags.timeout, "timeout", "", time.Second*120, "Await timout before fail the command")
	rootCmd.Flags().DurationVarP(&rootFlags.gracePeriod, "grace-period", "", time.Second*10, "Force remove all finalizers after the grace period was reached")
	rootCmd.Flags().StringSliceVarP(rootFlags.whitelistResources, "resources", "", nil, "Whitelist resources from which the finalizers are removed. If not set all resources are targeted.")
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

var (
	rootFlags = rootArgs{
		whitelistResources: &[]string{},
	}
	kubeconfigArgs = genericclioptions.NewConfigFlags(false)
	rootCmd        = &cobra.Command{
		Use:   "kubectl del-namespace [namespaces]",
		Short: "Force delete namespace",
		Long:  `Force delete kubernetes namespace(s) including all resources with blocking finalizers`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("expected at least one namespace")
			}

			ctx, cancel := context.WithTimeout(context.TODO(), rootFlags.timeout)
			defer cancel()

			clusterDiscoveryClient, err := getDiscoveryClient(kubeconfigArgs)
			if err != nil {
				return err
			}

			clusterDynClient, err := getDynClient(kubeconfigArgs)
			if err != nil {
				return err
			}

			resources, err := gatherResourceGroups(ctx, clusterDiscoveryClient)
			if err != nil {
				return err
			}

			g, ctx := errgroup.WithContext(ctx)

			for _, ns := range args {
				ns := ns
				g.Go(func() error {
					_, err = clusterDynClient.Resource(schema.GroupVersionResource{
						Version:  "v1",
						Resource: "namespaces",
					}).Get(ctx, ns, metav1.GetOptions{})

					if err != nil {
						return err
					}

					if rootFlags.yes {
						watcher, err := clusterDynClient.Resource(schema.GroupVersionResource{
							Version:  "v1",
							Resource: "namespaces",
						}).Watch(ctx, metav1.ListOptions{
							LabelSelector: fmt.Sprintf("kubernetes.io/metadata.name=%s", ns),
						})

						if err != nil {
							return err
						}

						defer watcher.Stop()

						err = clusterDynClient.Resource(schema.GroupVersionResource{
							Version:  "v1",
							Resource: "namespaces",
						}).Delete(ctx, ns, metav1.DeleteOptions{})

						if err != nil {
							return err
						}

						gracePeriod := time.After(rootFlags.gracePeriod)

					AWAIT:
						for {
							select {
							case <-ctx.Done():
								return nil
							case <-gracePeriod:
								klog.Infof("grace period reached, removing finalizers")
								break AWAIT
							case event := <-watcher.ResultChan():
								if event.Object.(*unstructured.Unstructured).Object["metadata"].(map[string]interface{})["name"] != ns {
									continue
								}

								if event.Type == watch.Deleted {
									return nil
								}

								if event.Type == watch.Modified && event.Object.(*unstructured.Unstructured).Object["status"].(map[string]interface{})["phase"] == "Terminating" {
									klog.Infof("deleting namespace, awaiting grace period")
								}
							}
						}
					}

					err = cleanupFinalizers(ctx, g, resources, clusterDynClient, ns)
					if err != nil {
						return err
					}

					return nil
				})
			}
			return g.Wait()
		},
	}
)

func validateResource(gv schema.GroupVersion, resource metav1.APIResource) (schema.GroupVersionResource, error) {
	if !resource.Namespaced {
		return schema.GroupVersionResource{}, errors.New("expected namespaced resource")
	}

	gvr := schema.GroupVersionResource{
		Group:    gv.Group,
		Version:  gv.Version,
		Resource: resource.Name,
	}

	if !slices.Contains(resource.Verbs, "list") {
		return schema.GroupVersionResource{}, errors.New("expected listable resource")
	}

	return gvr, nil
}

type resourceDefinition struct {
	gvr      schema.GroupVersionResource
	resource metav1.APIResource
}

func gatherResourceGroups(ctx context.Context, clusterDiscoveryClient *discovery.DiscoveryClient) ([]resourceDefinition, error) {
	var resources []resourceDefinition

	_, list, err := clusterDiscoveryClient.ServerGroupsAndResources()
	if err != nil {
		return resources, err
	}

	for _, group := range list {
		klog.V(1).Infof("discover resource group %#v", group.GroupVersion)
		gv, err := schema.ParseGroupVersion(group.GroupVersion)
		if err != nil {
			return resources, err
		}

		for _, resource := range group.APIResources {
			resource := resource

			klog.V(1).Infof("discover resource %s.%s.%s", resource.Name, gv.Group, gv.Version)

			gvr, err := validateResource(gv, resource)
			if err != nil {
				klog.V(1).Infof("skipping resource: %s", err.Error())
				continue
			}

			resources = append(resources, resourceDefinition{
				gvr:      gvr,
				resource: resource,
			})
		}
	}

	return resources, nil
}

func cleanupFinalizers(ctx context.Context, g *errgroup.Group, resources []resourceDefinition, clusterDynClient dynamic.Interface, ns string) error {
	for _, resource := range resources {
		resource := resource

		resAPI := clusterDynClient.Resource(resource.gvr).Namespace(ns)

		g.Go(func() error {
			clean := func() error {
				list, err := resAPI.List(ctx, metav1.ListOptions{})

				if err != nil {
					return err
				}

				for _, res := range list.Items {
					//not really required but acts as an additional gate
					if res.GetNamespace() != ns {
						continue
					}

					if len(res.GetFinalizers()) > 0 {
						resourceName := fmt.Sprintf("%s.%s", resource.resource.Name, resource.gvr.Group)
						if resource.gvr.Group == "" {
							resourceName = resource.resource.Name
						}

						klog.Infof("resource has finalizers: %s.%s [%s] => %#v", res.GetName(), res.GetNamespace(), resourceName, res.GetFinalizers())

						if len(*rootFlags.whitelistResources) > 0 && !slices.Contains(*rootFlags.whitelistResources, resourceName) {
							klog.Infof("resource [%s] not whitelisted", resourceName)
							continue
						}

						if rootFlags.yes {
							res.SetFinalizers(nil)
							_, err = resAPI.Update(ctx, &res, metav1.UpdateOptions{})
							if err != nil {
								klog.Errorf("failed to remove finalizer, backoff: %s", err.Error())
								return err
							}
						}
					}
				}

				return nil
			}

			return backoff.Retry(clean, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))
		})
	}

	return nil
}

func getDiscoveryClient(kubeconfigArgs *genericclioptions.ConfigFlags) (*discovery.DiscoveryClient, error) {
	cfg, err := kubeconfigArgs.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	cfg.WarningHandler = rest.NoWarnings{}

	client, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func getDynClient(kubeconfigArgs *genericclioptions.ConfigFlags) (dynamic.Interface, error) {
	cfg, err := kubeconfigArgs.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	client, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return client, nil
}
