package memory

import (
	"context"
	"fmt"
	"github.com/doc-ai/tensorio-models/storage"
	"sort"
	"strings"
	"sync"
)

type memory struct {
	lock      *sync.RWMutex
	modelList []string
	models    map[string]storage.Model

	hyperparametersList []string
	hyperparameters     map[string]storage.Hyperparameters

	checkpointsList []string
	checkpoints     map[string]storage.Checkpoint
}

func NewMemoryRepositoryStorage() storage.RepositoryStorage {
	store := &memory{
		lock: &sync.RWMutex{},

		modelList: make([]string, 0),
		models:    make(map[string]storage.Model),

		hyperparametersList: make([]string, 0),
		hyperparameters:     make(map[string]storage.Hyperparameters),

		checkpointsList: make([]string, 0),
		checkpoints:     make(map[string]storage.Checkpoint),
	}
	return store
}

func (s *memory) ListModels(ctx context.Context, marker string, maxItems int) ([]string, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	firstIndex := sort.SearchStrings(s.modelList, marker)
	lastIndex := firstIndex + maxItems
	if lastIndex > len(s.modelList) {
		lastIndex = len(s.modelList)
	}

	unsafeSlice := s.modelList[firstIndex:lastIndex]
	safeSlice := make([]string, len(unsafeSlice))
	copy(safeSlice, unsafeSlice)

	return safeSlice, nil
}

func (s *memory) GetModel(ctx context.Context, modelId string) (storage.Model, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	model, ok := s.models[modelId]
	if ok {
		return model, nil
	}
	return model, storage.ModelDoesNotExistError
}

