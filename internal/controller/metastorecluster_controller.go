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
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
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

	var cluster metastorev1alpha1.MetastoreCluster
	err := r.Get(ctx, req.NamespacedName, &cluster)
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

	if requestName == cluster.Name {
		logger.Info("Create or update clusters")
		err = r.reconcileClusters(ctx, &cluster, logger)
		if err != nil {
			logger.Info("Error occurred during create or update clusters")
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

func (r *MetastoreClusterReconciler) constructProbe() *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.IntOrString{
					Type:   intstr.String,
					StrVal: string(metastorev1alpha1.ExposedThriftHttp),
				},
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       10,
		TimeoutSeconds:      2,
		FailureThreshold:    10,
		SuccessThreshold:    1,
	}
}

func (r *MetastoreClusterReconciler) constructVolume(cluster *metastorev1alpha1.MetastoreCluster) []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: cluster.Name + "-hivesite",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: r.resourceName(cluster),
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
	}

	for _, v := range cluster.Spec.ClusterRefs {
		if v.Type == metastorev1alpha1.HdfsClusterType {
			if v.Hdfs.HdfsSite != nil {
				volume := corev1.Volume{
					Name: cluster.Name + "-hdfssite",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: r.resourceName(cluster),
							},
							Items: []corev1.KeyToPath{
								{
									Key:  "hdfs-site.xml",
									Path: "hdfs-site.xml",
								},
							},
						},
					},
				}
				volumes = append(volumes, volume)
			}
			if v.Hdfs.HdfsSite != nil {
				volume := corev1.Volume{
					Name: cluster.Name + "-coresite",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: r.resourceName(cluster),
							},
							Items: []corev1.KeyToPath{
								{
									Key:  "core-site.xml",
									Path: "core-site.xml",
								},
							},
						},
					},
				}
				volumes = append(volumes, volume)
			}
		}
	}
	return volumes
}

func (r *MetastoreClusterReconciler) constructVolumeMounts(cluster *metastorev1alpha1.MetastoreCluster) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      cluster.Name + "-hivesite",
			MountPath: "/opt/hive/conf/hive-site.xml",
			SubPath:   "hive-site.xml",
		},
	}
	for _, v := range cluster.Spec.ClusterRefs {
		if v.Type == metastorev1alpha1.HdfsClusterType {
			if v.Hdfs.HdfsSite != nil {
				volumeMount := corev1.VolumeMount{
					Name:      cluster.Name + "-hdfssite",
					MountPath: "/opt/hadoop/etc/hadoop/hdfs-site.xml",
					SubPath:   "hdfs-site.xml",
				}
				volumeMounts = append(volumeMounts, volumeMount)
			}
			if v.Hdfs.CoreSite != nil {
				volumeMount := corev1.VolumeMount{
					Name:      cluster.Name + "-coresite",
					MountPath: "/opt/hadoop/etc/hadoop/core-site.xml",
					SubPath:   "core-site.xml",
				}
				volumeMounts = append(volumeMounts, volumeMount)
			}
		}
	}
	return volumeMounts
}

func (r *MetastoreClusterReconciler) resourceName(cluster *metastorev1alpha1.MetastoreCluster) string {
	return cluster.Name + metastorev1alpha1.ClusterNameSuffix
}

func (r *MetastoreClusterReconciler) objectMeta(cluster *metastorev1alpha1.MetastoreCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      r.resourceName(cluster),
		Namespace: cluster.Namespace,
		Labels:    r.constructLabels(cluster),
	}
}

