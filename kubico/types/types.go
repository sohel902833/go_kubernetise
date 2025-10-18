package types

import "time"

// Pod represents a pod in the cluster
type Pod struct {
	APIVersion string      `json:"apiVersion" yaml:"apiVersion"`
	Kind       string      `json:"kind" yaml:"kind"`
	Metadata   Metadata    `json:"metadata" yaml:"metadata"`
	Spec       PodSpec     `json:"spec" yaml:"spec"`
	Status     PodStatus   `json:"status" yaml:"status"`
}

// ReplicaSet manages multiple identical pods
type ReplicaSet struct {
	APIVersion string         `json:"apiVersion" yaml:"apiVersion"`
	Kind       string         `json:"kind" yaml:"kind"`
	Metadata   Metadata       `json:"metadata" yaml:"metadata"`
	Spec       ReplicaSetSpec `json:"spec" yaml:"spec"`
	Status     ReplicaSetStatus `json:"status" yaml:"status"`
}

// ReplicaSetSpec defines the desired state of a ReplicaSet
type ReplicaSetSpec struct {
	Replicas int32             `json:"replicas" yaml:"replicas"`
	Selector map[string]string `json:"selector" yaml:"selector"`
	Template PodTemplateSpec   `json:"template" yaml:"template"`
}

// PodTemplateSpec describes the pods that will be created
type PodTemplateSpec struct {
	Metadata Metadata `json:"metadata" yaml:"metadata"`
	Spec     PodSpec  `json:"spec" yaml:"spec"`
}

// ReplicaSetStatus represents the current state
type ReplicaSetStatus struct {
	Replicas          int32     `json:"replicas"`
	ReadyReplicas     int32     `json:"readyReplicas"`
	AvailableReplicas int32     `json:"availableReplicas"`
	ObservedGeneration int64    `json:"observedGeneration"`
}

// Metadata contains metadata about the resource
type Metadata struct {
	Name      string            `json:"name" yaml:"name"`
	Namespace string            `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Labels    map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	CreatedAt time.Time         `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	UID       string            `json:"uid,omitempty" yaml:"uid,omitempty"`
	OwnerReferences []OwnerReference `json:"ownerReferences,omitempty" yaml:"ownerReferences,omitempty"`
}

// OwnerReference contains enough information to let you identify an owning object
type OwnerReference struct {
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`
	Kind       string `json:"kind" yaml:"kind"`
	Name       string `json:"name" yaml:"name"`
	UID        string `json:"uid" yaml:"uid"`
}

// PodSpec describes the desired state of a pod
type PodSpec struct {
	Containers  []Container `json:"containers" yaml:"containers"`
	NodeName    string      `json:"nodeName,omitempty" yaml:"nodeName,omitempty"`
	RestartPolicy string    `json:"restartPolicy,omitempty" yaml:"restartPolicy,omitempty"` // Always, OnFailure, Never
}

// Container describes a container in a pod
type Container struct {
	Name           string          `json:"name" yaml:"name"`
	Image          string          `json:"image" yaml:"image"`
	Command        []string        `json:"command,omitempty" yaml:"command,omitempty"`
	Args           []string        `json:"args,omitempty" yaml:"args,omitempty"`
	Env            []EnvVar        `json:"env,omitempty" yaml:"env,omitempty"`
	Ports          []ContainerPort `json:"ports,omitempty" yaml:"ports,omitempty"`
	LivenessProbe  *Probe          `json:"livenessProbe,omitempty" yaml:"livenessProbe,omitempty"`
	ReadinessProbe *Probe          `json:"readinessProbe,omitempty" yaml:"readinessProbe,omitempty"`
}

// Probe describes a health check to be performed against a container
type Probe struct {
	HTTPGet             *HTTPGetAction `json:"httpGet,omitempty" yaml:"httpGet,omitempty"`
	TCPSocket           *TCPSocketAction `json:"tcpSocket,omitempty" yaml:"tcpSocket,omitempty"`
	InitialDelaySeconds int32          `json:"initialDelaySeconds,omitempty" yaml:"initialDelaySeconds,omitempty"`
	PeriodSeconds       int32          `json:"periodSeconds,omitempty" yaml:"periodSeconds,omitempty"`
	TimeoutSeconds      int32          `json:"timeoutSeconds,omitempty" yaml:"timeoutSeconds,omitempty"`
	SuccessThreshold    int32          `json:"successThreshold,omitempty" yaml:"successThreshold,omitempty"`
	FailureThreshold    int32          `json:"failureThreshold,omitempty" yaml:"failureThreshold,omitempty"`
}

// HTTPGetAction describes an action based on HTTP Get requests
type HTTPGetAction struct {
	Path   string `json:"path" yaml:"path"`
	Port   int32  `json:"port" yaml:"port"`
	Scheme string `json:"scheme,omitempty" yaml:"scheme,omitempty"`
}

// TCPSocketAction describes an action based on opening a socket
type TCPSocketAction struct {
	Port int32 `json:"port" yaml:"port"`
}

// EnvVar represents an environment variable
type EnvVar struct {
	Name  string `json:"name" yaml:"name"`
	Value string `json:"value" yaml:"value"`
}

// ContainerPort represents a port exposed by a container
type ContainerPort struct {
	ContainerPort int32  `json:"containerPort" yaml:"containerPort"`
	Protocol      string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
}

// PodStatus describes the current state of a pod
type PodStatus struct {
	Phase            PodPhase          `json:"phase" yaml:"phase"`
	Conditions       []PodCondition    `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	Message          string            `json:"message,omitempty" yaml:"message,omitempty"`
	Reason           string            `json:"reason,omitempty" yaml:"reason,omitempty"`
	HostIP           string            `json:"hostIP,omitempty" yaml:"hostIP,omitempty"`
	PodIP            string            `json:"podIP,omitempty" yaml:"podIP,omitempty"`
	StartTime        *time.Time        `json:"startTime,omitempty" yaml:"startTime,omitempty"`
	ContainerStatuses []ContainerStatus `json:"containerStatuses,omitempty" yaml:"containerStatuses,omitempty"`
	RestartCount     int32             `json:"restartCount" yaml:"restartCount"`
}

