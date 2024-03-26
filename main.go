package main

import (
	"context"
	"errors"
	"flag"
	"io"

	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type rootArgs struct {
	yes bool
}

var (
	rootFlags      = rootArgs{}
	kubeconfigArgs = genericclioptions.NewConfigFlags(false)
	rootCmd        = &cobra.Command{
		Use:   "kubectl del-namespace [namespace]",
		Short: "Force delete namespace",
		Long:  `Force delete a kubernetes namespace including all resources with blocking finalizers`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("expected exactly one namespace")
			}

			ctx := context.TODO()
			clusterDiscoveryClient, err := getDiscoveryClient(kubeconfigArgs)
			if err != nil {
				return err
			}

			clusterDynClient, err := getDynClient(kubeconfigArgs)
			if err != nil {
				return err
			}

			_, list, err := clusterDiscoveryClient.ServerGroupsAndResources()
			if err != nil {
				return err
			}

			g, ctx := errgroup.WithContext(ctx)

			if rootFlags.yes {
				err = clusterDynClient.Resource(schema.GroupVersionResource{
					Version:  "v1",
					Resource: "namespaces",
				}).Delete(ctx, args[0], metav1.DeleteOptions{})

				if err != nil {
					return err
				}
			}

			for _, group := range list {
				klog.V(1).Infof("discover resource group %#v", group.GroupVersion)
				gv, err := schema.ParseGroupVersion(group.GroupVersion)
				if err != nil {
					return err
				}

				for _, resource := range group.APIResources {
					resource := resource
					klog.V(1).Infof("discover resource %#v.%#v.%#v", resource.Name, resource.Group, resource.Version)

					gvr, err := validateResource(args[0], gv, resource)
					if err != nil {
						klog.V(1).Infof("skipping resource: %s", err.Error())
						continue
					}

					resAPI := clusterDynClient.Resource(gvr).Namespace(args[0])

					g.Go(func() error {
						list, err := resAPI.List(ctx, metav1.ListOptions{})

						if err != nil {
							return err
						}

						for _, res := range list.Items {
							//not really required but acts as an additional gate
							if res.GetNamespace() != args[0] {
								continue
							}

							if len(res.GetFinalizers()) > 0 {
								klog.Infof("resource has finalizers: %s [%s.%s.%s] => %#v", res.GetName(), resource.Name, resource.Version, resource.Group, res.GetFinalizers())

								if rootFlags.yes {
									res.SetFinalizers(nil)
									_, err = resAPI.Update(ctx, &res, metav1.UpdateOptions{})
									if err != nil {
										return err
									}
								}
							}
						}

						return nil
					})
				}
			}

			return g.Wait()
		},
	}
)

func validateResource(ns string, gv schema.GroupVersion, resource metav1.APIResource) (schema.GroupVersionResource, error) {
	if ns != "" && !resource.Namespaced {
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
}

func must(err error) {
	if err != nil {
		panic(err)
	}
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
