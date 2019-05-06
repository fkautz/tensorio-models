package server

import (
	"context"
	"fmt"

	"github.com/doc-ai/tensorio-models/api"
	"github.com/doc-ai/tensorio-models/storage"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	storage storage.RepositoryStorage
}

// NewServer - Creates an api.RepositoryServer which handles gRPC requests using a given
// storage.RepositoryStorage backend
func NewServer(storage storage.RepositoryStorage) api.RepositoryServer {
	return &server{storage: storage}
}

func (srv *server) Healthz(ctx context.Context, req *api.HealthCheckRequest) (*api.HealthCheckResponse, error) {
	log.Println(req)
	resp := &api.HealthCheckResponse{
		Status: 1,
	}
	return resp, nil
}

func (srv *server) ListModels(ctx context.Context, req *api.ListModelsRequest) (*api.ListModelsResponse, error) {
	marker := req.Marker
	maxItems := int(req.MaxItems)
	log.Printf("ListModels request - Marker: %s, MaxItems: %d", marker, maxItems)
	models, err := srv.storage.ListModels(ctx, marker, maxItems)
	if err != nil {
		log.Printf("ERROR: %v", err)
		grpcErr := status.Error(codes.Unavailable, "Could not retrieve models from storage")
		return nil, grpcErr
	}
	res := &api.ListModelsResponse{
		ModelIds: models,
	}
	return res, nil
}

func (srv *server) GetModel(ctx context.Context, req *api.GetModelRequest) (*api.GetModelResponse, error) {
	modelID := req.ModelId
	log.Printf("GetModel request - ModelId: %s", modelID)
	model, err := srv.storage.GetModel(ctx, modelID)
	if err != nil {
		log.Printf("ERROR: %v", err)
		message := fmt.Sprintf("Cloud not retrieve model (%s) from storage", modelID)
		grpcErr := status.Error(codes.Unavailable, message)
		return nil, grpcErr
	}
	respModel := api.Model{
		ModelId:                  model.ModelId,
		Description:              model.Description,
		CanonicalHyperParameters: model.CanonicalHyperParameters,
	}
	resp := &api.GetModelResponse{
		Model: &respModel,
	}
	return resp, nil
}

func (srv *server) CreateModel(ctx context.Context, req *api.CreateModelRequest) (*api.CreateModelResponse, error) {
	model := req.Model
	log.Printf("CreateModel request - Model: %v", model)
	storageModel := storage.Model{
		ModelId:                  model.ModelId,
		Description:              model.Description,
		CanonicalHyperParameters: model.CanonicalHyperParameters,
	}
	err := srv.storage.AddModel(ctx, storageModel)
	if err != nil {
		log.Printf("ERROR: %v", err)
		grpcErr := status.Error(codes.Unavailable, "Could not store model")
		return nil, grpcErr
	}
	resourcePath := fmt.Sprintf("/models/%s", storageModel.ModelId)
	resp := &api.CreateModelResponse{ResourcePath: resourcePath}
	return resp, nil
}

func (srv *server) UpdateModel(ctx context.Context, req *api.UpdateModelRequest) (*api.UpdateModelResponse, error) {
	modelID := req.ModelId
	model := req.Model
	if model == nil {
		return nil, api.MissingRequiredFieldError("model", "model to update").Err()
	}
	log.Printf("UpdateModel request - ModelId: %s, Model: %v", modelID, model)
	storedModel, err := srv.storage.GetModel(ctx, modelID)
	if err != nil {
		log.Printf("ERROR: %v", err)
		message := fmt.Sprintf("Cloud not retrieve model (%s) from storage", modelID)
		grpcErr := status.Error(codes.Unavailable, message)
		return nil, grpcErr
	}
	updatedModel := storedModel
	if model.Description != "" {
		updatedModel.Description = model.Description
	}
	if model.CanonicalHyperParameters != "" {
		updatedModel.CanonicalHyperParameters = model.CanonicalHyperParameters
	}
	newlyStoredModel, err := srv.storage.UpdateModel(ctx, updatedModel)
	if err != nil {
		log.Printf("ERROR: %v", err)
		message := fmt.Sprintf("Cloud not update model (%s) in storage", modelID)
		grpcErr := status.Error(codes.Unavailable, message)
		return nil, grpcErr
	}
	resp := &api.UpdateModelResponse{
		Model: &api.Model{
			ModelId:                  newlyStoredModel.ModelId,
			Description:              newlyStoredModel.Description,
			CanonicalHyperParameters: newlyStoredModel.CanonicalHyperParameters,
		},
	}
	return resp, nil
}

