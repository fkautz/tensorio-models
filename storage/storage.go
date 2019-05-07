package storage

import (
	"context"
	"errors"
	"time"
)

var ModelDoesNotExistError = errors.New("model does not exist")
var ModelExistsError = errors.New("model already exists")

var HyperparametersDoesNotExistError = errors.New("hyperparameters does not exist")
var HyperparametersExistsError = errors.New("hyperparameters already exists")

var CheckpointDoesNotExistError = errors.New("checkpoint does not exist")
var CheckpointExistsError = errors.New("checkpoint already exists")

type Model struct {
	ModelId                  string
	Details                  string
	CanonicalHyperparameters string
}

type Hyperparameters struct {
	ModelId             string
	HyperparametersId   string
	CanonicalCheckpoint string
	UpgradeTo           string
	Hyperparameters     map[string]string
}

type Checkpoint struct {
	ModelId           string
	HyperparametersId string
	CheckpointId      string
	Link              string
	CreatedAt         time.Time
	Info              map[string]string
}

type RepositoryStorage interface {
	// MODELS

	ListModels(ctx context.Context, marker string, maxItems int) ([]string, error)
	GetModel(ctx context.Context, modelId string) (Model, error)

	AddModel(ctx context.Context, model Model) error
	UpdateModel(ctx context.Context, model Model) (Model, error)

	// HYPERPARAMETERS

	ListHyperparameters(ctx context.Context, modelId, marker string, maxItems int) ([]string, error)
	GetHyperparameters(ctx context.Context, modelId string, hyperparametersId string) (Hyperparameters, error)

	AddHyperparameters(ctx context.Context, hyperparameters Hyperparameters) error
	UpdateHyperparameters(ctx context.Context, hyperparameters Hyperparameters) (Hyperparameters, error)

	// CHECKPOINTS

	ListCheckpoints(ctx context.Context, modelId, hyperparametersId, marker string, maxItems int) ([]string, error)
	GetCheckpoint(ctx context.Context, modelId, hyperparametersId, checkpointId string) (Checkpoint, error)

	AddCheckpoint(ctx context.Context, checkpoint Checkpoint) error
}
