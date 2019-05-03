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

	hyperParametersList []string
	hyperParameters     map[string]storage.HyperParameters

	checkpointsList []string
	checkpoints     map[string]storage.Checkpoint
}

func NewMemoryRepositoryStorage() storage.RepositoryStorage {
	store := &memory{
		lock: &sync.RWMutex{},

		modelList: make([]string, 0),
		models:    make(map[string]storage.Model),

		hyperParametersList: make([]string, 0),
		hyperParameters:     make(map[string]storage.HyperParameters),

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

	if strings.TrimSpace(model.Description) != "" {
		currentModel.Description = model.Description
	}
	if strings.TrimSpace(model.CanonicalHyperParameters) != "" {
		currentModel.CanonicalHyperParameters = model.CanonicalHyperParameters
	}

	s.models[currentModel.ModelId] = currentModel

	return s.models[model.ModelId], nil
}

func (s *memory) ListHyperParameters(ctx context.Context, modelId, marker string, maxItems int) ([]string, error) {
	if _, err := s.GetModel(ctx, modelId); err != nil {
		return nil, err
	}

	s.lock.RLock()
	defer s.lock.RUnlock()

	qualifiedMarker := fmt.Sprintf("%s:%s", modelId, marker)

	firstIndex := sort.SearchStrings(s.hyperParametersList, qualifiedMarker)
	lastIndex := firstIndex + maxItems
	if lastIndex > len(s.hyperParametersList) {
		lastIndex = len(s.hyperParametersList)
	}

	unsafeSlice := s.hyperParametersList[firstIndex:lastIndex]
	safeSlice := make([]string, 0, len(unsafeSlice))

	for i := 0; i < len(unsafeSlice); i++ {
		if strings.HasPrefix(unsafeSlice[i], modelId+":") {
			safeSlice = append(safeSlice, unsafeSlice[i])
		}
	}

	return safeSlice, nil
}

func (s *memory) GetHyperParameters(ctx context.Context, modelId string, hyperParametersId string) (storage.HyperParameters, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	key := fmt.Sprintf("%s:%s", modelId, hyperParametersId)
	if hyperParameters, ok := s.hyperParameters[key]; ok {
		return hyperParameters, nil
	}

	return storage.HyperParameters{}, storage.HyperParametersDoesNotExistError
}

func (s *memory) AddHyperParameters(ctx context.Context, hyperParameters storage.HyperParameters) error {
	if _, err := s.GetModel(ctx, hyperParameters.ModelId); err != nil {
		return err
	}

	if _, err := s.GetHyperParameters(ctx, hyperParameters.ModelId, hyperParameters.HyperParametersId); err == nil {
		return storage.HyperParametersExistsError
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	key := fmt.Sprintf("%s:%s", hyperParameters.ModelId, hyperParameters.HyperParametersId)

	s.hyperParametersList = insert(s.hyperParametersList, key)
	s.hyperParameters[key] = hyperParameters

	return nil
}

func (s *memory) UpdateHyperParameters(ctx context.Context, hyperParameters storage.HyperParameters) (storage.HyperParameters, error) {
	if _, err := s.GetModel(ctx, hyperParameters.ModelId); err != nil {
		return storage.HyperParameters{}, err
	}

	if _, err := s.GetHyperParameters(ctx, hyperParameters.ModelId, hyperParameters.HyperParametersId); err != nil {
		return storage.HyperParameters{}, err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	key := fmt.Sprintf("%s:%s", hyperParameters.ModelId, hyperParameters.HyperParametersId)

	currentHyperParameters, _ := s.hyperParameters[key]
	if strings.TrimSpace(hyperParameters.CanonicalCheckpoint) != "" {
		currentHyperParameters.CanonicalCheckpoint = hyperParameters.CanonicalCheckpoint
	}

	if hyperParameters.HyperParameters != nil {
		for k, v := range hyperParameters.HyperParameters {
			if strings.TrimSpace(v) != "" {
				currentHyperParameters.HyperParameters[k] = v
			}
		}
	}

	s.hyperParameters[key] = currentHyperParameters

	return currentHyperParameters, nil
}

func (s *memory) ListCheckpoints(ctx context.Context, modelId, hyperParametersId, marker string, maxItems int) ([]string, error) {
	if _, err := s.GetModel(ctx, modelId); err != nil {
		return nil, err
	}

	if _, err := s.GetHyperParameters(ctx, modelId, hyperParametersId); err != nil {
		return nil, err
	}

	s.lock.RLock()
	defer s.lock.RUnlock()

	qualifiedMarker := fmt.Sprintf("%s:%s:%s", modelId, hyperParametersId, marker)

	firstIndex := sort.SearchStrings(s.checkpointsList, qualifiedMarker)
	lastIndex := firstIndex + maxItems
	if lastIndex > len(s.checkpointsList) {
		lastIndex = len(s.checkpointsList)
	}

	unsafeSlice := s.checkpointsList[firstIndex:lastIndex]
	safeSlice := make([]string, 0, len(unsafeSlice))

	for i := 0; i < len(unsafeSlice); i++ {
		if strings.HasPrefix(unsafeSlice[i], modelId+":"+hyperParametersId+":") {
			safeSlice = append(safeSlice, unsafeSlice[i])
		}
	}

	return safeSlice, nil
}

func (s *memory) GetCheckpoint(ctx context.Context, modelId, hyperParametersId, checkpointId string) (storage.Checkpoint, error) {
	if _, err := s.GetModel(ctx, modelId); err != nil {
		return storage.Checkpoint{}, err
	}

	if _, err := s.GetHyperParameters(ctx, modelId, hyperParametersId); err != nil {
		return storage.Checkpoint{}, err
	}

	s.lock.RLock()
	defer s.lock.RUnlock()

	key := fmt.Sprintf("%s:%s:%s", modelId, hyperParametersId, checkpointId)

	if checkpoint, ok := s.checkpoints[key]; ok {
		return checkpoint, nil
	}

	return storage.Checkpoint{}, storage.CheckpointDoesNotExistError
}

func (s *memory) AddCheckpoint(ctx context.Context, checkpoint storage.Checkpoint) error {
	if _, err := s.GetModel(ctx, checkpoint.ModelId); err != nil {
		return err
	}

	if _, err := s.GetHyperParameters(ctx, checkpoint.ModelId, checkpoint.HyperParametersId); err != nil {
		return err
	}

	if _, err := s.GetCheckpoint(ctx, checkpoint.ModelId, checkpoint.HyperParametersId, checkpoint.CheckpointId); err == nil {
		return storage.CheckpointExistsError
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	key := fmt.Sprintf("%s:%s:%s", checkpoint.ModelId, checkpoint.HyperParametersId, checkpoint.CheckpointId)

	s.checkpointsList = insert(s.checkpointsList, key)
	s.checkpoints[key] = checkpoint

	return nil
}

func insert(list []string, id string) []string {
	list = append(list, id)
	sort.Strings(list)
	return list
}
