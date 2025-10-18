package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"kubico/types"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var apiServerURL string

func getReplicaSets(cmd *cobra.Command, args []string) {
	if len(args) > 0 {
		// Get specific replicaset
		rs, err := fetchReplicaSet(args[0])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		printReplicaSet(rs)
	} else {
		// List all replicasets
		replicaSets, err := fetchReplicaSets()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		printReplicaSetList(replicaSets)
	}
}

// fetchReplicaSets fetches all replicasets from the API server
func fetchReplicaSets() ([]*types.ReplicaSet, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/replicasets", apiServerURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []*types.ReplicaSet `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Items, nil
}

// fetchReplicaSet fetches a specific replicaset
func fetchReplicaSet(name string) (*types.ReplicaSet, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/replicasets/%s", apiServerURL, name))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("replicaset not found")
	}

	var rs types.ReplicaSet
	if err := json.NewDecoder(resp.Body).Decode(&rs); err != nil {
		return nil, err
	}

	return &rs, nil
}

// printReplicaSetList prints a list of replicasets in table format
func printReplicaSetList(replicaSets []*types.ReplicaSet) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tDESIRED\tCURRENT\tREADY\tAGE")

	for _, rs := range replicaSets {
		age := time.Since(rs.Metadata.CreatedAt).Round(time.Second)
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%s\n", 
			rs.Metadata.Name, 
			rs.Spec.Replicas, 
			rs.Status.Replicas, 
			rs.Status.ReadyReplicas, 
			age)
	}

	w.Flush()
}


// printReplicaSet prints a single replicaset
func printReplicaSet(rs *types.ReplicaSet) {
	fmt.Printf("Name: %s\n", rs.Metadata.Name)
	fmt.Printf("Namespace: %s\n", rs.Metadata.Namespace)
	fmt.Printf("Desired Replicas: %d\n", rs.Spec.Replicas)
	fmt.Printf("Current Replicas: %d\n", rs.Status.Replicas)
	fmt.Printf("Ready Replicas: %d\n", rs.Status.ReadyReplicas)
	fmt.Printf("Created: %s\n", rs.Metadata.CreatedAt.Format(time.RFC3339))
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "minictl",
		Short: "Mini Kubernetes CLI",
		Long:  "Command line tool for interacting with Mini Kubernetes cluster",
	}

	rootCmd.PersistentFlags().StringVar(&apiServerURL, "server", "http://localhost:8080", "API server URL")

	// Add subcommands
	rootCmd.AddCommand(getCmd())
	rootCmd.AddCommand(applyCmd())
	rootCmd.AddCommand(deleteCmd())
	rootCmd.AddCommand(describeCmd())
	rootCmd.AddCommand(logsCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// getCmd creates the 'get' command
func getCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [resource]",
		Short: "Display one or many resources",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "pods [NAME]",
		Short: "Get pods",
		Run:   getPods,
	})

	cmd.AddCommand(&cobra.Command{
		Use:     "replicasets [NAME]",
		Aliases: []string{"rs"},
		Short:   "Get replicasets",
		Run:     getReplicaSets,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "nodes [NAME]",
		Short: "Get nodes",
		Run:   getNodes,
	})

	return cmd
}


// applyCmd creates the 'apply' command
func applyCmd() *cobra.Command {
	var filename string

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply a configuration to a resource",
		Run: func(cmd *cobra.Command, args []string) {
			applyConfig(filename)
		},
	}

	cmd.Flags().StringVarP(&filename, "filename", "f", "", "Filename containing the resource to apply")
	cmd.MarkFlagRequired("filename")

	return cmd
}

// deleteCmd creates the 'delete' command
func deleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [resource] [name]",
		Short: "Delete resources",
		Args:  cobra.ExactArgs(2),
		Run:   deleteResource,
	}
}

// describeCmd creates the 'describe' command
func describeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "describe [resource] [name]",
		Short: "Show details of a specific resource",
		Args:  cobra.ExactArgs(2),
		Run:   describeResource,
	}
}

// logsCmd creates the 'logs' command
func logsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logs [pod-name]",
		Short: "Print the logs for a container in a pod",
		Args:  cobra.ExactArgs(1),
		Run:   getLogs,
	}
}

// getPods handles 'get pods' command
func getPods(cmd *cobra.Command, args []string) {
	if len(args) > 0 {
		// Get specific pod
		pod, err := fetchPod(args[0])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		printPod(pod)
	} else {
		// List all pods
		pods, err := fetchPods()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		printPodList(pods)
	}
}

// getNodes handles 'get nodes' command
func getNodes(cmd *cobra.Command, args []string) {
	if len(args) > 0 {
		// Get specific node
		node, err := fetchNode(args[0])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		printNode(node)
	} else {
		// List all nodes
		nodes, err := fetchNodes()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		printNodeList(nodes)
	}
}

