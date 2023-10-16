# Helm Chart for Metastore Operator for Apache Hive Metastore

This Helm Chart can be used to install Custom Resource Definitions and the Operator for Apache Hive Metastore provided by Nineinfra.

## Requirements

- Create a [Kubernetes Cluster](../Readme.md)
- Install [Helm](https://helm.sh/docs/intro/install/)

## Install the Metastore Operator for Apache Hive Metastore

```bash
# From the root of the operator repository

helm install metastore-operator charts/metastore-operator
```

## Usage of the CRDs

The usage of this operator and its CRDs is described in the [documentation](https://github.com/nineinfra/metastore-operator/blob/main/README.md).

## Links

https://github.com/nineinfra/metastore-operator