func (r *MetastoreClusterReconciler) reconcileResource(ctx context.Context, cluster *metastorev1alpha1.MetastoreCluster,
	constructFunc func(*metastorev1alpha1.MetastoreCluster) (client.Object, error),
	compareFunc func(client.Object, client.Object) bool,
	updateFunc func(context.Context, client.Object, client.Object) error,
	existingResource client.Object, resourceType string) error {
	logger := log.FromContext(ctx)
	resourceName := r.resourceName(cluster)
	desiredResource, err := constructFunc(cluster)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to define new %s resource for Cluster", resourceType))

		// The following implementation will update the status
		meta.SetStatusCondition(&cluster.Status.Conditions, metav1.Condition{Type: metastorev1alpha1.StateFailed,
			Status: metav1.ConditionFalse, Reason: "Reconciling",
			Message: fmt.Sprintf("Failed to create %s for the custom resource (%s): (%s)", resourceType, cluster.Name, err)})

		if err := r.Status().Update(ctx, cluster); err != nil {
			logger.Error(err, "Failed to update cluster status")
			return err
		}

		return err
	}

	err = r.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: cluster.Namespace}, existingResource)
	if err != nil && errors.IsNotFound(err) {

		logger.Info(fmt.Sprintf("Creating a new %s", resourceType),
			fmt.Sprintf("%s.Namespace", resourceType), desiredResource.GetNamespace(), fmt.Sprintf("%s.Name", resourceType), desiredResource.GetName())

		if err = r.Create(ctx, desiredResource); err != nil {
			logger.Error(err, fmt.Sprintf("Failed to create new %s", resourceType),
				fmt.Sprintf("%s.Namespace", resourceType), desiredResource.GetNamespace(), fmt.Sprintf("%s.Name", resourceType), desiredResource.GetName())
			return err
		}

		if err := r.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: cluster.Namespace}, existingResource); err != nil {
			logger.Error(err, fmt.Sprintf("Failed to get newly created %s", resourceType))
			return err
		}

	} else if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to get %s", resourceType))
		return err
	} else {
		if compareFunc != nil {
			logger.Info(fmt.Sprintf("Checking modified %s", resourceType),
				fmt.Sprintf("%s.Namespace", resourceType), cluster.Namespace, fmt.Sprintf("%s.Name", resourceType), cluster.Name)
			if !compareFunc(existingResource, desiredResource) {
				err = updateFunc(ctx, existingResource, desiredResource)
				if err != nil {
					logger.Error(err, fmt.Sprintf("Failed to update %s", resourceType),
						fmt.Sprintf("%s.Namespace", resourceType), desiredResource.GetNamespace(), fmt.Sprintf("%s.Name", resourceType), desiredResource.GetName())
					return err
				}
			}
		}
	}
	return nil
}

func (r *MetastoreClusterReconciler) constructServiceAccount(cluster *metastorev1alpha1.MetastoreCluster) (client.Object, error) {
	saDesired := &corev1.ServiceAccount{
		ObjectMeta: r.objectMeta(cluster),
	}

	if err := ctrl.SetControllerReference(cluster, saDesired, r.Scheme); err != nil {
		return saDesired, err
	}

	return saDesired, nil
}

