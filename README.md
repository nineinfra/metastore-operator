# metastore-operator
This is a Kubernetes Operator to manage Apache Hive Metastore

## Description
The metastore-operator is an important part of the Nineinfra Data Platform, a curated combination of open-source projects
including Spark, Flink, HDFS, Kafka, ClickHouse, Kyuubi, and Superset, all working together seamlessly to provide users
with a stable and user-friendly big data processing platform. Nineinfra is a full-stack data platform built on Kubernetes,
capable of running on public cloud, private cloud, or on-premises environments.

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Installing on a kubernetes cluster
Install metastore operator by helm:

```sh
helm repo add nineinfra-charts https://nineinfra.github.io/nineinfra-charts/
kubectl create namespace metastore-operator
helm install metastore-operator nineinfra-charts/metastore-operator -n metastore-operator
```

### Deploying a metastore cluster by metastore-operator
1. Obtain the necessary configuration information, including HDFS cluster or Minio cluster, and Spark cluster configuration information.
   example:
```sh
kubectl get svc -n dwh |grep hdfs
hdfs              ClusterIP   10.100.208.68    <none>        9820/TCP,9870/TCP,9871/TCP    
                              
kubectl get svc -n minio-t1 |grep minio
minio              LoadBalancer   10.106.61.173   <pending>     80:31063/TCP     11d
minio-t1-console   LoadBalancer   10.96.125.48    <pending>     9090:30848/TCP   11d
minio-t1-hl        ClusterIP      None            <none>        9000/TCP         11d
```
And the service of hdfs is hdfs with the suffix .dwh.svc

