package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"kubico/types"

	_ "github.com/lib/pq"
)

type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgreSQL-backed store
func NewPostgresStore(connStr string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &PostgresStore{db: db}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// initSchema creates the necessary tables
func (s *PostgresStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS pods (
		name VARCHAR(255) PRIMARY KEY,
		namespace VARCHAR(255) NOT NULL DEFAULT 'default',
		labels JSONB,
		spec JSONB NOT NULL,
		status JSONB NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
		uid VARCHAR(255),
		owner_references JSONB
	);

	CREATE TABLE IF NOT EXISTS replica_sets (
		name VARCHAR(255) PRIMARY KEY,
		namespace VARCHAR(255) NOT NULL DEFAULT 'default',
		labels JSONB,
		spec JSONB NOT NULL,
		status JSONB NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
		uid VARCHAR(255)
	);

	CREATE TABLE IF NOT EXISTS nodes (
		name VARCHAR(255) PRIMARY KEY,
		spec JSONB,
		status JSONB NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_pods_labels ON pods USING GIN (labels);
	CREATE INDEX IF NOT EXISTS idx_pods_node ON pods ((spec->>'nodeName'));
	CREATE INDEX IF NOT EXISTS idx_pods_phase ON pods ((status->>'phase'));
	CREATE INDEX IF NOT EXISTS idx_replica_sets_labels ON replica_sets USING GIN (labels);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Pod operations

func (s *PostgresStore) CreatePod(pod *types.Pod) error {
	specJSON, _ := json.Marshal(pod.Spec)
	statusJSON, _ := json.Marshal(pod.Status)
	labelsJSON, _ := json.Marshal(pod.Metadata.Labels)
	ownerRefsJSON, _ := json.Marshal(pod.Metadata.OwnerReferences)

	_, err := s.db.Exec(`
		INSERT INTO pods (name, namespace, labels, spec, status, created_at, uid, owner_references)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, pod.Metadata.Name, pod.Metadata.Namespace, labelsJSON, specJSON, statusJSON,
		pod.Metadata.CreatedAt, pod.Metadata.UID, ownerRefsJSON)

	if err != nil {
		return fmt.Errorf("failed to create pod: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetPod(name string) (*types.Pod, error) {
	var pod types.Pod
	var specJSON, statusJSON, labelsJSON, ownerRefsJSON []byte

	err := s.db.QueryRow(`
		SELECT name, namespace, labels, spec, status, created_at, uid, owner_references
		FROM pods WHERE name = $1
	`, name).Scan(
		&pod.Metadata.Name,
		&pod.Metadata.Namespace,
		&labelsJSON,
		&specJSON,
		&statusJSON,
		&pod.Metadata.CreatedAt,
		&pod.Metadata.UID,
		&ownerRefsJSON,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(labelsJSON, &pod.Metadata.Labels)
	json.Unmarshal(specJSON, &pod.Spec)
	json.Unmarshal(statusJSON, &pod.Status)
	json.Unmarshal(ownerRefsJSON, &pod.Metadata.OwnerReferences)

	pod.APIVersion = "v1"
	pod.Kind = "Pod"

	return &pod, nil
}

func (s *PostgresStore) ListPods() ([]*types.Pod, error) {
	rows, err := s.db.Query(`
		SELECT name, namespace, labels, spec, status, created_at, uid, owner_references
		FROM pods ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pods := make([]*types.Pod, 0)
	for rows.Next() {
		var pod types.Pod
		var specJSON, statusJSON, labelsJSON, ownerRefsJSON []byte

		err := rows.Scan(
			&pod.Metadata.Name,
			&pod.Metadata.Namespace,
			&labelsJSON,
			&specJSON,
			&statusJSON,
			&pod.Metadata.CreatedAt,
			&pod.Metadata.UID,
			&ownerRefsJSON,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(labelsJSON, &pod.Metadata.Labels)
		json.Unmarshal(specJSON, &pod.Spec)
		json.Unmarshal(statusJSON, &pod.Status)
		json.Unmarshal(ownerRefsJSON, &pod.Metadata.OwnerReferences)

		pod.APIVersion = "v1"
		pod.Kind = "Pod"

		pods = append(pods, &pod)
	}

	return pods, nil
}

func (s *PostgresStore) UpdatePod(pod *types.Pod) error {
	specJSON, _ := json.Marshal(pod.Spec)
	statusJSON, _ := json.Marshal(pod.Status)
	labelsJSON, _ := json.Marshal(pod.Metadata.Labels)
	ownerRefsJSON, _ := json.Marshal(pod.Metadata.OwnerReferences)

	result, err := s.db.Exec(`
		UPDATE pods 
		SET spec = $1, status = $2, labels = $3, updated_at = $4, owner_references = $5
		WHERE name = $6
	`, specJSON, statusJSON, labelsJSON, time.Now(), ownerRefsJSON, pod.Metadata.Name)

	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
		
	}

	return nil
}

func (s *PostgresStore) DeletePod(name string) error {
	result, err := s.db.Exec("DELETE FROM pods WHERE name = $1", name)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// GetPodsByLabels returns pods matching the label selector
func (s *PostgresStore) GetPodsByLabels(labels map[string]string) ([]*types.Pod, error) {
	labelsJSON, _ := json.Marshal(labels)

	rows, err := s.db.Query(`
		SELECT name, namespace, labels, spec, status, created_at, uid, owner_references
		FROM pods 
		WHERE labels @> $1
		ORDER BY created_at DESC
	`, labelsJSON)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pods := make([]*types.Pod, 0)
	for rows.Next() {
		var pod types.Pod
		var specJSON, statusJSON, labelsJSON, ownerRefsJSON []byte

		err := rows.Scan(
			&pod.Metadata.Name,
			&pod.Metadata.Namespace,
			&labelsJSON,
			&specJSON,
			&statusJSON,
			&pod.Metadata.CreatedAt,
			&pod.Metadata.UID,
			&ownerRefsJSON,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(labelsJSON, &pod.Metadata.Labels)
		json.Unmarshal(specJSON, &pod.Spec)
		json.Unmarshal(statusJSON, &pod.Status)
		json.Unmarshal(ownerRefsJSON, &pod.Metadata.OwnerReferences)

		pod.APIVersion = "v1"
		pod.Kind = "Pod"

		pods = append(pods, &pod)
	}

	return pods, nil
}

// ReplicaSet operations

func (s *PostgresStore) CreateReplicaSet(rs *types.ReplicaSet) error {
	specJSON, _ := json.Marshal(rs.Spec)
	statusJSON, _ := json.Marshal(rs.Status)
	labelsJSON, _ := json.Marshal(rs.Metadata.Labels)

	_, err := s.db.Exec(`
		INSERT INTO replica_sets (name, namespace, labels, spec, status, created_at, uid)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, rs.Metadata.Name, rs.Metadata.Namespace, labelsJSON, specJSON, statusJSON,
		rs.Metadata.CreatedAt, rs.Metadata.UID)

	if err != nil {
		return fmt.Errorf("failed to create replicaset: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetReplicaSet(name string) (*types.ReplicaSet, error) {
	var rs types.ReplicaSet
	var specJSON, statusJSON, labelsJSON []byte

	err := s.db.QueryRow(`
		SELECT name, namespace, labels, spec, status, created_at, uid
		FROM replica_sets WHERE name = $1
	`, name).Scan(
		&rs.Metadata.Name,
		&rs.Metadata.Namespace,
		&labelsJSON,
		&specJSON,
		&statusJSON,
		&rs.Metadata.CreatedAt,
		&rs.Metadata.UID,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(labelsJSON, &rs.Metadata.Labels)
	json.Unmarshal(specJSON, &rs.Spec)
	json.Unmarshal(statusJSON, &rs.Status)

	rs.APIVersion = "apps/v1"
	rs.Kind = "ReplicaSet"

	return &rs, nil
}

func (s *PostgresStore) ListReplicaSets() ([]*types.ReplicaSet, error) {
	rows, err := s.db.Query(`
		SELECT name, namespace, labels, spec, status, created_at, uid
		FROM replica_sets ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	replicaSets := make([]*types.ReplicaSet, 0)
	for rows.Next() {
		var rs types.ReplicaSet
		var specJSON, statusJSON, labelsJSON []byte

		err := rows.Scan(
			&rs.Metadata.Name,
			&rs.Metadata.Namespace,
			&labelsJSON,
			&specJSON,
			&statusJSON,
			&rs.Metadata.CreatedAt,
			&rs.Metadata.UID,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(labelsJSON, &rs.Metadata.Labels)
		json.Unmarshal(specJSON, &rs.Spec)
		json.Unmarshal(statusJSON, &rs.Status)

		rs.APIVersion = "apps/v1"
		rs.Kind = "ReplicaSet"

		replicaSets = append(replicaSets, &rs)
	}

	return replicaSets, nil
}

func (s *PostgresStore) UpdateReplicaSet(rs *types.ReplicaSet) error {
	specJSON, _ := json.Marshal(rs.Spec)
	statusJSON, _ := json.Marshal(rs.Status)
	labelsJSON, _ := json.Marshal(rs.Metadata.Labels)

	result, err := s.db.Exec(`
		UPDATE replica_sets 
		SET spec = $1, status = $2, labels = $3, updated_at = $4
		WHERE name = $5
	`, specJSON, statusJSON, labelsJSON, time.Now(), rs.Metadata.Name)

	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *PostgresStore) DeleteReplicaSet(name string) error {
	result, err := s.db.Exec("DELETE FROM replica_sets WHERE name = $1", name)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// Node operations (keeping existing implementation)

func (s *PostgresStore) CreateNode(node *types.Node) error {
	specJSON, _ := json.Marshal(node.Spec)
	statusJSON, _ := json.Marshal(node.Status)

	_, err := s.db.Exec(`
		INSERT INTO nodes (name, spec, status, created_at)
		VALUES ($1, $2, $3, $4)
	`, node.Metadata.Name, specJSON, statusJSON, node.Metadata.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create node: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetNode(name string) (*types.Node, error) {
	var node types.Node
	var specJSON, statusJSON []byte

	err := s.db.QueryRow(`
		SELECT name, spec, status, created_at
		FROM nodes WHERE name = $1
	`, name).Scan(
		&node.Metadata.Name,
		&specJSON,
		&statusJSON,
		&node.Metadata.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(specJSON, &node.Spec)
	json.Unmarshal(statusJSON, &node.Status)

	return &node, nil
}

func (s *PostgresStore) ListNodes() ([]*types.Node, error) {
	rows, err := s.db.Query(`
		SELECT name, spec, status, created_at
		FROM nodes ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := make([]*types.Node, 0)
	for rows.Next() {
		var node types.Node
		var specJSON, statusJSON []byte

		err := rows.Scan(
			&node.Metadata.Name,
			&specJSON,
			&statusJSON,
			&node.Metadata.CreatedAt,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(specJSON, &node.Spec)
		json.Unmarshal(statusJSON, &node.Status)

		nodes = append(nodes, &node)
	}

	return nodes, nil
}

func (s *PostgresStore) UpdateNode(node *types.Node) error {
	specJSON, _ := json.Marshal(node.Spec)
	statusJSON, _ := json.Marshal(node.Status)

	result, err := s.db.Exec(`
		UPDATE nodes 
		SET spec = $1, status = $2, updated_at = $3
		WHERE name = $4
	`, specJSON, statusJSON, time.Now(), node.Metadata.Name)

	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *PostgresStore) DeleteNode(name string) error {
	result, err := s.db.Exec("DELETE FROM nodes WHERE name = $1", name)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}


// ReplicaSet operations