// applyConfig applies a configuration from a file
func applyConfig(filename string) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		return
	}

	// Build absolute file path
	filePath := filename
	if !filepath.IsAbs(filename) {
		filePath = filepath.Join(cwd, filename)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	var resource struct {
		Kind string `yaml:"kind"`
	}
	if err := yaml.Unmarshal(data, &resource); err != nil {
		fmt.Printf("Error parsing YAML: %v\n", err)
		return
	}
	

	switch resource.Kind {
	case "Pod":
		applyPod(data)
	case "ReplicaSet":
		applyReplicaSet(data)
	default:
		fmt.Printf("Unknown resource kind: %s\n", resource.Kind)
	}
}

func applyReplicaSet(data []byte) {
    // Unmarshal YAML into a ReplicaSet struct
    var rs types.ReplicaSet
    if err := yaml.Unmarshal(data, &rs); err != nil {
        fmt.Printf("Error parsing YAML: %v\n", err)
        return
    }

    // Marshal to JSON for sending to API
    jsonData, err := json.Marshal(rs)
    if err != nil {
        fmt.Printf("Error marshaling JSON: %v\n", err)
        return
    }

    replicaSetName := rs.Metadata.Name
    // Check if ReplicaSet already exists
    resp, err := http.Get(fmt.Sprintf("%s/api/v1/replicasets/%s", apiServerURL, replicaSetName))
    if err != nil {
        fmt.Printf("Error checking replicaset existence: %v\n", err)
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusOK {
        // ReplicaSet exists → call update API
       client := &http.Client{}
		req, err := http.NewRequest(
			http.MethodPut,
			fmt.Sprintf("%s/api/v1/replicasets/%s", apiServerURL, replicaSetName),
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("Error creating PUT request: %v\n", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		updateResp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error updating replicaset: %v\n", err)
			return
		}
		defer updateResp.Body.Close()

		if updateResp.StatusCode == http.StatusOK {
			fmt.Printf("ReplicaSet %s updated successfully\n", replicaSetName)
		} else {
			body, _ := io.ReadAll(updateResp.Body)
			fmt.Printf("Update error: %s\n", string(body))
		}
		return;
    }else{
	// If not found → call create API
		createResp, err := http.Post(
			fmt.Sprintf("%s/api/v1/replicasets", apiServerURL),
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("Error creating replicaset: %v\n", err)
			return
		}
		defer createResp.Body.Close()

		if createResp.StatusCode == http.StatusCreated {
			fmt.Printf("ReplicaSet %s created successfully\n", replicaSetName)
		} else {
			body, _ := io.ReadAll(createResp.Body)
			fmt.Printf("Create error: %s\n", string(body))
		}
	}

   
}

func applyPod(data []byte) {
	var pod types.Pod
	
	if err := yaml.Unmarshal(data, &pod); err != nil {
		fmt.Printf("Error parsing YAML: %v\n", err)
		return
	}

	jsonData, err := json.Marshal(pod)
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}
	resp, err := http.Post(
		fmt.Sprintf("%s/api/v1/pods", apiServerURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		fmt.Printf("Error creating pod: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		fmt.Printf("Pod %s created successfully\n", pod.Metadata.Name)
	} else {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Error: %s\n", string(body))
	}
}

// applyReplicaSet applies a replicaset configuration
func applyReplicaSet2(data []byte) {
	var pod types.ReplicaSet
	if err := yaml.Unmarshal(data, &pod); err != nil {
		fmt.Printf("Error parsing YAML: %v\n", err)
		return
	}
	jsonData, err := json.Marshal(pod)
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}

	resp, err := http.Post(
		fmt.Sprintf("%s/api/v1/replicasets", apiServerURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		fmt.Printf("Error creating replicaset: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		fmt.Printf("ReplicaSet %s created successfully\n",pod.Metadata.Name)
	} else {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Error: %s\n", string(body))
	}
}

// deleteResource deletes a resource
func deleteResource(cmd *cobra.Command, args []string) {
	resource := args[0]
	name := args[1]

	var url string
	switch resource {
	case "pod", "pods":
		url = fmt.Sprintf("%s/api/v1/pods/%s", apiServerURL, name)
	case "replicaset", "replicasets", "rs":
		url = fmt.Sprintf("%s/api/v1/replicasets/%s", apiServerURL, name)
	case "node", "nodes":
		url = fmt.Sprintf("%s/api/v1/nodes/%s", apiServerURL, name)
	default:
		fmt.Printf("Unknown resource type: %s\n", resource)
		return
	}

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error deleting resource: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("%s %s deleted successfully\n", resource, name)
	} else {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Error: %s\n", string(body))
	}
}

// describeResource shows detailed information about a resource
func describeResource(cmd *cobra.Command, args []string) {
	resource := args[0]
	name := args[1]

	switch resource {
	case "pod", "pods":
		pod, err := fetchPod(name)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		printPodDetails(pod)
	case "replicaset", "replicasets", "rs":
		_, err := fetchReplicaSet(name)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println("TODO:Need to implement replica set describe")
		// printReplicaSetDetails(rs)
	case "node", "nodes":
		node, err := fetchNode(name)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		printNodeDetails(node)
	default:
		fmt.Printf("Unknown resource type: %s\n", resource)
	}
}

// getLogs fetches and prints pod logs
func getLogs(cmd *cobra.Command, args []string) {
	podName := args[0]
	fmt.Printf("Logs for pod %s:\n", podName)
	fmt.Println("(Log fetching from containers not fully implemented in this basic version)")
	fmt.Println("In production, this would fetch container logs via Docker API")
}

// fetchPods fetches all pods from the API server
func fetchPods() ([]*types.Pod, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/pods", apiServerURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []*types.Pod `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Items, nil
}

// fetchPod fetches a specific pod
func fetchPod(name string) (*types.Pod, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/pods/%s", apiServerURL, name))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pod not found")
	}

	var pod types.Pod
	if err := json.NewDecoder(resp.Body).Decode(&pod); err != nil {
		return nil, err
	}

	return &pod, nil
}

// fetchNodes fetches all nodes from the API server
func fetchNodes() ([]*types.Node, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/nodes", apiServerURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []*types.Node `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Items, nil
}

// fetchNode fetches a specific node
func fetchNode(name string) (*types.Node, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/nodes/%s", apiServerURL, name))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("node not found")
	}

	var node types.Node
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return nil, err
	}

	return &node, nil
}

