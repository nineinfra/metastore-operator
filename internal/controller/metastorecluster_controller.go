/*
Copyright 2023 nineinfra.

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

package controller

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"
	"strings"

	metastorev1alpha1 "github.com/nineinfra/metastore-operator/api/v1alpha1"
)

// MetastoreClusterReconciler reconciles a MetastoreCluster object
type MetastoreClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=metastore.nineinfra.tech,resources=metastoreclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=metastore.nineinfra.tech,resources=metastoreclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=metastore.nineinfra.tech,resources=metastoreclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MetastoreCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *MetastoreClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var metastore metastorev1alpha1.MetastoreCluster
	err := r.Get(ctx, req.NamespacedName, &metastore)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Object not found, it could have been deleted")
		} else {
			logger.Info("Error occurred during fetching the object")
		}
		return ctrl.Result{}, err
	}
	requestArray := strings.Split(fmt.Sprint(req), "/")
	requestName := requestArray[1]

	if requestName == metastore.Name {
		logger.Info("Create or update metastoreclusters")
		err = r.reconcileClusters(ctx, &metastore, logger)
		if err != nil {
			logger.Info("Error occurred during create or update metastoreclusters")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *MetastoreClusterReconciler) constructLabels(cluster *metastorev1alpha1.MetastoreCluster) map[string]string {
	return map[string]string{
		"cluster": cluster.Name,
		"app":     metastorev1alpha1.ClusterSign,
	}
}
func (r *MetastoreClusterReconciler) reconcileResource(ctx context.Context,
	cluster *metastorev1alpha1.MetastoreCluster,
	constructFunc func(*metastorev1alpha1.MetastoreCluster) (client.Object, error),
	existingResource client.Object,
	resourceType string) error {
	logger := log.FromContext(ctx)
	resourceName := cluster.Name + metastorev1alpha1.ClusterNameSuffix
	err := r.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: cluster.Namespace}, existingResource)
	if err != nil && errors.IsNotFound(err) {
		resource, err := constructFunc(cluster)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Failed to define new %s resource for Nifi", resourceType))

			// The following implementation will update the status
			meta.SetStatusCondition(&cluster.Status.Conditions, metav1.Condition{Type: metastorev1alpha1.StateFailed,
				Status: metav1.ConditionFalse, Reason: "Reconciling",
				Message: fmt.Sprintf("Failed to create %s for the custom resource (%s): (%s)", resourceType, cluster.Name, err)})

			if err := r.Status().Update(ctx, nine); err != nil {
				logger.Error(err, "Failed to update cluster status")
				return err
			}

			return err
		}

		logger.Info(fmt.Sprintf("Creating a new %s", resourceType),
			fmt.Sprintf("%s.Namespace", resourceType), resource.GetNamespace(), fmt.Sprintf("%s.Name", resourceType), resource.GetName())

		if err = r.Create(ctx, resource); err != nil {
			logger.Error(err, fmt.Sprintf("Failed to create new %s", resourceType),
				fmt.Sprintf("%s.Namespace", resourceType), resource.GetNamespace(), fmt.Sprintf("%s.Name", resourceType), resource.GetName())
			return err
		}

		if err := r.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: cluster.Namespace}, existingResource); err != nil {
			logger.Error(err, fmt.Sprintf("Failed to get newly created %s", resourceType))
			return err
		}

	} else if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to get %s", resourceType))
		return err
	}
	return nil
}

func (r *MetastoreClusterReconciler) constructServiceAccount(cluster *metastorev1alpha1.MetastoreCluster) (*corev1.ServiceAccount, error) {
	saDesired := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + metastorev1alpha1.ClusterSign,
			Namespace: cluster.Namespace,
			Labels:    r.constructLabels(cluster),
		},
	}

	if err := ctrl.SetControllerReference(cluster, saDesired, r.Scheme); err != nil {
		return saDesired, err
	}

	return saDesired, nil
}

func (r *MetastoreClusterReconciler) constructRole(cluster *metastorev1alpha1.MetastoreCluster) (*rbacv1.Role, error) {
	roleDesired := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + metastorev1alpha1.ClusterSign,
			Namespace: cluster.Namespace,
			Labels:    r.constructLabels(cluster),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"pods",
					"configmaps",
					"services",
					"persistentvolumeclaims",
				},
				Verbs: []string{
					"create",
					"list",
					"delete",
					"watch",
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(cluster, roleDesired, r.Scheme); err != nil {
		return roleDesired, err
	}

	return roleDesired, nil
}

func (r *MetastoreClusterReconciler) constructRoleBinding(cluster *metastorev1alpha1.MetastoreCluster) (*rbacv1.RoleBinding, error) {
	roleBindingDesired := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + metastorev1alpha1.ClusterSign,
			Namespace: cluster.Namespace,
			Labels:    r.constructLabels(cluster),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: cluster.Name + metastorev1alpha1.ClusterSign,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     cluster.Name + metastorev1alpha1.ClusterSign,
		},
	}

	if err := ctrl.SetControllerReference(cluster, roleBindingDesired, r.Scheme); err != nil {
		return roleBindingDesired, err
	}

	return roleBindingDesired, nil
}

func (r *MetastoreClusterReconciler) reconcileRbacResources(ctx context.Context, cluster *metastorev1alpha1.MetastoreCluster, logger logr.Logger) error {
	existingResource := &corev1.ServiceAccount{}
	err := r.reconcileResource(ctx, cluster, r.constructServiceAccount, existingResource, "ServiceAccount")
	if err != nil {
		logger.Error(err, "Error occurred during reconcileResource for serviceaccount")
		return err
	}

	existingResource = &rbacv1.RoleBinding{}
	err = r.reconcileResource(ctx, cluster, r.constructRole, existingResource, "RoleBinding")
	if err != nil {
		logger.Error(err, "Error occurred during reconcileResource role")
		return err
	}

	existingResource = &rbacv1.Role{}
	err = r.reconcileResource(ctx, cluster, r.constructRoleBinding, existingResource, "Role")
	if err != nil {
		logger.Error(err, "Error occurred during reconcileResource rolebinding")
		return err
	}

	return nil
}

func (r *MetastoreClusterReconciler) constructConfigMap(cluster *metastorev1alpha1.MetastoreCluster) (*corev1.ConfigMap, error) {

	cmDesired := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + metastorev1alpha1.ClusterSign,
			Namespace: cluster.Namespace,
			Labels:    r.constructLabels(cluster),
			},
		},
		Data: map[string]string{},
	}
	clusterConf := make(map[string]string)
	for k, v := range metastore.spec.MetastoreConf {
		clusterConf[k] = v
	}

	for _, clusterRef := range cluster.Spec.ClusterRefs {
		switch clusterRef.Type {
		case metastorev1alpha1.DatabaseClusterType:
			switch clusterRef.DbType {
			case "mysql":
				clusterConf["javax.jdo.option.ConnectionDriverNam"] = "org.mysql.Driver"
			case "postgres":
				clusterConf["javax.jdo.option.ConnectionDriverNam"] = "org.postgresql.Driver"
			}
			clusterConf["javax.jdo.option.ConnectionURL"] = clusterRef.ConnectionUrl
			clusterConf["javax.jdo.option.ConnectionUserName"] = clusterRef.UserName
			clusterConf["javax.jdo.option.ConnectionPassword"] = clusterRef.Password
		case metastorev1alpha1.MinioClusterType:
			clusterConf["fs.s3a.endpoint"] = clusterRef.Endpoint
			clusterConf["fs.s3a.path.style.access"] = clusterRef.PathStyleAccess
			clusterConf["fs.s3a.connection.ssl.enabled"] = clusterRef.SSLEnabled
			clusterConf["fs.s3a.access.key"] = clusterRef.AccessKey
			clusterConf["fs.s3a.secret.key"] = cluster.SecretKey
		case metastorev1alpha1.HdfsClusterType:
			cmDesired.Data["hdfs-site.xml"] = map2Xml(clusterRef.Hdfs.HdfsSite)
			cmDesired.Data["core-site.xml"] = map2Xml(clusterRef.Hdfs.CoreSite)
		}
	}
	clusterConf["hive.metastore.warehouse.dir"] = "/usr/hive/warehouse"
	cmDesired["hive-site.xml"] = map2Xml(clusterConf)

	if err := ctrl.SetControllerReference(cluster, cmDesired, r.Scheme); err != nil {
		return cmDesired, err
	}

	return cmDesired, nil
}

func (r *MetastoreClusterReconciler) reconcileConfigmap(ctx context.Context, cluster *metastorev1alpha1.MetastoreCluster, logger logr.Logger) error {
	existingConfigMap := &corev1.ConfigMap{}
	err := r.reconcileResource(ctx, cluster, r.constructConfigMap, existingConfigMap, "ConfigMap")
	if err != nil {
		logger.Error(err, "Error occurred during reconcileResource configmap")
		return err
	}
	return nil
}

func (r *MetastoreClusterReconciler) constructWorkload(cluster *metastorev1alpha1.MetastoreCluster) (*appsv1.StatefulSet, error) {
	for _, v := range cluster.Spec.ClusterRefs {
		if v.Type == DatabaseClusterType {
			dbType := v.DbType
		}
	}
	stsDesired := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + metastorev1alpha1.ClusterNameSuffix,
			Namespace: cluster.Namespace,
			Labels: r.constructLabels(cluster),
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: r.constructLabels(cluster),
			},
			ServiceName: cluster.Name + metastorev1alpha1.ClusterNameSuffix,
			Replicas:    int32Ptr(cluster.Spec.MetastoreResource.Replicas),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: r.constructLabels(cluster),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            cluster.Name,
							Image:           cluster.Spec.MetastoreImage.Repository + ":" + cluster.Spec.MetastoreImage.Tag,
							ImagePullPolicy: corev1.PullPolicy(cluster.Spec.MetastoreImage.PullPolicy),
							Env: []Object{
								{
									Name:  "DB_TYPE",
									Value: dbType,
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "thrift-http",
									ContainerPort: int32(9083),
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"/bin/bash",
											"-c",
											"bin/kyuubi status",
										},
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
								TimeoutSeconds:      2,
								FailureThreshold:    10,
								SuccessThreshold:    1,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"/bin/bash",
											"-c",
											""},
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
								TimeoutSeconds:      2,
								FailureThreshold:    10,
								SuccessThreshold:    1,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      cluster.Name + "-hdfssite",
									MountPath: "/opt/hadoop/etc/hadoop/hdfs-site.xml",
									SubPath:   "hdfs-site.xml",
								},
								{
									Name:      cluster.Name + "-coresite",
									MountPath: "/opt/hadoop/etc/hadoop/core-site.xml",
									SubPath:   "core-site.xml",
								},
								{
									Name:      cluster.Name + "-hivesite",
									MountPath: "/opt/hive/conf/hive-site.xml",
									SubPath:   "hive-site.xml",
								},
							},
						},
					},
					RestartPolicy:      corev1.RestartPolicyAlways,
					ServiceAccountName: cluster.Name + metastorev1alpha1.ClusterNameSuffix,
					Volumes: []corev1.Volume{
						{
							Name: cluster.Name + "-hdfssite",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: cluster.Name + metastorev1alpha1.ClusterNameSuffix,
									},
									Items: []corev1.KeyToPath{
										{
											Key:  "hdfs-site.xml",
											Path: "hdfs-site.xml",
										},
									},
								},
							},
						},
						{
							Name: cluster.Name + "-coresite",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: cluster.Name + metastorev1alpha1.ClusterNameSuffix,
									},
									Items: []corev1.KeyToPath{
										{
											Key:  "core-site.xml",
											Path: "core-site.xml",
										},
									},
								},
							},
						},
						{
							Name: cluster.Name + "-hivesite",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: cluster.Name + metastorev1alpha1.ClusterNameSuffix,
									},
									Items: []corev1.KeyToPath{
										{
											Key:  "hive-site.xml",
											Path: "hive-site.xml",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(cluster, stsDesired, r.Scheme); err != nil {
		return stsDesired, err
	}
	return stsDesired, nil
}

func (r *MetastoreClusterReconciler) reconcileWorkload(ctx context.Context, cluster *metastorev1alpha1.MetastoreCluster, logger logr.Logger) error {
	existingWorkload := &appsv1.StatefulSet{}
	err := r.reconcileResource(ctx, cluster, r.constructWorkload, existingWorkload, "StatefulSet")
	if err != nil {
		logger.Error(err, "Error occurred during reconcileResource role")
		return err
	}
	return nil
}

func (r *MetastoreClusterReconciler) contructService(cluster *metastorev1alpha1.MetastoreCluster) (*corev1.Service, error) {
	svcDesired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + metastorev1alpha1.ClusterNameSuffix,
			Namespace: cluster.Namespace,
			Labels: map[string]string{
				"cluster": cluster.Name,
				"app":     "metastore",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name: string(metastorev1alpha1.ExposedThriftHttp),
					Port: 9083,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: int32(9083),
					},
				},
			},
			Selector: map[string]string{
				"cluster": cluster.Name,
				"app":     "metastore",
			},
		},
	}

	if err := ctrl.SetControllerReference(cluster, svcDesired, r.Scheme); err != nil {
		return svcDesired, err
	}

	return svcDesired, nil
}

func (r *MetastoreClusterReconciler) reconcileService(ctx context.Context, cluster *metastorev1alpha1.MetastoreCluster) (*corev1.Service, error) {
	existingService := &corev1.Service{}
	err := r.reconcileResource(ctx, cluster, r.constructService, existingService, "Service")
	if err != nil {
		logger.Error(err, "Error occurred during reconcileResource service")
		return existingService, err
	}
	return existingService, nil
}

func (r *MetastoreClusterReconciler) updateClusterStatus(ctx context.Context, cluster *metastorev1alpha1.MetastoreCluster, service *corev1.Service) error {
	exposedInfos := make([]metastorev1alpha1.ExposedInfo, 0)
	for k, v := range service.Spec.Ports {
		var exposedInfo metastorev1alpha1.ExposedInfo
		exposedInfo.ServiceName = service.Name
		exposedInfo.Name = v.Name + "-" + strconv.Itoa(k)
		exposedInfo.ExposedType = metastorev1alpha1.ExposedType(v.Name)
		v.DeepCopyInto(&exposedInfo.ServicePort)
		exposedInfos = append(exposedInfos, exposedInfo)
	}

	desiredStatus := &metastorev1alpha1.MetastoreClusterStatus{
		ExposedInfos: exposedInfos,
	}

	if cluster.Status.ExposedInfos == nil || !reflect.DeepEqual(exposedInfos, cluster.Status.ExposedInfos) {
		if cluster.Status.ExposedInfos == nil {
			desiredStatus.CreationTime = metav1.Now()
		} else {
			desiredStatus.CreationTime = cluster.Status.CreationTime
		}
		desiredStatus.UpdateTime = metav1.Now()
		desiredStatus.DeepCopyInto(&cluster.Status)
		err := r.Status().Update(ctx, cluster)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *MetastoreClusterReconciler) reconcileClusters(ctx context.Context, cluster *metastorev1alpha1.MetastoreCluster, logger logr.Logger) error {
	err := r.reconcileRbacResources(ctx, cluster, logger)
	if err != nil {
		logger.Error(err, "Error occurred during reconcileRbacResources")
		return err
	}

	err = r.reconcileConfigmap(ctx, cluster, logger)
	if err != nil {
		logger.Error(err, "Error occurred during reconcileConfigmap")
		return err
	}

	err = r.reconcileWorkload(ctx, cluster, logger)
	if err != nil {
		logger.Error(err, "Error occurred during reconcileWorkload")
		return err
	}

	existingService, err := r.reconcileService(ctx, cluster)
	if err != nil {
		logger.Error(err, "Error occurred during reconcileService")
		return err
	}

	err = r.updateClusterStatus(ctx, cluster, existingService)
	if err != nil {
		logger.Error(err, "Error occurred during updateClusterStatus")
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MetastoreClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&metastorev1alpha1.MetastoreCluster{}).
		Complete(r)
}