func (r *MetastoreClusterReconciler) constructClusterRole(cluster *metastorev1alpha1.MetastoreCluster) (client.Object, error) {
	roleDesired := &rbacv1.Role{
		ObjectMeta: r.objectMeta(cluster),
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

func (r *MetastoreClusterReconciler) constructRoleBinding(cluster *metastorev1alpha1.MetastoreCluster) (client.Object, error) {
	roleBindingDesired := &rbacv1.RoleBinding{
		ObjectMeta: r.objectMeta(cluster),
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: r.resourceName(cluster),
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     r.resourceName(cluster),
		},
	}

	if err := ctrl.SetControllerReference(cluster, roleBindingDesired, r.Scheme); err != nil {
		return roleBindingDesired, err
	}

	return roleBindingDesired, nil
}

func (r *MetastoreClusterReconciler) reconcileRbacResources(ctx context.Context, cluster *metastorev1alpha1.MetastoreCluster, logger logr.Logger) error {
	existingSa := &corev1.ServiceAccount{}
	err := r.reconcileResource(ctx, cluster, r.constructServiceAccount, nil, nil, existingSa, "ServiceAccount")
	if err != nil {
		logger.Error(err, "Error occurred during reconcileResource for serviceaccount")
		return err
	}

	existingRole := &rbacv1.Role{}
	err = r.reconcileResource(ctx, cluster, r.constructClusterRole, nil, nil, existingRole, "Role")
	if err != nil {
		logger.Error(err, "Error occurred during reconcileResource role")
		return err
	}

	existingRoleBinding := &rbacv1.RoleBinding{}
	err = r.reconcileResource(ctx, cluster, r.constructRoleBinding, nil, nil, existingRoleBinding, "RoleBinding")
	if err != nil {
		logger.Error(err, "Error occurred during reconcileResource rolebinding")
		return err
	}

	return nil
}

func (r *MetastoreClusterReconciler) constructConfigMap(cluster *metastorev1alpha1.MetastoreCluster) (client.Object, error) {
	cmDesired := &corev1.ConfigMap{
		ObjectMeta: r.objectMeta(cluster),
		Data:       map[string]string{},
	}

	clusterConf := make(map[string]string)
	for k, v := range cluster.Spec.MetastoreConf {
		clusterConf[k] = v
	}

	for _, clusterRef := range cluster.Spec.ClusterRefs {
		switch clusterRef.Type {
		case metastorev1alpha1.DatabaseClusterType:
			switch clusterRef.Database.DbType {
			case "mysql":
				clusterConf["javax.jdo.option.ConnectionDriverName"] = "org.mysql.Driver"
			case "postgres":
				clusterConf["javax.jdo.option.ConnectionDriverName"] = "org.postgresql.Driver"
			}
			clusterConf["javax.jdo.option.ConnectionURL"] = clusterRef.Database.ConnectionUrl
			clusterConf["javax.jdo.option.ConnectionUserName"] = clusterRef.Database.UserName
			clusterConf["javax.jdo.option.ConnectionPassword"] = clusterRef.Database.Password
		case metastorev1alpha1.MinioClusterType:
			clusterConf["fs.s3a.endpoint"] = clusterRef.Minio.Endpoint
			clusterConf["fs.s3a.path.style.access"] = clusterRef.Minio.PathStyleAccess
			clusterConf["fs.s3a.connection.ssl.enabled"] = clusterRef.Minio.SSLEnabled
			clusterConf["fs.s3a.access.key"] = clusterRef.Minio.AccessKey
			clusterConf["fs.s3a.secret.key"] = clusterRef.Minio.SecretKey
		case metastorev1alpha1.HdfsClusterType:
			cmDesired.Data["hdfs-site.xml"] = map2Xml(clusterRef.Hdfs.HdfsSite)
			cmDesired.Data["core-site.xml"] = map2Xml(clusterRef.Hdfs.CoreSite)
		}
	}
	if _, ok := clusterConf["hive.metastore.warehouse.dir"]; !ok {
		clusterConf["hive.metastore.warehouse.dir"] = "/usr/hive/warehouse"
	}
	cmDesired.Data["hive-site.xml"] = map2Xml(clusterConf)

	if err := ctrl.SetControllerReference(cluster, cmDesired, r.Scheme); err != nil {
		return cmDesired, err
	}

	return cmDesired, nil
}

func (r *MetastoreClusterReconciler) compareConfigMap(cm1 client.Object, cm2 client.Object) bool {
	//Got data filed through reflect. The type of this filed is map.
	cm1Data := reflect.ValueOf(cm1).Elem().FieldByName("Data")
	cm2Data := reflect.ValueOf(cm2).Elem().FieldByName("Data")

	cm1Map := make(map[string]string)
	cm2Map := make(map[string]string)

	for _, k := range cm1Data.MapKeys() {
		cm1Map[k.Interface().(string)] = cm1Data.MapIndex(k).Interface().(string)
	}
	for _, k := range cm2Data.MapKeys() {
		cm2Map[k.Interface().(string)] = cm2Data.MapIndex(k).Interface().(string)
	}

	return compareConf(cm1Map, cm2Map)
}

func (r *MetastoreClusterReconciler) updateConfigMap(ctx context.Context, cm1 client.Object, cm2 client.Object) error {
	cm1Value := reflect.ValueOf(cm1)
	cm2Value := reflect.ValueOf(cm2)
	cm1Value.FieldByName("Data").Set(cm2Value.FieldByName("Data"))
	return r.Update(ctx, cm1Value.Interface().(client.Object))
}

func (r *MetastoreClusterReconciler) reconcileConfigmap(ctx context.Context, cluster *metastorev1alpha1.MetastoreCluster, logger logr.Logger) error {
	existingConfigMap := &corev1.ConfigMap{}
	err := r.reconcileResource(ctx, cluster, r.constructConfigMap, r.compareConfigMap, r.updateConfigMap, existingConfigMap, "ConfigMap")
	if err != nil {
		logger.Error(err, "Error occurred during reconcileResource configmap")
		return err
	}
	return nil
}

func (r *MetastoreClusterReconciler) constructWorkload(cluster *metastorev1alpha1.MetastoreCluster) (client.Object, error) {
	dbType := "mysql"
	for _, v := range cluster.Spec.ClusterRefs {
		if v.Type == metastorev1alpha1.DatabaseClusterType {
			dbType = v.Database.DbType
			break
		}
	}
	stsDesired := &appsv1.StatefulSet{
		ObjectMeta: r.objectMeta(cluster),
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: r.constructLabels(cluster),
			},
			ServiceName: r.resourceName(cluster),
			Replicas:    int32Ptr(cluster.Spec.MetastoreResource.Replicas),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: r.objectMeta(cluster),
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            cluster.Name,
							Image:           cluster.Spec.MetastoreImage.Repository + ":" + cluster.Spec.MetastoreImage.Tag,
							ImagePullPolicy: corev1.PullPolicy(cluster.Spec.MetastoreImage.PullPolicy),
							Env: []corev1.EnvVar{
								{
									Name:  "DB_TYPE",
									Value: dbType,
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          string(metastorev1alpha1.ExposedThriftHttp),
									ContainerPort: int32(9083),
								},
							},
							LivenessProbe:  r.constructProbe(),
							ReadinessProbe: r.constructProbe(),
							VolumeMounts:   r.constructVolumeMounts(cluster),
						},
					},
					RestartPolicy:      corev1.RestartPolicyAlways,
					ServiceAccountName: r.resourceName(cluster),
					Volumes:            r.constructVolume(cluster),
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(cluster, stsDesired, r.Scheme); err != nil {
		return stsDesired, err
	}
	return stsDesired, nil
}