// printPodList prints a list of pods in table format
func printPodList(pods []*types.Pod) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tNODE\tAGE")

	for _, pod := range pods {
		age := time.Since(pod.Metadata.CreatedAt).Round(time.Second)
		node := pod.Spec.NodeName
		if node == "" {
			node = "<none>"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", pod.Metadata.Name, pod.Status.Phase, node, age)
	}

	w.Flush()
}

// printPod prints a single pod
func printPod(pod *types.Pod) {
	fmt.Printf("Name: %s\n", pod.Metadata.Name)
	fmt.Printf("Status: %s\n", pod.Status.Phase)
	fmt.Printf("Node: %s\n", pod.Spec.NodeName)
	fmt.Printf("Created: %s\n", pod.Metadata.CreatedAt.Format(time.RFC3339))
}

// printPodDetails prints detailed pod information
func printPodDetails(pod *types.Pod) {
	fmt.Printf("Name:         %s\n", pod.Metadata.Name)
	fmt.Printf("Namespace:    %s\n", pod.Metadata.Namespace)
	fmt.Printf("Status:       %s\n", pod.Status.Phase)
	fmt.Printf("Node:         %s\n", pod.Spec.NodeName)
	fmt.Printf("Created:      %s\n", pod.Metadata.CreatedAt.Format(time.RFC3339))
	
	if pod.Status.Message != "" {
		fmt.Printf("Message:      %s\n", pod.Status.Message)
	}

	fmt.Println("\nContainers:")
	for _, container := range pod.Spec.Containers {
		fmt.Printf("  %s:\n", container.Name)
		fmt.Printf("    Image: %s\n", container.Image)
		if len(container.Ports) > 0 {
			fmt.Printf("    Ports: ")
			for i, port := range container.Ports {
				if i > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%d", port.ContainerPort)
			}
			fmt.Println()
		}
	}
}

// printNodeList prints a list of nodes in table format
func printNodeList(nodes []*types.Node) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tAGE")

	for _, node := range nodes {
		age := time.Since(node.Metadata.CreatedAt).Round(time.Second)
		fmt.Fprintf(w, "%s\t%s\t%s\n", node.Metadata.Name, node.Status.Phase, age)
	}

	w.Flush()
}

// printNode prints a single node
func printNode(node *types.Node) {
	fmt.Printf("Name: %s\n", node.Metadata.Name)
	fmt.Printf("Status: %s\n", node.Status.Phase)
	fmt.Printf("Created: %s\n", node.Metadata.CreatedAt.Format(time.RFC3339))
}

// printNodeDetails prints detailed node information
func printNodeDetails(node *types.Node) {
	fmt.Printf("Name:         %s\n", node.Metadata.Name)
	fmt.Printf("Status:       %s\n", node.Status.Phase)
	fmt.Printf("Created:      %s\n", node.Metadata.CreatedAt.Format(time.RFC3339))

	if len(node.Status.Addresses) > 0 {
		fmt.Println("\nAddresses:")
		for _, addr := range node.Status.Addresses {
			fmt.Printf("  %s: %s\n", addr.Type, addr.Address)
		}
	}

	if len(node.Status.Conditions) > 0 {
		fmt.Println("\nConditions:")
		for _, cond := range node.Status.Conditions {
			fmt.Printf("  %s: %s\n", cond.Type, cond.Status)
			if cond.Message != "" {
				fmt.Printf("    Message: %s\n", cond.Message)
			}
		}
	}
}