/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"

	appsv1 "k8s.io/api/apps/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1 "elastic-web/api/v1"
)

const (
	// deployment中的APP标签名
	APP_NAME = "elastic-web"
	// tomcat容器的端口号
	CONTAINER_PORT = 80
	// 单个POD的CPU资源申请
	CPU_REQUEST = "100m"
	// 单个POD的CPU资源上限
	CPU_LIMIT = "400m"
	// 单个POD的内存资源申请
	MEM_REQUEST = "256Mi"
	// 单个POD的内存资源上限
	MEM_LIMIT = "512Mi"
)

// ElasticWebReconciler reconciles a ElasticWeb object
type ElasticWebReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=k8s.com.13sai,resources=elasticwebs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.com.13sai,resources=elasticwebs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.com.13sai,resources=elasticwebs/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ElasticWeb object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *ElasticWebReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logrus.Info("Reconcile")

	instance := &k8sv1.ElasticWeb{}

	err := r.Get(ctx, req.NamespacedName, instance)

	if err != nil {
		logrus.Info("reconcile err", err)
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		dp := &appsv1.Deployment{}

		err := r.Get(ctx, req.NamespacedName, dp)
		if err != nil {

		}

		return reconcile.Result{}, nil
	}

	dp := &appsv1.Deployment{}

	if err = r.Get(ctx, req.NamespacedName, dp); err != nil {
		if errors.IsNotFound(err) {
			logrus.Info("deployment not exists")

			if *(instance.Spec.TotalQPS) < 1 {
				return ctrl.Result{}, nil
			}

			if err := createServiceIfNotExists(ctx, r, instance, req); err != nil {
				return ctrl.Result{}, nil
			}

			if err = createDeployment(ctx, r, instance); err != nil {
				return ctrl.Result{}, nil
			}

			if err = updateStatus(ctx, r, instance); err != nil {
				logrus.Errorf("updateStatus err=%v", err)
				return ctrl.Result{}, nil
			}

			return ctrl.Result{}, nil
		} else {
			logrus.Errorf("err=%v", err)
			return ctrl.Result{}, nil
		}
	}

	expectReplicas := getExpectReplicas(instance)

	realReplicas := *dp.Spec.Replicas

	logrus.Infof("expectReplicas [%d], realReplicas [%d]", expectReplicas, realReplicas)

	if err = r.Update(ctx, dp); err != nil {
		logrus.Errorf("Update err=%v", err)
		return ctrl.Result{}, nil
	}

	if err = updateStatus(ctx, r, instance); err != nil {
		logrus.Errorf("updateStatus err=%v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ElasticWebReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1.ElasticWeb{}).
		Complete(r)
}

func getExpectReplicas(web *k8sv1.ElasticWeb) int {
	singleQPS := *(web.Spec.SingleQPS)
	totalQPS := *web.Spec.TotalQPS
	replicas := totalQPS / singleQPS
	if totalQPS%singleQPS > 0 {
		replicas++
	}
	return replicas
}

func createServiceIfNotExists(ctx context.Context, r *ElasticWebReconciler, web *k8sv1.ElasticWeb, req ctrl.Request) error {
	logrus.Info("createService")
	service := &corev1.Service{}
	err := r.Get(ctx, req.NamespacedName, service)
	if err != nil {
		return nil
	}

	if !errors.IsNotFound(err) {
		return err
	}

	service = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: web.Namespace,
			Name:      web.Name,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:     "http",
				Port:     80,
				NodePort: int32(*web.Spec.Port),
			}},
			Selector: map[string]string{
				"app": APP_NAME,
			},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	// 这一步非常关键！
	// 建立关联后，删除elasticweb资源时就会将deployment也删除掉
	if err := controllerutil.SetControllerReference(web, service, r.Scheme); err != nil {
		return err
	}

	if err := r.Create(ctx, service); err != nil {
		return err
	}

	logrus.Info("createService success")

	return nil
}

func createDeployment(ctx context.Context, r *ElasticWebReconciler, web *k8sv1.ElasticWeb) error {
	logrus.Info("createDeployment")

	expectReplicas := getExpectReplicas(web)

	logrus.Infof("createDeployment expectReplicas=%d", expectReplicas)

	dp := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: web.Namespace,
			Name:      web.Name,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(int32(expectReplicas)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": APP_NAME,
				},
			},

			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": APP_NAME,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: APP_NAME,
							// 用指定的镜像
							Image:           web.Spec.Image,
							ImagePullPolicy: "IfNotPresent",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolSCTP,
									ContainerPort: CONTAINER_PORT,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse(CPU_REQUEST),
									"memory": resource.MustParse(MEM_REQUEST),
								},
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse(CPU_LIMIT),
									"memory": resource.MustParse(MEM_LIMIT),
								},
							},
						},
					},
				},
			},
		},
	}

	// 这一步非常关键！
	// 建立关联后，删除elasticweb资源时就会将deployment也删除掉
	logrus.Info("set reference")
	if err := controllerutil.SetControllerReference(web, dp, r.Scheme); err != nil {
		logrus.Error(err, "SetControllerReference error")
		return err
	}

	logrus.Info("start create deployment")
	if err := r.Create(ctx, dp); err != nil {
		logrus.Error(err, "create deployment error")
		return err
	}

	logrus.Info("createDeployment success")
	return nil
}

func updateStatus(ctx context.Context, r *ElasticWebReconciler, web *k8sv1.ElasticWeb) error {
	SingleQPS := *(web.Spec.SingleQPS)

	replicas := getExpectReplicas(web)

	if web.Status.RealQPS == nil {
		web.Status.RealQPS = new(int)
	}

	*web.Status.RealQPS = SingleQPS * replicas
	logrus.Infof("singlePosQPS [%d], replicas [%d], realQPS [%d]", SingleQPS, replicas, *web.Status.RealQPS)

	if err := r.Update(ctx, web); err != nil {
		logrus.Errorf("updateStatus Update instance err=%v", err)
		return err
	}

	return nil
}