func (r *MetastoreClusterReconciler) compareWorkload(workload1 client.Object, workload2 client.Object) bool {
	//got the spec field. The type is appsv1.StatefulSetSpec
	wl1Spec := reflect.ValueOf(workload1).Elem().FieldByName("Spec")
	wl2Spec := reflect.ValueOf(workload2).Elem().FieldByName("Spec")

	wl1StsSpec := wl1Spec.Interface().(appsv1.StatefulSetSpec)
	wl2StsSpec := wl2Spec.Interface().(appsv1.StatefulSetSpec)

	return *wl1StsSpec.Replicas == *wl2StsSpec.Replicas &&
		wl1StsSpec.Template.Spec.Containers[0].Image == wl2StsSpec.Template.Spec.Containers[0].Image &&
		wl1StsSpec.Template.Spec.Containers[0].ImagePullPolicy == wl2StsSpec.Template.Spec.Containers[0].ImagePullPolicy
}

func (r *MetastoreClusterReconciler) updateWorkload(ctx context.Context, workload1 client.Object, workload2 client.Object) error {
	wl1Value := reflect.ValueOf(workload1)
	wl2Value := reflect.ValueOf(workload2)
	wl1Spec := wl1Value.Elem().FieldByName("Spec")
	wl2Spec := wl2Value.Elem().FieldByName("Spec")
	wl1StsSpec := wl1Spec.Interface().(appsv1.StatefulSetSpec)
	wl2StsSpec := wl2Spec.Interface().(appsv1.StatefulSetSpec)

	wl1StsSpec.Replicas = wl2StsSpec.Replicas
	wl1StsSpec.Template.Spec.Containers[0].Image = wl2StsSpec.Template.Spec.Containers[0].Image
	wl1StsSpec.Template.Spec.Containers[0].ImagePullPolicy = wl2StsSpec.Template.Spec.Containers[0].ImagePullPolicy

	return r.Update(ctx, wl1Value.Interface().(client.Object))
}

func (r *MetastoreClusterReconciler) reconcileWorkload(ctx context.Context, cluster *metastorev1alpha1.MetastoreCluster, logger logr.Logger) error {
	existingWorkload := &appsv1.StatefulSet{}
	err := r.reconcileResource(ctx, cluster, r.constructWorkload, r.compareWorkload, r.updateWorkload, existingWorkload, "StatefulSet")
	if err != nil {
		logger.Error(err, "Error occurred during reconcileResource workload")
		return err
	}
	return nil
}

func (r *MetastoreClusterReconciler) constructService(cluster *metastorev1alpha1.MetastoreCluster) (client.Object, error) {
	svcDesired := &corev1.Service{
		ObjectMeta: r.objectMeta(cluster),
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
			Selector: r.constructLabels(cluster),
		},
	}

	if err := ctrl.SetControllerReference(cluster, svcDesired, r.Scheme); err != nil {
		return svcDesired, err
	}

	return svcDesired, nil
}

func (r *MetastoreClusterReconciler) reconcileService(ctx context.Context, cluster *metastorev1alpha1.MetastoreCluster, logger logr.Logger) (*corev1.Service, error) {
	existingService := &corev1.Service{}
	err := r.reconcileResource(ctx, cluster, r.constructService, nil, nil, existingService, "Service")
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

	existingService, err := r.reconcileService(ctx, cluster, logger)
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
