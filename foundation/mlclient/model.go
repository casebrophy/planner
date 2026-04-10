package mlclient

// TaskInput represents task data sent to the clustering endpoint.
type TaskInput struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
}

// ClusterResult represents the cluster assignment for a task.
type ClusterResult struct {
	TaskID    string `json:"task_id"`
	ClusterID string `json:"cluster_id"`
}

// SimilarMatch represents a similar situation found by the ML service.
type SimilarMatch struct {
	ContextID   string  `json:"context_id"`
	Similarity  float64 `json:"similarity"`
	Description string  `json:"description"`
}