2. Edit the cr yaml, there is a sample file like config/samples/metastore_v1alpha1_metastorecluster.yaml
```yaml
apiVersion: metastore.nineinfra.tech/v1alpha1
kind: MetastoreCluster
metadata:
  labels:
    app.kubernetes.io/name: metastorecluster
    app.kubernetes.io/instance: metastorecluster-sample
    app.kubernetes.io/part-of: metastore-operator
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/created-by: metastore-operator
  name: metastorecluster-sample
spec:
  metastoreVersion: 3.1.3
  metastoreImage:
    repository: nineinfra/metastore
    tag: v3.1.3
  metastoreResource:
    replicas: 1
  metastoreConf:
    "hive.metastore.warehouse.dir": "/usr/hive/warehouse"
  clusterRefs:
    - name: "database"
      type: "database"
      database:
        connectionUrl: "jdbc:postgresql://postgresql:5432/hive"
        dbType: postgres
        userName: hive
        password: hive
    - name: "minio"
      type: "minio"
      minio:
        endpoint: "http://192.168.123.24:31063"
        accessKey: "984GcQyUWobTVl3B"
        secretKey: "wE5ffRYxSacalsYT5UAVgo1AMlK2uune"
        sslEnabled: "false"
        pathStyleAccess: "true"
```
3. Deploy a metastore cluster
```sh
kubectl apply -f metastore_v1alpha1_metastorecluster_postgres.yaml -n metastore-operator
```
4. Get status of the metastore cluster,you can access the metastore cluster with the service in the output
```sh
kubectl get metastorecluster metastorecluster-sample -n metastore-operator -o yaml

status:
  creationTime: "2023-10-15T15:01:48Z"
  exposedInfos:
  - exposedType: thrift-http
    name: thrift-http-0
    serviceName: metastorecluster-sample-metastore
    servicePort:
      name: thrift-http
      port: 9083
      protocol: TCP
      targetPort: 9083
  updateTime: "2023-10-15T15:01:48Z"
```
### Edit the cr of the kyuubi
The hive.metastore.uris value is from the metastorecluster's status above
```yaml
apiVersion: kyuubi.nineinfra.tech/v1alpha1
kind: KyuubiCluster
metadata:
  labels:
    app.kubernetes.io/name: kyuubicluster
    app.kubernetes.io/instance: kyuubicluster-sample
    app.kubernetes.io/part-of: kyuubi-operator
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/created-by: kyuubi-operator
  name: kyuubicluster-sample
spec:
  kyuubiVersion: 1.8.1
  kyuubiImage:
    repository: 172.18.123.24:30003/library/kyuubi
    tag: v1.8.1-minio
  kyuubiResource:
    replicas: 1
  kyuubiConf:
    kyuubi.kubernetes.namespace: dwh
    kyuubi.frontend.connection.url.use.hostname: 'false'
    kyuubi.frontend.thrift.binary.bind.port: '10009'
    kyuubi.frontend.thrift.http.bind.port: '10010'
    kyuubi.frontend.rest.bind.port: '10099'
    kyuubi.frontend.mysql.bind.port: '3309'
    kyuubi.frontend.protocols: REST,THRIFT_BINARY
    kyuubi.metrics.enabled: 'false'
  clusterRefs:
    - name: spark
      type: spark
      spark:
        sparkMaster: k8s
        sparkImage:
          repository: 172.18.123.24:30003/library/spark
          tag: v3.2.4-minio
        sparkNamespace: dwh
        sparkDefaults:
          spark.hadoop.fs.s3a.access.key: 984GcQyUWobTVl3B
          spark.hadoop.fs.s3a.secret.key: wE5ffRYxSacalsYT5UAVgo1AMlK2uune
          spark.hadoop.fs.s3a.path.style.access: "true"
          spark.hadoop.fs.s3a.connection.ssl.enabled: "false"
          spark.hadoop.fs.s3a.endpoint: http://172.18.123.24:31063
    - name: hdfs
      type: hdfs
      hdfs:
        coreSite:
        hdfsSite:
          dfs.client.block.write.retries: '3'
    - name: metastore
      type: metastore
      metastore:
        hiveSite:
          hive.metastore.uris: thrift://metastorecluster-sample-metastore.dwh.svc:9083
          hive.metastore.warehouse.dir: s3a://usr/hive/warehouse
```
### Executing some example sql statements
1. Login the kyuubi pod
```sh
kubectl get pod -n kyuubi-operator
NAME                                                                                                                              READY   STATUS      RESTARTS   AGE
kyuubi-kyuubi-user-spark-sql-anonymous-default-2f33bda8-a9b0-4584-a726-1d653c809d15-2f33bda8-a9b0-4584-a726-1d653c809d15-exec-1   0/1     Completed   0          92m
kyuubi-kyuubi-user-spark-sql-anonymous-default-2f33bda8-a9b0-4584-a726-1d653c809d15-2f33bda8-a9b0-4584-a726-1d653c809d15-exec-2   0/1     Completed   0          92m
kyuubi-operator-deployment-57b54cbc6-fc8bz                                                                                        1/1     Running     0          85m
kyuubicluster-sample-kyuubi-0                                                                                                     1/1     Running     0          83m

kubectl exec -it kyuubicluster-sample-kyuubi-0 -n kyuubi-operator -- bash
```
2. Run beeline command
```sh
kyuubi@kyuubicluster-sample-kyuubi-0:/opt/kyuubi$ cd bin
kyuubi@kyuubicluster-sample-kyuubi-0:/opt/kyuubi/bin$ ./beeline
Warn: Not find kyuubi environment file /opt/kyuubi/conf/kyuubi-env.sh, using default ones...
Beeline version 1.8.0-SNAPSHOT by Apache Kyuubi
beeline> 
```
3. Connect the kyuubi cluster by the thrift-binary protocol through the service kyuubicluster-sample-kyuubi
```sh
beeline> !connect jdbc:hive2://kyuubicluster-sample-kyuubi:10009
```
4. Execute some example sql statements
```sh
0: jdbc:hive2://kyuubicluster-sample-kyuubi:1> show databases;
+------------+
| namespace  |
+------------+
| default    |
| test       |
+------------+
0: jdbc:hive2://kyuubicluster-sample-kyuubi:1> use test;
0: jdbc:hive2://kyuubicluster-sample-kyuubi:1> create table test3 (name string,id int);
0: jdbc:hive2://kyuubicluster-sample-kyuubi:1> insert into test3 values("kyuubi",1);
0: jdbc:hive2://kyuubicluster-sample-kyuubi:1> select * from test3;
+---------+-----+
|  name   | id  |
+---------+-----+
| kyuubi  | 1   |
+---------+-----+
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

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