func (srv *server) ListHyperParameters(ctx context.Context, req *api.ListHyperParametersRequest) (*api.ListHyperParametersResponse, error) {
	modelID := req.ModelId
	marker := req.Marker
	maxItems := int(req.MaxItems)
	log.Printf("ListHyperParameters request - ModelId: %s, Marker: %s, MaxItems: %d", modelID, marker, maxItems)
	hyperparametersIDs, err := srv.storage.ListHyperParameters(ctx, modelID, marker, maxItems)
	if err != nil {
		log.Printf("ERROR: %v", err)
		message := fmt.Sprintf("Could not list hyperparameters for model (%s) in storage", modelID)
		grpcErr := status.Error(codes.Unavailable, message)
		return nil, grpcErr
	}
	resp := &api.ListHyperParametersResponse{
		HyperParametersIds: hyperparametersIDs,
	}
	return resp, nil
}

func (srv *server) CreateHyperParameters(ctx context.Context, req *api.CreateHyperParametersRequest) (*api.CreateHyperParametersResponse, error) {
	modelID := req.ModelId
	hyperparametersID := req.HyperParametersId
	canonicalCheckpoint := req.CanonicalCheckpoint
	hyperparameters := req.HyperParameters
	log.Printf("CreateHyperParameters request - ModelId: %s, HyperParametersId: %s, CanonicalCheckpoint: %s, HyperParameters: %v", modelID, hyperparametersID, canonicalCheckpoint, hyperparameters)
	storageHyperparameters := storage.HyperParameters{
		ModelId:             modelID,
		HyperParametersId:   hyperparametersID,
		CanonicalCheckpoint: canonicalCheckpoint,
		HyperParameters:     hyperparameters,
	}
	err := srv.storage.AddHyperParameters(ctx, storageHyperparameters)
	if err != nil {
		log.Printf("ERROR: %v", err)
		message := fmt.Sprintf("Could not store hyperparameters (%v) in storage", storageHyperparameters)
		grpcErr := status.Error(codes.Unavailable, message)
		return nil, grpcErr
	}
	resourcePath := fmt.Sprintf("/models/%s/hyperparameters/%s", modelID, hyperparametersID)
	resp := &api.CreateHyperParametersResponse{
		ResourcePath: resourcePath,
	}
	return resp, nil
}

func (srv *server) GetHyperParameters(ctx context.Context, req *api.GetHyperParametersRequest) (*api.GetHyperParametersResponse, error) {
	modelID := req.ModelId
	hyperparametersID := req.HyperParametersId
	log.Printf("GetHyperParameters request - ModelId: %s, HyperParametersId: %s", modelID, hyperparametersID)
	storedHyperparameters, err := srv.storage.GetHyperparameters(ctx, modelID, hyperparametersID)
	if err != nil {
		log.Printf("ERROR: %v", err)
		message := fmt.Sprintf("Could not get hyperparameters (%s) for model (%s) from storage", hyperparametersID, modelID)
		grpcErr := status.Error(codes.Unavailable, message)
		return nil, grpcErr
	}
	resp := &api.GetHyperParametersResponse{
		ModelId:             storedHyperparameters.ModelId,
		HyperParametersId:   storedHyperparameters.HyperParametersId,
		UpgradeTo:           "",
		CanonicalCheckpoint: storedHyperparameters.CanonicalCheckpoint,
		HyperParameters:     storedHyperparameters.HyperParameters,
	}
	return resp, nil
}

func (srv *server) UpdateHyperParameters(ctx context.Context, req *api.UpdateHyperParametersRequest) (*api.UpdateHyperParametersResponse, error) {
	modelID := req.ModelId
	hyperparametersID := req.HyperParametersId
	canonicalCheckpoint := req.CanonicalCheckpoint
	hyperparameters := req.HyperParameters
	log.Printf("UpdateHyperParameters request - ModelId: %s, HyperParametersId: %s, CanonicalCheckpoint: %s, HyperParameters: %v", modelID, hyperparametersID, canonicalCheckpoint, hyperparameters)

	existingHyperparameters, err := srv.storage.GetHyperparameters(ctx, modelID, hyperparametersID)
	if err != nil {
		log.Printf("ERROR: %v", err)
		message := fmt.Sprintf("Could not get hyperparameters (%s) for model (%s) from storage", hyperparametersID, modelID)
		grpcErr := status.Error(codes.Unavailable, message)
		return nil, grpcErr
	}

	updatedHyperparameters := existingHyperparameters
	if canonicalCheckpoint != "" {
		updatedHyperparameters.CanonicalCheckpoint = canonicalCheckpoint
	}
	for k, v := range hyperparameters {
		updatedHyperparameters.HyperParameters[k] = v
	}
	storedHyperparameters, err := srv.storage.UpdateHyperParameters(ctx, updatedHyperparameters)
	if err != nil {
		log.Printf("ERROR: %v", err)
		message := fmt.Sprintf("Could not store hyperparameters (%v) in storage", updatedHyperparameters)
		grpcErr := status.Error(codes.Unavailable, message)
		return nil, grpcErr
	}

	resp := &api.UpdateHyperParametersResponse{
		ModelId:             storedHyperparameters.ModelId,
		HyperParametersId:   storedHyperparameters.HyperParametersId,
		UpgradeTo:           "",
		CanonicalCheckpoint: storedHyperparameters.CanonicalCheckpoint,
		HyperParameters:     storedHyperparameters.HyperParameters,
	}
	return resp, nil
}