// PodPhase represents the phase of a pod
type PodPhase string

const (
	PodPending   PodPhase = "Pending"
	PodRunning   PodPhase = "Running"
	PodSucceeded PodPhase = "Succeeded"
	PodFailed    PodPhase = "Failed"
	PodUnknown   PodPhase = "Unknown"
)

// PodCondition contains details for the current condition
type PodCondition struct {
	Type               string    `json:"type" yaml:"type"`
	Status             string    `json:"status" yaml:"status"`
	LastTransitionTime time.Time `json:"lastTransitionTime" yaml:"lastTransitionTime"`
	Reason             string    `json:"reason,omitempty" yaml:"reason,omitempty"`
	Message            string    `json:"message,omitempty" yaml:"message,omitempty"`
}

// ContainerStatus contains the status of a container
type ContainerStatus struct {
	Name         string    `json:"name" yaml:"name"`
	Ready        bool      `json:"ready" yaml:"ready"`
	RestartCount int32     `json:"restartCount" yaml:"restartCount"`
	State        string    `json:"state" yaml:"state"`
	ContainerID  string    `json:"containerID,omitempty" yaml:"containerID,omitempty"`
	LastProbeTime *time.Time `json:"lastProbeTime,omitempty" yaml:"lastProbeTime,omitempty"`
}

// Node represents a worker node in the cluster
type Node struct {
	Metadata Metadata   `json:"metadata" yaml:"metadata"`
	Spec     NodeSpec   `json:"spec" yaml:"spec"`
	Status   NodeStatus `json:"status" yaml:"status"`
}

// NodeSpec describes node specifications
type NodeSpec struct {
	PodCIDR string `json:"podCIDR,omitempty" yaml:"podCIDR,omitempty"`
}

// NodeStatus describes the current state of a node
type NodeStatus struct {
	Capacity    ResourceList    `json:"capacity,omitempty" yaml:"capacity,omitempty"`
	Allocatable ResourceList    `json:"allocatable,omitempty" yaml:"allocatable,omitempty"`
	Conditions  []NodeCondition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	Addresses   []NodeAddress   `json:"addresses,omitempty" yaml:"addresses,omitempty"`
	Phase       NodePhase       `json:"phase" yaml:"phase"`
}

// ResourceList is a map of resource name to quantity
type ResourceList map[string]string

// NodeCondition contains condition information for a node
type NodeCondition struct {
	Type               string    `json:"type" yaml:"type"`
	Status             string    `json:"status" yaml:"status"`
	LastHeartbeatTime  time.Time `json:"lastHeartbeatTime" yaml:"lastHeartbeatTime"`
	LastTransitionTime time.Time `json:"lastTransitionTime" yaml:"lastTransitionTime"`
	Reason             string    `json:"reason,omitempty" yaml:"reason,omitempty"`
	Message            string    `json:"message,omitempty" yaml:"message,omitempty"`
}

// NodeAddress contains information for the node's address
type NodeAddress struct {
	Type    string `json:"type" yaml:"type"`
	Address string `json:"address" yaml:"address"`
}

// NodePhase represents the phase of a node
type NodePhase string

const (
	NodePending    NodePhase = "Pending"
	NodeRunning    NodePhase = "Running"
	NodeTerminated NodePhase = "Terminated"
)

// Event represents an event in the cluster
type Event struct {
	Type      string    `json:"type" yaml:"type"`
	Reason    string    `json:"reason" yaml:"reason"`
	Message   string    `json:"message" yaml:"message"`
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
	Object    string    `json:"object" yaml:"object"`
}