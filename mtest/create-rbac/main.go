package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type client struct {
	clientSet *kubernetes.Clientset
}

const (
	roleName    = "node-reader"
	bingingName = "read-node"
)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	c := client{clientSet: clientSet}

	_, err = c.clientSet.RbacV1().ClusterRoles().Get(roleName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		fmt.Println("cluster-role not found, creating...")
		_, err = c.createClusterRole()
		if err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	}

	_, err = c.clientSet.RbacV1().ClusterRoleBindings().Get(bingingName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		fmt.Println("cluster-role-binding not found, creating...")
		_, err = c.createClusterRoleBinding("taro")
		if err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	}

}

func (c *client) createClusterRole() (*v1.ClusterRole, error) {
	return c.clientSet.RbacV1().ClusterRoles().Create(&v1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   roleName,
			Labels: map[string]string{"kubernetes.io/bootstrapping": "rbac-defaults"},
		},
		Rules: []v1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	})
}

func (c *client) createClusterRoleBinding(user string) (*v1.ClusterRoleBinding, error) {
	return c.clientSet.RbacV1().ClusterRoleBindings().Create(&v1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   bingingName,
			Labels: map[string]string{"kubernetes.io/bootstrapping": "rbac-defaults"},
		},
		Subjects: []v1.Subject{
			{
				Kind: "User",
				Name: user,
			},
		},
		RoleRef: v1.RoleRef{
			Kind:     "ClusterRole",
			Name:     roleName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	})
}