func (srv *server) ListCheckpoints(ctx context.Context, req *api.ListCheckpointsRequest) (*api.ListCheckpointsResponse, error) {
	modelID := req.ModelId
	hyperparametersID := req.HyperParametersId
	marker := req.Marker
	maxItems := int(req.MaxItems)
	log.Printf("ListCheckpoints request - ModelId: %s, HyperParametersId: %s, Marker: %s, MaxItems: %d", modelID, hyperparametersID, marker, maxItems)
	checkpointIDs, err := srv.storage.ListCheckpoints(ctx, modelID, hyperparametersID, marker, maxItems)
	if err != nil {
		log.Printf("ERROR: %v", err)
		message := fmt.Sprintf("Could not list checkpoints for model (%s) and hyperparameters (%s) in storage", modelID, hyperparametersID)
		grpcErr := status.Error(codes.Unavailable, message)
		return nil, grpcErr
	}
	resp := &api.ListCheckpointsResponse{
		CheckpointIds: checkpointIDs,
	}
	return resp, nil
}

func (srv *server) CreateCheckpoint(ctx context.Context, req *api.CreateCheckpointRequest) (*api.CreateCheckpointResponse, error) {
	modelID := req.ModelId
	hyperparametersID := req.HyperParametersId
	checkpointID := req.CheckpointId
	link := req.Link
	log.Printf("CreateCheckpoint request - ModelId: %s, HyperParametersId: %s, CheckpointId: %s, Link: %s", modelID, hyperparametersID, checkpointID, link)
	storageCheckpoint := storage.Checkpoint{
		ModelId:           modelID,
		HyperParametersId: hyperparametersID,
		CheckpointId:      checkpointID,
		Link:              link,
	}
	err := srv.storage.AddCheckpoint(ctx, storageCheckpoint)
	if err != nil {
		log.Printf("ERROR: %v", err)
		message := fmt.Sprintf("Could not store checkpoint (%v) in storage", storageCheckpoint)
		grpcErr := status.Error(codes.Unavailable, message)
		return nil, grpcErr
	}
	resourcePath := getCheckpointResourcePath(modelID, hyperparametersID, checkpointID)
	resp := &api.CreateCheckpointResponse{
		ResourcePath: resourcePath,
	}
	return resp, nil
}

func (srv *server) GetCheckpoint(ctx context.Context, req *api.GetCheckpointRequest) (*api.GetCheckpointResponse, error) {
	modelID := req.ModelId
	hyperparametersID := req.HyperParametersId
	checkpointID := req.CheckpointId
	log.Printf("GetCheckpoint request - ModelId: %s, HyperParametersId: %s, CheckpointId: %s", modelID, hyperparametersID, checkpointID)
	storedCheckpoint, err := srv.storage.GetCheckpoint(ctx, modelID, hyperparametersID, checkpointID)
	if err != nil {
		log.Printf("ERROR: %v", err)
		message := fmt.Sprintf("Could not get checkpoint (%s) of hyperparameters (%s) for model (%s) from storage", checkpointID, hyperparametersID, modelID)
		grpcErr := status.Error(codes.Unavailable, message)
		return nil, grpcErr
	}
	resourcePath := getCheckpointResourcePath(modelID, hyperparametersID, checkpointID)
	resp := &api.GetCheckpointResponse{
		ResourcePath: resourcePath,
		Link:         storedCheckpoint.Link,
		Info:         storedCheckpoint.Info,
	}
	return resp, nil
}

func getCheckpointResourcePath(modelID, hyperParametersID, checkpointID string) string {
	resourcePath := fmt.Sprintf("/models/%s/hyperparameters/%s/checkpoints/%s", modelID, hyperParametersID, checkpointID)
	return resourcePath
}