func (s *memory) AddModel(ctx context.Context, model storage.Model) error {
	if _, err := s.GetModel(ctx, model.ModelId); err == nil {
		return storage.ModelExistsError
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	s.modelList = insert(s.modelList, model.ModelId)
	s.models[model.ModelId] = model

	return nil
}

func (s *memory) UpdateModel(ctx context.Context, model storage.Model) (storage.Model, error) {
	var currentModel storage.Model
	var err error
	if currentModel, err = s.GetModel(ctx, model.ModelId); err != nil {
		return storage.Model{}, err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	if strings.TrimSpace(model.Details) != "" {
		currentModel.Details = model.Details
	}
	if strings.TrimSpace(model.CanonicalHyperparameters) != "" {
		currentModel.CanonicalHyperparameters = model.CanonicalHyperparameters
	}

	s.models[currentModel.ModelId] = currentModel

	return s.models[model.ModelId], nil
}

func (s *memory) ListHyperparameters(ctx context.Context, modelId, marker string, maxItems int) ([]string, error) {
	if _, err := s.GetModel(ctx, modelId); err != nil {
		return nil, err
	}

	s.lock.RLock()
	defer s.lock.RUnlock()

	qualifiedMarker := fmt.Sprintf("%s:%s", modelId, marker)

	firstIndex := sort.SearchStrings(s.hyperparametersList, qualifiedMarker)
	lastIndex := firstIndex + maxItems
	if lastIndex > len(s.hyperparametersList) {
		lastIndex = len(s.hyperparametersList)
	}

	unsafeSlice := s.hyperparametersList[firstIndex:lastIndex]
	safeSlice := make([]string, 0, len(unsafeSlice))

	for i := 0; i < len(unsafeSlice); i++ {
		if strings.HasPrefix(unsafeSlice[i], modelId+":") {
			safeSlice = append(safeSlice, unsafeSlice[i])
		}
	}

	return safeSlice, nil
}

func (s *memory) GetHyperparameters(ctx context.Context, modelId string, hyperparametersId string) (storage.Hyperparameters, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	key := fmt.Sprintf("%s:%s", modelId, hyperparametersId)
	if hyperparameters, ok := s.hyperparameters[key]; ok {
		return hyperparameters, nil
	}

	return storage.Hyperparameters{}, storage.HyperparametersDoesNotExistError
}

func (s *memory) AddHyperparameters(ctx context.Context, hyperparameters storage.Hyperparameters) error {
	if _, err := s.GetModel(ctx, hyperparameters.ModelId); err != nil {
		return err
	}

	if _, err := s.GetHyperparameters(ctx, hyperparameters.ModelId, hyperparameters.HyperparametersId); err == nil {
		return storage.HyperparametersExistsError
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	key := fmt.Sprintf("%s:%s", hyperparameters.ModelId, hyperparameters.HyperparametersId)

	s.hyperparametersList = insert(s.hyperparametersList, key)
	s.hyperparameters[key] = hyperparameters

	return nil
}

func (s *memory) UpdateHyperparameters(ctx context.Context, hyperparameters storage.Hyperparameters) (storage.Hyperparameters, error) {
	if _, err := s.GetModel(ctx, hyperparameters.ModelId); err != nil {
		return storage.Hyperparameters{}, err
	}

	if _, err := s.GetHyperparameters(ctx, hyperparameters.ModelId, hyperparameters.HyperparametersId); err != nil {
		return storage.Hyperparameters{}, err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	key := fmt.Sprintf("%s:%s", hyperparameters.ModelId, hyperparameters.HyperparametersId)

	currentHyperparameters, _ := s.hyperparameters[key]

	if strings.TrimSpace(hyperparameters.CanonicalCheckpoint) != "" {
		currentHyperparameters.CanonicalCheckpoint = hyperparameters.CanonicalCheckpoint
	}

	if strings.TrimSpace(hyperparameters.UpgradeTo) != "" {
		currentHyperparameters.UpgradeTo = hyperparameters.UpgradeTo
	}

	if hyperparameters.Hyperparameters != nil {
		for k, v := range hyperparameters.Hyperparameters {
			if strings.TrimSpace(v) != "" {
				currentHyperparameters.Hyperparameters[k] = v
			}
		}
	}

	s.hyperparameters[key] = currentHyperparameters

	return currentHyperparameters, nil
}

func (s *memory) ListCheckpoints(ctx context.Context, modelId, hyperparametersId, marker string, maxItems int) ([]string, error) {
	if _, err := s.GetModel(ctx, modelId); err != nil {
		return nil, err
	}

	if _, err := s.GetHyperparameters(ctx, modelId, hyperparametersId); err != nil {
		return nil, err
	}

	s.lock.RLock()
	defer s.lock.RUnlock()

	qualifiedMarker := fmt.Sprintf("%s:%s:%s", modelId, hyperparametersId, marker)

	firstIndex := sort.SearchStrings(s.checkpointsList, qualifiedMarker)
	lastIndex := firstIndex + maxItems
	if lastIndex > len(s.checkpointsList) {
		lastIndex = len(s.checkpointsList)
	}

	unsafeSlice := s.checkpointsList[firstIndex:lastIndex]
	safeSlice := make([]string, 0, len(unsafeSlice))

	for i := 0; i < len(unsafeSlice); i++ {
		if strings.HasPrefix(unsafeSlice[i], modelId+":"+hyperparametersId+":") {
			safeSlice = append(safeSlice, unsafeSlice[i])
		}
	}

	return safeSlice, nil
}

func (s *memory) GetCheckpoint(ctx context.Context, modelId, hyperparametersId, checkpointId string) (storage.Checkpoint, error) {
	if _, err := s.GetModel(ctx, modelId); err != nil {
		return storage.Checkpoint{}, err
	}

	if _, err := s.GetHyperparameters(ctx, modelId, hyperparametersId); err != nil {
		return storage.Checkpoint{}, err
	}

	s.lock.RLock()
	defer s.lock.RUnlock()

	key := fmt.Sprintf("%s:%s:%s", modelId, hyperparametersId, checkpointId)

	if checkpoint, ok := s.checkpoints[key]; ok {
		return checkpoint, nil
	}

	return storage.Checkpoint{}, storage.CheckpointDoesNotExistError
}

func (s *memory) AddCheckpoint(ctx context.Context, checkpoint storage.Checkpoint) error {
	if _, err := s.GetModel(ctx, checkpoint.ModelId); err != nil {
		return err
	}

	if _, err := s.GetHyperparameters(ctx, checkpoint.ModelId, checkpoint.HyperparametersId); err != nil {
		return err
	}

	if _, err := s.GetCheckpoint(ctx, checkpoint.ModelId, checkpoint.HyperparametersId, checkpoint.CheckpointId); err == nil {
		return storage.CheckpointExistsError
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	key := fmt.Sprintf("%s:%s:%s", checkpoint.ModelId, checkpoint.HyperparametersId, checkpoint.CheckpointId)

	s.checkpointsList = insert(s.checkpointsList, key)
	s.checkpoints[key] = checkpoint

	return nil
}

func insert(list []string, id string) []string {
	list = append(list, id)
	sort.Strings(list)
	return list
}
