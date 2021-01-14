# k8s-aws-operator

Manage AWS Elastic IPs (EIPs) and Elastic Network Interfaces (ENIs) as Custom Resources in your Kubernetes cluster and assign them your pods.

**Warning:** This project is still work in progress. There might be breaking API changes in the future. Use at your own risk.

## Requirements

* Your pod IPs must be allocated from your VPC subnets. This is the default setup on AWS EKS by using the [AWS VPC CNI plugin](https://github.com/aws/amazon-vpc-cni-k8s).
* If you wish egress traffic to be sourced from assigned EIPs: In AWS VPC CNI plugin, `AWS_VPC_K8S_CNI_EXTERNALSNAT` must be set to `true` or `AWS_VPC_K8S_CNI_EXCLUDE_SNAT_CIDRS` must include the destination CIDR's.
* Your worker nodes must reside in a public subnet with an internet gateway attached.

## Installation

### Create an IAM role

Create an IAM role with the policy [here](iam/policy.json).

### Install the operator

Ensure that the k8s-aws-operator uses this role, e.g. using [»IAM Roles for Service Accounts« (IRSA)](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) or [kube2iam](https://github.com/jtblin/kube2iam)/[kiam](https://github.com/uswitch/kiam). Modify the manifests [here](deploy) accordingly, then run:

```bash
$ kubectl apply -f config/crd/bases/ # install Custom Resource Definition (CRD) for EIP Custom Resource
$ kubectl apply -f deploy/          # install the operator
```

## Usage

### EIPs

#### Basic usage

##### Allocate an EIP

Create a new file `example.yaml`:
```yaml
apiVersion: aws.k8s.logmein.com/v1alpha1
kind: EIP
metadata:
  name: my-eip
spec:
  tags:
    owner: My team
```

Apply it:
```bash
$ kubectl apply -f example.yaml
eip.aws.k8s.logmein.com/my-eip created
```

Describe it:
```bash
$ kubectl get eip my-eip
NAME     STATE      PUBLIC IP       POD
my-eip   allocated  34.228.250.93
```

###### Using BYOIP and requesting a specific address

Request a random address from a [BYOIP](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-byoip.html) address pool:

```yaml
apiVersion: aws.k8s.logmein.com/v1alpha1
kind: EIP
# ...
spec:
  publicIPv4Pool: <your pool ID here>
  # ...
```

Request a specific address from a BYOIP address pool:

```yaml
apiVersion: aws.k8s.logmein.com/v1alpha1
kind: EIP
# ...
spec:
  publicIPv4Address: 12.34.56.78
  # ...
```

##### Assign the EIP to a pod

Adjust `example.yaml` to include an `assignment` section:
```yaml
apiVersion: aws.k8s.logmein.com/v1alpha1
kind: EIP
metadata:
  name: my-eip
spec:
  tags:
    owner: My team
  assignment:
    podName: some-pod
```

Apply it:
```bash
$ kubectl apply -f example.yaml
eip.aws.k8s.logmein.com/my-eip configured
```

Describe it:
```bash
$ kubectl get eip my-eip
NAME     STATE      PUBLIC IP       POD
my-eip   assigned   34.228.250.93   my-pod
```

Allocating and assigning can also be done in one step.

##### Unassign an EIP from a pod

Remove the `assignment` section again and reapply the manifest.

##### Release the EIP

```bash
$ kubectl delete eip my-eip
eip.aws.k8s.logmein.com/my-eip deleted
```

Unassigning and releasing can also be done in one step.

#### One EIP per pod in a deployment / statefulset

##### EIP creation

You can use an `initContainer` as part of your pod definition to create the `EIP` custom resource. This requires that your pod has RBAC permissions to create `EIP` resources.

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: eip-user
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: eip-user-role
rules:
- apiGroups:
  - aws.k8s.logmein.com
  resources:
  - eips
  verbs:
  - '*'
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: eip-user-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: eip-user-role
subjects:
- kind: ServiceAccount
  name: eip-user
---
apiVersion: apps/v1
kind: Deployment
# ...
spec:
  # ...
  template:
    spec:
      # ...
      serviceAccountName: eip-user
      initContainers:
      - name: init-eip
        image: <some image that has kubectl>
        env:
        - name: MY_POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: MY_POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        command:
        - /bin/sh
        - -c
        - |
            # allocate and assign EIP
            cat <<EOS | kubectl apply -f-
            apiVersion: aws.k8s.logmein.com/v1alpha1
            kind: EIP
            metadata:
              name: $(MY_POD_NAME)
              namespace: $(MY_POD_NAMESPACE)
            spec:
              tags:
                owner: My team
                pod: $(MY_POD_NAME)
                namespace: $(MY_POD_NAMESPACE)
              assignment:
                podName: $(MY_POD_NAME)
            EOS

            # wait for EIP to be assigned
            while [ "$(kubectl get eip $(MY_POD_NAME) -o jsonpath='{.status.state}')" != "assigned" ]
            do
              sleep 1
            done
```

##### Cleanup

You can ensure that an EIP is released when your pod is terminated by including `ownerReferences` in your `EIP` resource. Setting `blockOwnerDeletion: true` prevents the pod from vanishing until the EIP is unassigned and released.

```yaml
apiVersion: aws.k8s.logmein.com/v1alpha1
kind: EIP
metadata:
  name: my-eip
  ownerReferences:
  - apiVersion: v1
    kind: Pod
    name: some-pod
    uid: ... # put the UID of the pod here
    blockOwnerDeletion: true
spec:
  tags:
    owner: My team
  assignment:
    podName: some-pod
```

### ENIs

To be documented
